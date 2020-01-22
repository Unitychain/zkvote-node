package protocol

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"

	proto "github.com/gogo/protobuf/proto"
	uuid "github.com/google/uuid"
	"github.com/unitychain/zkvote-node/zkvote/model/context"
	"github.com/unitychain/zkvote-node/zkvote/model/identity"
	pb "github.com/unitychain/zkvote-node/zkvote/model/pb"
	"github.com/unitychain/zkvote-node/zkvote/model/subject"
	"github.com/unitychain/zkvote-node/zkvote/service/utils"
)

// pattern: /protocol-name/request-or-response-message/version
const subjectRequest = "/subject/req/0.0.1"
const subjectResponse = "/subject/res/0.0.1"

// SubjectProtocol type
type SubjectProtocol struct {
	channel  map[peer.ID]chan<- []string
	context  *context.Context
	requests map[string]*pb.SubjectRequest // used to access request data from response handlers
}

// NewSubjectProtocol ...
func NewSubjectProtocol(context *context.Context) Protocol {
	sp := &SubjectProtocol{
		context:  context,
		requests: make(map[string]*pb.SubjectRequest),
	}
	sp.channel = make(map[peer.ID]chan<- []string)
	sp.context.Host.SetStreamHandler(subjectRequest, sp.onRequest)
	sp.context.Host.SetStreamHandler(subjectResponse, sp.onResponse)
	return sp
}

// remote peer requests handler
func (sp *SubjectProtocol) onRequest(s network.Stream) {

	// get request data
	data := &pb.SubjectRequest{}
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

	utils.LogInfof("Received subject request from %s. Message: %s", s.Conn().RemotePeer(), data.Message)

	// generate response message
	// utils.LogInfof("Sending subject response to %s. Message id: %s...", s.Conn().RemotePeer(), data.Metadata.Id)

	// List created subjects
	subjects := make([]*pb.Subject, 0)
	for _, s := range sp.context.Cache.GetCreatedSubjects() {
		identity := s.GetProposer()
		subject := &pb.Subject{Title: s.GetTitle(), Description: s.GetDescription(), Proposer: identity.String()}
		subjects = append(subjects, subject)
	}
	resp := &pb.SubjectResponse{Metadata: NewMetadata(sp.context.Host, data.Metadata.Id, false),
		Message: fmt.Sprintf("Subject response from %s", sp.context.Host.ID()), Subjects: subjects}

	// send the response
	ok := SendProtoMessage(sp.context.Host, s.Conn().RemotePeer(), subjectResponse, resp)
	if ok {
		utils.LogInfof("Subject response(%v) to %s sent.", subjects, s.Conn().RemotePeer().String())
	}
}

// remote ping response handler
func (sp *SubjectProtocol) onResponse(s network.Stream) {
	// results := make([]*subject.Subject, 0)

	data := &pb.SubjectResponse{}
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

	// Store all topics]
	var results []string
	for _, sub := range data.Subjects {
		identity := identity.NewIdentity(sub.Proposer)
		subject := subject.NewSubject(sub.Title, sub.Description, identity)
		// subjectMap := sp.context.Cache.GetCollectedSubjects()
		// subjectMap[*subject.HashHex()] = subject

		b, err := json.Marshal(subject)
		if err != nil {
			utils.LogWarningf("Marshal failed, %v", err)
		}
		results = append(results, string(b))
	}
	sp.channel[s.Conn().RemotePeer()] <- results

	// locate request data and remove it if found
	_, ok := sp.requests[data.Metadata.Id]
	if ok {
		// remove request from map as we have processed it here
		delete(sp.requests, data.Metadata.Id)
	} else {
		utils.LogWarning("Failed to locate request data boject for response")
		return
	}
	utils.LogInfof("Received subject response from %s. Message id:%s. Message: %s.", s.Conn().RemotePeer(), data.Metadata.Id, data.Message)

}

// SubmitRequest ...
// TODO: use callback instead of channel
func (sp *SubjectProtocol) SubmitRequest(peerID peer.ID, subjectHash *subject.Hash, ch chan<- []string) bool {
	utils.LogInfof("Sending subject request to: %s....", peerID)
	_ = subjectHash

	// create message data
	req := &pb.SubjectRequest{Metadata: NewMetadata(sp.context.Host, uuid.New().String(), false),
		Message: fmt.Sprintf("Subject request from %s", sp.context.Host.ID())}

	ok := SendProtoMessage(sp.context.Host, peerID, subjectRequest, req)
	if !ok {
		return false
	}

	sp.channel[peerID] = ch
	// store ref request so response handler has access to it
	sp.requests[req.Metadata.Id] = req
	// utils.LogInfof("Subject request to: %s was sent. Message Id: %s, Message: %s", peerID, req.Metadata.Id, req.Message)
	return true
}
