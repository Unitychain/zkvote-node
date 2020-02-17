package protocol

import (
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/unitychain/zkvote-node/zkvote/operator/model/subject"
)

type Protocol interface {
	onRequest(s network.Stream)
	onResponse(s network.Stream)

	SubmitRequest(peerID peer.ID, subjectHash *subject.Hash, ch chan<- []string) bool
}
