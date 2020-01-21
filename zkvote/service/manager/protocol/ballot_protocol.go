package protocol

import (
	"fmt"
	"io/ioutil"
	"log"

	proto "github.com/gogo/protobuf/proto"
	uuid "github.com/google/uuid"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/unitychain/zkvote-node/zkvote/model/context"
	pb "github.com/unitychain/zkvote-node/zkvote/model/pb"
	"github.com/unitychain/zkvote-node/zkvote/model/subject"
	"github.com/unitychain/zkvote-node/zkvote/service/utils"
)

// pattern: /protocol-name/request-or-response-message/version
const ballotRequest = "/ballot/req/0.0.1"
const ballotResponse = "/ballot/res/0.0.1"

// BallotProtocol type
type BallotProtocol struct {
	channels map[subject.HashHex]chan<- []string
	context  context.Context
	requests map[string]*pb.BallotRequest // used to access request data from response handlers
}

// NewBallotProtocol ...
func NewBallotProtocol(context context.Context) Protocol {
	sp := BallotProtocol{
		context:  context,
		requests: make(map[string]*pb.BallotRequest),
	}
	sp.context.Host.SetStreamHandler(ballotRequest, sp.onRequest)
	sp.context.Host.SetStreamHandler(ballotResponse, sp.onResponse)
	return sp
}

// remote peer requests handler
func (sp *BallotProtocol) onRequest(s network.Stream) {

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

	// generate response message
	log.Printf("Sending ballot response to %s. Message id: %s...", s.Conn().RemotePeer(), data.Metadata.Id)

	// List ballot index
	subjectHash := subject.Hash(data.SubjectHash)
	var ballotSet []string
	set := sp.context.Cache.GetBallotSet(subjectHash.Hex())
	// set, err := sp.manager.GetBallotSet(&subjectHash)
	for _, h := range set {
		s, _ := h.JSON()
		ballotSet = append(ballotSet, s)
	}
	resp := &pb.BallotResponse{Metadata: NewMetadata(sp.context.Host, data.Metadata.Id, false),
		Message: fmt.Sprintf("Ballot response from %s", sp.context.Host.ID()), SubjectHash: subjectHash.Byte(), BallotSet: ballotSet}

	// send the response
	ok := SendProtoMessage(sp.context.Host, s.Conn().RemotePeer(), ballotResponse, resp)

	if ok {
		log.Printf("Ballot response to %s sent.", s.Conn().RemotePeer().String())
	}
}

// remote ping response handler
func (sp *BallotProtocol) onResponse(s network.Stream) {
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

	subjectHash := subject.Hash(data.SubjectHash)
	ch := sp.channels[subjectHash.Hex()]
	ch <- data.BallotSet

	// err = sp.manager.InsertBallots(subjectHash.Hex().String(), data.BallotSet)
	// if err != nil {
	// 	utils.LogErrorf("Failed to insert ballotSet, %v", err.Error())
	// 	return
	// }

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

// SubmitRequest ...
func (sp *BallotProtocol) SubmitRequest(peerID peer.ID, subjectHash *subject.Hash, ch chan<- []string) bool {
	log.Printf("Sending ballot request to: %s....", peerID)

	// create message data
	req := &pb.BallotRequest{Metadata: NewMetadata(sp.context.Host, uuid.New().String(), false),
		Message: fmt.Sprintf("Ballot request from %s", sp.context.Host.ID()), SubjectHash: subjectHash.Byte()}

	ok := SendProtoMessage(sp.context.Host, peerID, ballotRequest, req)
	if !ok {
		return false
	}

	// store ref request so response handler has access to it
	sp.requests[req.Metadata.Id] = req

	sp.channels[subjectHash.Hex()] = ch
	log.Printf("Ballot request to: %s was sent. Message Id: %s, Message: %s", peerID, req.Metadata.Id, req.Message)
	return true
}
