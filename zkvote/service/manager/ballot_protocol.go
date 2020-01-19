package manager

import (
	"fmt"
	"io/ioutil"
	"log"

	proto "github.com/gogo/protobuf/proto"
	uuid "github.com/google/uuid"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	pb "github.com/unitychain/zkvote-node/zkvote/model/pb"
	"github.com/unitychain/zkvote-node/zkvote/model/subject"
	"github.com/unitychain/zkvote-node/zkvote/service/utils"
)

// pattern: /protocol-name/request-or-response-message/version
const ballotRequest = "/ballot/req/0.0.1"
const ballotResponse = "/ballot/res/0.0.1"

// BallotProtocol type
type BallotProtocol struct {
	manager  *Manager
	requests map[string]*pb.BallotRequest // used to access request data from response handlers
}

// NewBallotProtocol ...
func NewBallotProtocol(m *Manager) *BallotProtocol {
	sp := &BallotProtocol{
		manager:  m,
		requests: make(map[string]*pb.BallotRequest),
	}
	m.Host.SetStreamHandler(ballotRequest, sp.onBallotRequest)
	m.Host.SetStreamHandler(ballotResponse, sp.onBallotResponse)
	return sp
}

// remote peer requests handler
func (sp *BallotProtocol) onBallotRequest(s network.Stream) {

	// get request data
	data := &pb.BallotRequest{}
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

	log.Printf("Received ballot request from %s. Message: %s", s.Conn().RemotePeer(), data.Message)

	// valid := p.node.authenticateMessage(data, data.Metadata)

	// if !valid {
	// 	log.Println("Failed to authenticate message")
	// 	return
	// }

	// generate response message
	log.Printf("Sending ballot response to %s. Message id: %s...", s.Conn().RemotePeer(), data.Metadata.Id)

	// List ballot index
	subjectHash := subject.Hash(data.SubjectHash)
	var ballotSet []string
	set, err := sp.manager.GetBallotSet(&subjectHash)
	for _, h := range set {
		s, _ := h.JSON()
		ballotSet = append(ballotSet, s)
	}
	resp := &pb.BallotResponse{Metadata: NewMetadata(sp.manager.Host, data.Metadata.Id, false),
		Message: fmt.Sprintf("Ballot response from %s", sp.manager.Host.ID()), SubjectHash: subjectHash.Byte(), BallotSet: ballotSet}

	// sign the data
	// signature, err := p.node.signProtoMessage(resp)
	// if err != nil {
	// 	log.Println("failed to sign response")
	// 	return
	// }

	// add the signature to the message
	// resp.Metadata.Sign = signature

	// send the response
	ok := SendProtoMessage(sp.manager.Host, s.Conn().RemotePeer(), ballotResponse, resp)

	if ok {
		log.Printf("Ballot response to %s sent.", s.Conn().RemotePeer().String())
	}
}

// remote ping response handler
func (sp *BallotProtocol) onBallotResponse(s network.Stream) {
	utils.LogDebug("onBallotResponse")
	data := &pb.BallotResponse{}
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

	// Store all ballotHash
	subjectHash := subject.Hash(data.SubjectHash)
	err = sp.manager.InsertBallots(subjectHash.Hex().String(), data.BallotSet)
	if err != nil {
		utils.LogErrorf("Failed to insert ballotSet, %v", err.Error())
		return
	}

	// locate request data and remove it if found
	_, ok := sp.requests[data.Metadata.Id]
	if ok {
		// remove request from map as we have processed it here
		delete(sp.requests, data.Metadata.Id)
	} else {
		log.Println("Failed to locate request data boject for response")
		return
	}

	log.Printf("Received ballot response from %s. Message id:%s. Message: %s.", s.Conn().RemotePeer(), data.Metadata.Id, data.Message)
}

// GetBallotIndexFromPeer ...
func (sp *BallotProtocol) GetBallotIndexFromPeer(peerID peer.ID, subjectHash *subject.Hash) bool {
	log.Printf("Sending ballot request to: %s....", peerID)

	// create message data
	req := &pb.BallotRequest{Metadata: NewMetadata(sp.manager.Host, uuid.New().String(), false),
		Message: fmt.Sprintf("Ballot request from %s", sp.manager.Host.ID()), SubjectHash: subjectHash.Byte()}

	// sign the data
	// signature, err := p.node.signProtoMessage(req)
	// if err != nil {
	// 	log.Println("failed to sign pb data")
	// 	return false
	// }

	// add the signature to the message
	// req.Metadata.Sign = signature

	ok := SendProtoMessage(sp.manager.Host, peerID, ballotRequest, req)
	if !ok {
		return false
	}

	// store ref request so response handler has access to it
	sp.requests[req.Metadata.Id] = req
	log.Printf("Ballot request to: %s was sent. Message Id: %s, Message: %s", peerID, req.Metadata.Id, req.Message)
	return true
}
