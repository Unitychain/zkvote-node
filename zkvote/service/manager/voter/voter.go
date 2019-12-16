package voter

import (
	"github.com/unitychain/zkvote-node/zkvote/model/subject"
)

// Voter .
type Voter struct {
	subject *subject.Subject
	*IdentityPool
	*Proposal
}

// NewVoter ...
func NewVoter(subject *subject.Subject) (*Voter, error) {

	id, err := NewIdentityPool()
	if nil != err {
		return nil, err
	}
	p, err := NewProposal(id)
	if nil != err {
		return nil, err
	}

	return &Voter{
		subject:      subject,
		IdentityPool: id,
		Proposal:     p,
	}, nil
}
