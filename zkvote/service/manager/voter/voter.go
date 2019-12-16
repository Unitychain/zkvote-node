package voter

import (
	"math/big"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/unitychain/zkvote-node/zkvote/model/subject"
)

type VoterSubscription struct {
	idSub   *pubsub.Subscription
	voteSub *pubsub.Subscription
}

// Voter .
type Voter struct {
	subject *subject.Subject
	*IdentityPool
	*Proposal

	pubsub *VoterSubscription
	ps     *pubsub.PubSub
}

// NewVoter ...
func NewVoter(subject *subject.Subject, ps *pubsub.PubSub) (*Voter, error) {

	id, err := NewIdentityPool()
	if nil != err {
		return nil, err
	}
	p, err := NewProposal(id)
	if nil != err {
		return nil, err
	}

	// Subscribe to two topics: one for identities, one for votes
	identitySub, err := ps.Subscribe("identity/" + subject.Hash().Hex().String())
	if err != nil {
		return nil, err
	}
	voteSub, err := ps.Subscribe("vote/" + subject.Hash().Hex().String())
	if err != nil {
		return nil, err
	}

	return &Voter{
		subject:      subject,
		IdentityPool: id,
		Proposal:     p,
		ps:           ps,
		pubsub: &VoterSubscription{
			idSub:   identitySub,
			voteSub: voteSub,
		},
	}, nil
}

func (v *Voter) Register(idCommitment *big.Int) (int, error) {
	i, err := v.Register(idCommitment)
	if nil != err {
		return -1, err
	}
	err = v.ps.Publish(v.GetIdentitySub().Topic(), idCommitment.Bytes())
	if nil != err {
		return -1, err
	}
	return i, nil
}

func (v *Voter) GetSubject() *subject.Subject {
	return v.subject
}

// GetIdentitySub ...
func (v *Voter) GetIdentitySub() *pubsub.Subscription {
	return v.pubsub.idSub
}

// GetVoteSub ...
func (v *Voter) GetVoteSub() *pubsub.Subscription {
	return v.pubsub.voteSub
}
