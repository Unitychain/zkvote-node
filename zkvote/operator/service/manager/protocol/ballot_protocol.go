package protocol

import (
	"fmt"
	"io/ioutil"

	proto "github.com/gogo/protobuf/proto"
	uuid "github.com/google/uuid"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/unitychain/zkvote-node/zkvote/common/utils"
	"github.com/unitychain/zkvote-node/zkvote/model/context"
	pb "github.com/unitychain/zkvote-node/zkvote/model/pb"
	"github.com/unitychain/zkvote-node/zkvote/model/subject"
)

// pattern: /protocol-name/request-or-response-message/version
const ballotRequest = "/ballot/req/0.0.1"
const ballotResponse = "/ballot/res/0.0.1"

// BallotProtocol type
type BallotProtocol struct {
	channels map[peer.ID]map[subject.HashHex]chan<- []string
	context  *context.Context
	requests map[string]*pb.BallotRequest // used to access request data from response handlers
}

// NewBallotProtocol ...
func NewBallotProtocol(context *context.Context) Protocol {
	sp := &BallotProtocol{
		context:  context,
		requests: make(map[string]*pb.BallotRequest),
	}
	sp.channels = make(map[peer.ID]map[subject.HashHex]chan<- []string)
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
		utils.LogErrorf("%v", err)
		return
	}
	s.Close()

	// unmarshal it
	proto.Unmarshal(buf, data)
	if err != nil {
		utils.LogErrorf("%v", err)
		return
	}

	utils.LogInfof("Received ballot request from %s. Message: %s", s.Conn().RemotePeer(), data.Message)

	// generate response message
	// utils.LogInfof("Sending ballot response to %s. Message id: %s...", s.Conn().RemotePeer(), data.Metadata.Id)

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
		utils.LogInfof("Ballot response(%v) to %s sent.", ballotSet, s.Conn().RemotePeer().String())
	}
}

// remote ping response handler
func (sp *BallotProtocol) onResponse(s network.Stream) {

	data := &pb.BallotResponse{}
	buf, err := ioutil.ReadAll(s)
	if err != nil {
		s.Reset()
		utils.LogErrorf("%v", err)
		return
	}
	s.Close()

	// unmarshal it
	proto.Unmarshal(buf, data)
	if err != nil {
		utils.LogErrorf("%v", err)
		return
	}

	defer func() {
		err := recover()
		if err != nil {
			utils.LogWarningf("panic: %v", err)
		}
	}()
	// utils.LogDebugf("response, ballot %v", data.BallotSet)
	subjectHash := subject.Hash(data.SubjectHash)
	ch := sp.channels[s.Conn().RemotePeer()][subjectHash.Hex()]
	ch <- data.BallotSet

	// locate request data and remove it if found
	_, ok := sp.requests[data.Metadata.Id]
	if ok {
		// remove request from map as we have processed it here
		delete(sp.requests, data.Metadata.Id)
	} else {
		utils.LogWarning("Failed to locate request data boject for response")
		return
	}

	utils.LogInfof("Received ballot response from %s. Message id:%s. Message: %s.", s.Conn().RemotePeer(), data.Metadata.Id, data.Message)
}

// SubmitRequest ...
// TODO: use callback instead of channel
func (sp *BallotProtocol) SubmitRequest(peerID peer.ID, subjectHash *subject.Hash, ch chan<- []string) bool {
	utils.LogInfof("Sending ballot request to: %s....", peerID)

	// create message data
	req := &pb.BallotRequest{Metadata: NewMetadata(sp.context.Host, uuid.New().String(), false),
		Message: fmt.Sprintf("Ballot request from %s", sp.context.Host.ID()), SubjectHash: subjectHash.Byte()}

	ok := SendProtoMessage(sp.context.Host, peerID, ballotRequest, req)
	if !ok {
		return false
	}

	// store ref request so response handler has access to it
	sp.requests[req.Metadata.Id] = req

	chPeer := sp.channels[peerID]
	if chPeer == nil {
		sp.channels[peerID] = make(map[subject.HashHex]chan<- []string)
	}
	sp.channels[peerID][subjectHash.Hex()] = ch
	// utils.LogInfof("Ballot request to: %s was sent. Message Id: %s, Message: %s", peerID, req.Metadata.Id, req.Message)
	return true
}
