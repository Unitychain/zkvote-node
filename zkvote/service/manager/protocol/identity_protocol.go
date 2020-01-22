package protocol

import (
	"fmt"
	"io/ioutil"

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
const identityRequest = "/identity/req/0.0.1"
const identityResponse = "/identity/res/0.0.1"

// IdentityProtocol type
type IdentityProtocol struct {
	channels map[peer.ID]map[subject.HashHex]chan<- []string
	context  *context.Context
	requests map[string]*pb.IdentityRequest // used to access request data from response handlers
}

// NewIdentityProtocol ...
func NewIdentityProtocol(context *context.Context) Protocol {
	sp := &IdentityProtocol{
		context:  context,
		requests: make(map[string]*pb.IdentityRequest),
	}
	sp.channels = make(map[peer.ID]map[subject.HashHex]chan<- []string)
	sp.context.Host.SetStreamHandler(identityRequest, sp.onRequest)
	sp.context.Host.SetStreamHandler(identityResponse, sp.onResponse)
	return sp
}

// remote peer requests handler
func (sp *IdentityProtocol) onRequest(s network.Stream) {

	// get request data
	data := &pb.IdentityRequest{}
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

	utils.LogInfof("Received identity request from %s. Message: %s", s.Conn().RemotePeer(), data.Message)

	// generate response message
	utils.LogInfof("Sending identity response to %s. Message id: %s...", s.Conn().RemotePeer(), data.Metadata.Id)

	// List identity index
	subjectHash := subject.Hash(data.SubjectHash)
	var identitySet []string
	set := sp.context.Cache.GetIdentitySet(subjectHash.Hex())
	// set, err := sp.manager.GetIdentitySet(&subjectHash)
	for k := range set {
		identitySet = append(identitySet, k.String())
	}
	resp := &pb.IdentityResponse{Metadata: NewMetadata(sp.context.Host, data.Metadata.Id, false),
		Message: fmt.Sprintf("Identity response from %s", sp.context.Host.ID()), SubjectHash: subjectHash.Byte(), IdentitySet: identitySet}

	// send the response
	ok := SendProtoMessage(sp.context.Host, s.Conn().RemotePeer(), identityResponse, resp)

	if ok {
		utils.LogInfof("Identity response to %s sent.", s.Conn().RemotePeer().String())
	}
}

// remote ping response handler
func (sp *IdentityProtocol) onResponse(s network.Stream) {

	data := &pb.IdentityResponse{}
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

	// Store all identityHash
	subjectHash := subject.Hash(data.SubjectHash)

	ch := sp.channels[s.Conn().RemotePeer()][subjectHash.Hex()]
	ch <- data.IdentitySet

	// locate request data and remove it if found
	_, ok := sp.requests[data.Metadata.Id]
	if ok {
		// remove request from map as we have processed it here
		delete(sp.requests, data.Metadata.Id)
	} else {
		utils.LogWarningf("Failed to locate request data boject for response")
		return
	}

	utils.LogInfof("Received identity response from %s. Message id:%s. Message: %s.", s.Conn().RemotePeer(), data.Metadata.Id, data.Message)
}

// SubmitRequest ...
// TODO: use callback instead of channel
func (sp *IdentityProtocol) SubmitRequest(peerID peer.ID, subjectHash *subject.Hash, ch chan<- []string) bool {
	utils.LogInfof("Sending identity request to: %s....", peerID)

	// create message data
	req := &pb.IdentityRequest{Metadata: NewMetadata(sp.context.Host, uuid.New().String(), false),
		Message: fmt.Sprintf("Identity request from %s", sp.context.Host.ID()), SubjectHash: subjectHash.Byte()}

	ok := SendProtoMessage(sp.context.Host, peerID, identityRequest, req)
	if !ok {
		return false
	}

	chPeer := sp.channels[peerID]
	if chPeer == nil {
		sp.channels[peerID] = make(map[subject.HashHex]chan<- []string)
	}
	sp.channels[peerID][subjectHash.Hex()] = ch
	// store ref request so response handler has access to it
	sp.requests[req.Metadata.Id] = req
	utils.LogInfof("Identity request to: %s was sent. Message Id: %s, Message: %s", peerID, req.Metadata.Id, req.Message)
	return true
}
