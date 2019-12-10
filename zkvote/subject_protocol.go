package zkvote

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"

	proto "github.com/gogo/protobuf/proto"
	uuid "github.com/google/uuid"
	pb "github.com/unitychain/zkvote-node/zkvote/pb"
	"github.com/unitychain/zkvote-node/zkvote/subject"
)

// pattern: /protocol-name/request-or-response-message/version
const subjectRequest = "/subject/req/0.0.1"
const subjectResponse = "/subject/res/0.0.1"

// SubjectProtocol type
type SubjectProtocol struct {
	node     *Node
	requests map[string]*pb.SubjectRequest // used to access request data from response handlers
	done     chan bool                     // only for demo purposes to stop main from terminating
}

// NewSubjectProtocol ...
func NewSubjectProtocol(node *Node, done chan bool) *SubjectProtocol {
	sp := &SubjectProtocol{
		node:     node,
		requests: make(map[string]*pb.SubjectRequest),
		done:     done,
	}
	node.SetStreamHandler(subjectRequest, sp.onSubjectRequest)
	node.SetStreamHandler(subjectResponse, sp.onSubjectResponse)
	return sp
}

// remote peer requests handler
func (sp *SubjectProtocol) onSubjectRequest(s network.Stream) {

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

	// List subscribed topics
	subjects := make([]*pb.Subject, 0)
	for _, t := range sp.node.pubsub.GetTopics() {
		s := &pb.Subject{Title: t, Description: "foobar"}
		subjects = append(subjects, s)
	}

	resp := &pb.SubjectResponse{Metadata: sp.node.NewMetadata(data.Metadata.Id, false),
		Message: fmt.Sprintf("Subject response from %s", sp.node.ID()), Subjects: subjects}

	// sign the data
	// signature, err := p.node.signProtoMessage(resp)
	// if err != nil {
	// 	log.Println("failed to sign response")
	// 	return
	// }

	// add the signature to the message
	// resp.Metadata.Sign = signature

	// send the response
	ok := sp.node.sendProtoMessage(s.Conn().RemotePeer(), subjectResponse, resp)

	if ok {
		log.Printf("Subject response to %s sent.", s.Conn().RemotePeer().String())
	}
}

// remote ping response handler
func (sp *SubjectProtocol) onSubjectResponse(s network.Stream) {
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

	// Store all topics
	for _, sub := range data.Subjects {
		subject := subject.NewSubject(sub.Title, sub.Description)
		subjectMap := sp.node.collectedSubjects
		subjectMap[subject.Hash().Hex()] = subject
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
	sp.done <- true
}

// GetCreatedSubjects ...
func (sp *SubjectProtocol) GetCreatedSubjects(peerID peer.ID) bool {
	log.Printf("Sending subject request to: %s....", peerID)

	// create message data
	req := &pb.SubjectRequest{Metadata: sp.node.NewMetadata(uuid.New().String(), false),
		Message: fmt.Sprintf("Subject request from %s", sp.node.ID())}

	// sign the data
	// signature, err := p.node.signProtoMessage(req)
	// if err != nil {
	// 	log.Println("failed to sign pb data")
	// 	return false
	// }

	// add the signature to the message
	// req.Metadata.Sign = signature

	ok := sp.node.sendProtoMessage(peerID, subjectRequest, req)
	if !ok {
		return false
	}

	// store ref request so response handler has access to it
	sp.requests[req.Metadata.Id] = req
	log.Printf("Subject request to: %s was sent. Message Id: %s, Message: %s", peerID, req.Metadata.Id, req.Message)
	return true
}
