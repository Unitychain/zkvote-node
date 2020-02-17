package protocol

import "github.com/unitychain/zkvote-node/zkvote/operator/model/context"

type ProtocolType int

const (
	BallotProtocolType ProtocolType = 1 << iota
	IdentityProtocolType
	SubjectProtocolType
)

// NewProtocol .
func NewProtocol(t ProtocolType, context *context.Context) Protocol {
	switch t {
	case BallotProtocolType:
		return NewBallotProtocol(context)
	case IdentityProtocolType:
		return NewIdentityProtocol(context)
	case SubjectProtocolType:
		return NewSubjectProtocol(context)
	}
	return nil
}
