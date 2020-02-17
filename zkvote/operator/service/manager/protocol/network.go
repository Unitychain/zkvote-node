package protocol

import (
	"context"
	"log"
	"time"

	ggio "github.com/gogo/protobuf/io"
	proto "github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/helpers"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	pb "github.com/unitychain/zkvote-node/zkvote/model/pb"
	"github.com/unitychain/zkvote-node/zkvote/operator/service/utils"
)

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
