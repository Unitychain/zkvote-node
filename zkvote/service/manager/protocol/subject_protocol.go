package protocol

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"

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
	channel  chan<- []string
	context  *context.Context
	requests map[string]*pb.SubjectRequest // used to access request data from response handlers
}

// NewSubjectProtocol ...
func NewSubjectProtocol(context *context.Context) Protocol {
	sp := &SubjectProtocol{
		context:  context,
		requests: make(map[string]*pb.SubjectRequest),
	}
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

	log.Printf("Received subject request from %s. Message: %s", s.Conn().RemotePeer(), data.Message)

	// valid := p.node.authenticateMessage(data, data.Metadata)

	// if !valid {
	// 	log.Println("Failed to authenticate message")
	// 	return
	// }

	// generate response message
	log.Printf("Sending subject response to %s. Message id: %s...", s.Conn().RemotePeer(), data.Metadata.Id)

	// List created subjects
	subjects := make([]*pb.Subject, 0)
	for _, s := range sp.context.Cache.GetCreatedSubjects() {
		identity := s.GetProposer()
		subject := &pb.Subject{Title: s.GetTitle(), Description: s.GetDescription(), Proposer: identity.String()}
		subjects = append(subjects, subject)
	}
	resp := &pb.SubjectResponse{Metadata: NewMetadata(sp.context.Host, data.Metadata.Id, false),
		Message: fmt.Sprintf("Subject response from %s", sp.context.Host.ID()), Subjects: subjects}
	// resp := &pb.SubjectResponse{Metadata: NewMetadata(sp.manager.Host, data.Metadata.Id, false),
	// 	Message: fmt.Sprintf("Subject response from %s", sp.manager.Host.ID()), Subjects: nil}

	// sign the data
	// signature, err := p.node.signProtoMessage(resp)
	// if err != nil {
	// 	log.Println("failed to sign response")
	// 	return
	// }

	// add the signature to the message
	// resp.Metadata.Sign = signature

	// send the response
	ok := SendProtoMessage(sp.context.Host, s.Conn().RemotePeer(), subjectResponse, resp)

	if ok {
		log.Printf("Subject response to %s sent.", s.Conn().RemotePeer().String())
	}
}

// remote ping response handler
func (sp *SubjectProtocol) onResponse(s network.Stream) {
	// results := make([]*subject.Subject, 0)

	data := &pb.SubjectResponse{}
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

	// Store all topics]
	var results []string
	for _, sub := range data.Subjects {
		identity := identity.NewIdentity(sub.Proposer)
		subject := subject.NewSubject(sub.Title, sub.Description, identity)
		subjectMap := sp.context.Cache.GetCollectedSubjects()
		subjectMap[*subject.HashHex()] = subject

		b, err := json.Marshal(subject)
		if err != nil {
			utils.LogWarningf("Marshal failed, %v", err)
		}
		results = append(results, string(b))
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
	log.Printf("Received subject response from %s. Message id:%s. Message: %s.", s.Conn().RemotePeer(), data.Metadata.Id, data.Message)

	// b, err := json.Marshal(results)
	// if err != nil {
	// 	utils.LogWarningf("Marshal failed, %v", err)
	// }
	sp.channel <- results
	// sp.manager.subjectProtocolCh <- results
}

// SubmitRequest ...
func (sp *SubjectProtocol) SubmitRequest(peerID peer.ID, subjectHash *subject.Hash, ch chan<- []string) bool {
	log.Printf("Sending subject request to: %s....", peerID)
	_ = subjectHash

	// create message data
	req := &pb.SubjectRequest{Metadata: NewMetadata(sp.context.Host, uuid.New().String(), false),
		Message: fmt.Sprintf("Subject request from %s", sp.context.Host.ID())}

	// sign the data
	// signature, err := p.node.signProtoMessage(req)
	// if err != nil {
	// 	log.Println("failed to sign pb data")
	// 	return false
	// }

	// add the signature to the message
	// req.Metadata.Sign = signature

	ok := SendProtoMessage(sp.context.Host, peerID, subjectRequest, req)
	if !ok {
		return false
	}

	sp.channel = ch
	// store ref request so response handler has access to it
	sp.requests[req.Metadata.Id] = req
	log.Printf("Subject request to: %s was sent. Message Id: %s, Message: %s", peerID, req.Metadata.Id, req.Message)
	return true
}