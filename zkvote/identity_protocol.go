package zkvote

import (
	"fmt"
	"io/ioutil"
	"log"

	proto "github.com/gogo/protobuf/proto"
	uuid "github.com/google/uuid"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	pb "github.com/unitychain/zkvote-node/zkvote/pb"
	"github.com/unitychain/zkvote-node/zkvote/subject"
)

// pattern: /protocol-name/request-or-response-message/version
const identityRequest = "/identity/req/0.0.1"
const identityResponse = "/identity/res/0.0.1"

// IdentityProtocol type
type IdentityProtocol struct {
	node     *Node
	requests map[string]*pb.IdentityRequest // used to access request data from response handlers
	done     chan bool                      // only for demo purposes to stop main from terminating
}

// NewIdentityProtocol ...
func NewIdentityProtocol(node *Node, done chan bool) *IdentityProtocol {
	sp := &IdentityProtocol{node: node, requests: make(map[string]*pb.IdentityRequest), done: done}
	node.SetStreamHandler(identityRequest, sp.onIdentityRequest)
	node.SetStreamHandler(identityResponse, sp.onIdentityResponse)
	return sp
}

// remote peer requests handler
func (sp *IdentityProtocol) onIdentityRequest(s network.Stream) {

	// get request data
	data := &pb.IdentityRequest{}
	buf, err := ioutil.ReadAll(s)
	if err != nil {
		s.Reset()
		log.Println(err)
		return
	}
	s.Close()

	// unmarshal it
	proto.Unmarshal(buf, data)
	if err != nil {
		log.Println(err)
		return
	}

	log.Printf("Received identity request from %s. Message: %s", s.Conn().RemotePeer(), data.Message)

	// valid := p.node.authenticateMessage(data, data.MessageData)

	// if !valid {
	// 	log.Println("Failed to authenticate message")
	// 	return
	// }

	// generate response message
	log.Printf("Sending identity response to %s. Message id: %s...", s.Conn().RemotePeer(), data.MessageData.Id)

	// List identity index
	subjectHash := subject.Hash(data.SubjectHash)
	var identityHashes [][]byte
	for _, h := range sp.node.GetIdentityHashes(&subjectHash) {
		identityHashes = append(identityHashes, h.hash)
	}
	resp := &pb.IdentityResponse{MessageData: sp.node.NewMessageData(data.MessageData.Id, false),
		Message: fmt.Sprintf("Identity response from %s", sp.node.ID()), SubjectHash: subjectHash.Byte(), IdentityHashes: identityHashes}

	// sign the data
	// signature, err := p.node.signProtoMessage(resp)
	// if err != nil {
	// 	log.Println("failed to sign response")
	// 	return
	// }

	// add the signature to the message
	// resp.MessageData.Sign = signature

	// send the response
	ok := sp.node.sendProtoMessage(s.Conn().RemotePeer(), identityResponse, resp)

	if ok {
		log.Printf("Identity response to %s sent.", s.Conn().RemotePeer().String())
	}
}

// remote ping response handler
func (sp *IdentityProtocol) onIdentityResponse(s network.Stream) {
	data := &pb.IdentityResponse{}
	buf, err := ioutil.ReadAll(s)
	if err != nil {
		s.Reset()
		log.Println(err)
		return
	}
	s.Close()

	// unmarshal it
	proto.Unmarshal(buf, data)
	if err != nil {
		log.Println(err)
		return
	}

	// valid := p.node.authenticateMessage(data, data.MessageData)

	// if !valid {
	// 	log.Println("Failed to authenticate message")
	// 	return
	// }

	// Store all identityHash

	subjectHash := subject.Hash(data.SubjectHash)
	identityHashSet, ok := sp.node.identityIndex.Index[subjectHash.Hex()]
	if !ok {
		identityHashSet = &IdentityHashSet{set: make(map[IdentityHashHex]string)}
	}
	for _, idhash := range data.IdentityHashes {
		identityHash := &IdentityHash{hash: idhash}
		identityHashSet.set[identityHash.hex()] = ""
	}
	fmt.Println("***", subjectHash.Hex())
	sp.node.identityIndex.Index[subjectHash.Hex()] = identityHashSet

	// locate request data and remove it if found
	_, ok = sp.requests[data.MessageData.Id]
	if ok {
		// remove request from map as we have processed it here
		delete(sp.requests, data.MessageData.Id)
	} else {
		log.Println("Failed to locate request data boject for response")
		return
	}

	log.Printf("Received identity response from %s. Message id:%s. Message: %s.", s.Conn().RemotePeer(), data.MessageData.Id, data.Message)
	sp.done <- true
}

// GetIdentityIndexFromPeer ...
func (sp *IdentityProtocol) GetIdentityIndexFromPeer(peerID peer.ID, subjectHash *subject.Hash) bool {
	log.Printf("Sending identity request to: %s....", peerID)

	// create message data
	req := &pb.IdentityRequest{MessageData: sp.node.NewMessageData(uuid.New().String(), false),
		Message: fmt.Sprintf("Identity request from %s", sp.node.ID()), SubjectHash: subjectHash.Byte()}

	// sign the data
	// signature, err := p.node.signProtoMessage(req)
	// if err != nil {
	// 	log.Println("failed to sign pb data")
	// 	return false
	// }

	// add the signature to the message
	// req.MessageData.Sign = signature

	ok := sp.node.sendProtoMessage(peerID, identityRequest, req)
	if !ok {
		return false
	}

	// store ref request so response handler has access to it
	sp.requests[req.MessageData.Id] = req
	log.Printf("Identity request to: %s was sent. Message Id: %s, Message: %s", peerID, req.MessageData.Id, req.Message)
	return true
}
