package voter

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	ggio "github.com/gogo/protobuf/io"
	proto "github.com/gogo/protobuf/proto"
	uuid "github.com/google/uuid"
	"github.com/libp2p/go-libp2p-core/helpers"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	id "github.com/unitychain/zkvote-node/zkvote/model/identity"
	pb "github.com/unitychain/zkvote-node/zkvote/model/pb"
	"github.com/unitychain/zkvote-node/zkvote/model/subject"
	"github.com/unitychain/zkvote-node/zkvote/service/utils"
)

// pattern: /protocol-name/request-or-response-message/version
const identityRequest = "/identity/req/0.0.1"
const identityResponse = "/identity/res/0.0.1"

// IdentityProtocol type
type IdentityProtocol struct {
	manager  *Manager
	requests map[string]*pb.IdentityRequest // used to access request data from response handlers
	done     chan bool                      // only for demo purposes to stop main from terminating
}

// NewIdentityProtocol ...
func NewIdentityProtocol(m *Manager, done chan bool) *IdentityProtocol {
	sp := &IdentityProtocol{
		manager:  m,
		requests: make(map[string]*pb.IdentityRequest),
		done:     done,
	}
	m.Host.SetStreamHandler(identityRequest, sp.onIdentityRequest)
	m.Host.SetStreamHandler(identityResponse, sp.onIdentityResponse)
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

	// valid := p.node.authenticateMessage(data, data.Metadata)

	// if !valid {
	// 	log.Println("Failed to authenticate message")
	// 	return
	// }

	// generate response message
	log.Printf("Sending identity response to %s. Message id: %s...", s.Conn().RemotePeer(), data.Metadata.Id)

	// List identity index
	subjectHash := subject.Hash(data.SubjectHash)
	var identityHashes [][]byte
	set, err := sp.manager.GetIdentityHashes(&subjectHash)
	for _, h := range set {
		identityHashes = append(identityHashes, h.Byte())
	}
	resp := &pb.IdentityResponse{Metadata: NewMetadata(sp.manager.Host, data.Metadata.Id, false),
		Message: fmt.Sprintf("Identity response from %s", sp.manager.Host.ID()), SubjectHash: subjectHash.Byte(), IdentityHashes: identityHashes}

	// sign the data
	// signature, err := p.node.signProtoMessage(resp)
	// if err != nil {
	// 	log.Println("failed to sign response")
	// 	return
	// }

	// add the signature to the message
	// resp.Metadata.Sign = signature

	// send the response
	ok := SendProtoMessage(sp.manager.Host, s.Conn().RemotePeer(), identityResponse, resp)

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

	// valid := p.node.authenticateMessage(data, data.Metadata)

	// if !valid {
	// 	log.Println("Failed to authenticate message")
	// 	return
	// }

	// Store all identityHash
	subjectHash := subject.Hash(data.SubjectHash)
	for _, idhash := range data.IdentityHashes {
		sp.manager.InsertIdentity(&subjectHash, id.Hash(idhash))
	}
	fmt.Println("***", subjectHash.Hex())

	// locate request data and remove it if found
	_, ok := sp.requests[data.Metadata.Id]
	if ok {
		// remove request from map as we have processed it here
		delete(sp.requests, data.Metadata.Id)
	} else {
		log.Println("Failed to locate request data boject for response")
		return
	}

	log.Printf("Received identity response from %s. Message id:%s. Message: %s.", s.Conn().RemotePeer(), data.Metadata.Id, data.Message)
	sp.done <- true
}

// GetIdentityIndexFromPeer ...
func (sp *IdentityProtocol) GetIdentityIndexFromPeer(peerID peer.ID, subjectHash *subject.Hash) bool {
	log.Printf("Sending identity request to: %s....", peerID)

	// create message data
	req := &pb.IdentityRequest{Metadata: NewMetadata(sp.manager.Host, uuid.New().String(), false),
		Message: fmt.Sprintf("Identity request from %s", sp.manager.Host.ID()), SubjectHash: subjectHash.Byte()}

	// sign the data
	// signature, err := p.node.signProtoMessage(req)
	// if err != nil {
	// 	log.Println("failed to sign pb data")
	// 	return false
	// }

	// add the signature to the message
	// req.Metadata.Sign = signature

	ok := SendProtoMessage(sp.manager.Host, peerID, identityRequest, req)
	if !ok {
		return false
	}

	// store ref request so response handler has access to it
	sp.requests[req.Metadata.Id] = req
	log.Printf("Identity request to: %s was sent. Message Id: %s, Message: %s", peerID, req.Metadata.Id, req.Message)
	return true
}

// SendProtoMessage helper method - writes a protobuf go data object to a network stream
// data: reference of protobuf go data object to send (not the object itself)
// s: network stream to write the data to
func SendProtoMessage(host host.Host, id peer.ID, p protocol.ID, data proto.Message) bool {
	s, err := host.NewStream(context.Background(), id, p)
	if err != nil {
		log.Println(err)
		return false
	}
	writer := ggio.NewFullWriter(s)
	err = writer.WriteMsg(data)
	if err != nil {
		log.Println(err)
		s.Reset()
		return false
	}
	// FullClose closes the stream and waits for the other side to close their half.
	err = helpers.FullClose(s)
	if err != nil {
		log.Println(err)
		s.Reset()
		return false
	}
	return true
}

// NewMetadata helper method - generate message data shared between all node's p2p protocols
// messageId: unique for requests, copied from request for responses
func NewMetadata(host host.Host, messageID string, gossip bool) *pb.Metadata {
	// Add protobufs bin data for message author public key
	// this is useful for authenticating  messages forwarded by a node authored by another node
	nodePubKey, err := host.Peerstore().PubKey(host.ID()).Bytes()

	if err != nil {
		panic("Failed to get public key for sender from local peer store.")
	}

	return &pb.Metadata{
		ClientVersion: utils.ClientVersion,
		NodeId:        peer.IDB58Encode(host.ID()),
		NodePubKey:    nodePubKey,
		Timestamp:     time.Now().Unix(),
		Id:            messageID,
		Gossip:        gossip}
}
