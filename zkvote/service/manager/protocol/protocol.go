package protocol

import (
	"github.com/libp2p/go-libp2p-core/network"
)

type Protocol interface {
	onRequest(s network.Stream)
	onResponse(s network.Stream)

	// SubmitRequest(peerID peer.ID, subjectHash *subject.Hash, ch chan<- interface{}) bool
}
