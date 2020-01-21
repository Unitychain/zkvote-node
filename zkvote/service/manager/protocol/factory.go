package protocol

type ProtocolType int

const (
	BallotProtocolType ProtocolType = 1 << iota
	IdentityProtocolType
	SubjectProtocolType
)

func NewProtocol(t ProtocolType) Protocol {
	switch t {
	case BallotProtocolType:
		return NewBallotProtocol()
	case IdentityProtocolType:
		return NewIdentityProtocol()
	case SubjectProtocolType:
		return NewSubjectProtocol()
	}
	return nil
}
