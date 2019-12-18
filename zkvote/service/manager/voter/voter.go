package voter

import (
	"encoding/hex"
	"math/big"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	localContext "github.com/unitychain/zkvote-node/zkvote/model/context"
	"github.com/unitychain/zkvote-node/zkvote/model/identity"
	"github.com/unitychain/zkvote-node/zkvote/model/subject"
	"github.com/unitychain/zkvote-node/zkvote/service/utils"
)

type voterSubscription struct {
	idSub   *pubsub.Subscription
	voteSub *pubsub.Subscription
}

// Voter .
type Voter struct {
	subject *subject.Subject
	*IdentityPool
	*Proposal
	verificationKey string

	*localContext.Context
	ps           *pubsub.PubSub
	subscription *voterSubscription
	pubMsg       map[string][]*pubsub.Message
}

// NewVoter ...
func NewVoter(subject *subject.Subject, ps *pubsub.PubSub, lc *localContext.Context, verificationKey string) (*Voter, error) {
	id, err := NewIdentityPool()
	if nil != err {
		return nil, err
	}
	p, err := NewProposal(id)
	if nil != err {
		return nil, err
	}

	identitySub, err := ps.Subscribe("identity/" + subject.Hash().Hex().String())
	if err != nil {
		return nil, err
	}
	voteSub, err := ps.Subscribe("vote/" + subject.Hash().Hex().String())
	if err != nil {
		return nil, err
	}

	v := &Voter{
		subject:         subject,
		IdentityPool:    id,
		Proposal:        p,
		ps:              ps,
		Context:         lc,
		verificationKey: verificationKey,
		subscription: &voterSubscription{
			idSub:   identitySub,
			voteSub: voteSub,
		},
	}

	go v.identitySubHandler(v.subject.Hash(), v.subscription.idSub)
	// go v.voteSubHandler(v.subscription.idSub)
	return v, nil
}

// Register .
func (v *Voter) Register(idcHex identity.Identity) (int, error) {
	idc := utils.GetBigIntFromHexString(idcHex.String())
	i, err := v.Insert(idc)
	if nil != err {
		return -1, err
	}
	err = v.ps.Publish(v.GetIdentitySub().Topic(), idc.Bytes())
	if nil != err {
		return -1, err
	}
	return i, nil
}

// GetSubject .
func (v *Voter) GetSubject() *subject.Subject {
	return v.subject
}

// GetIdentitySub ...
func (v *Voter) GetIdentitySub() *pubsub.Subscription {
	return v.subscription.idSub
}

// GetVoteSub ...
func (v *Voter) GetVoteSub() *pubsub.Subscription {
	return v.subscription.voteSub
}

// Vote .
func (v *Voter) Vote(proofs string) error {
	return v.VoteWithProof(0, proofs, v.verificationKey)
}

// Open .
func (v *Voter) Open() (yes, no int) {
	return v.GetVotes(0)
}

// GetAllIdentities .
func (v *Voter) GetAllIdentities() []identity.Identity {
	ids := v.GetAllIds()
	hexArray := make([]identity.Identity, len(ids))
	for i, id := range ids {
		hexArray[i] = *identity.NewIdentity(hex.EncodeToString(id.Bytes()))
	}
	return hexArray
}

func (v *Voter) identitySubHandler(subjectHash *subject.Hash, subscription *pubsub.Subscription) {
	for {
		m, err := subscription.Next(*v.Ctx)
		if err != nil {
			utils.LogErrorf("Failed to get identity subscription, %v", err.Error())
			return
		}
		utils.LogDebugf("identitySubHandler: Received message")

		identityInt := big.NewInt(0).SetBytes(m.GetData())
		if v.HasRegistered(identityInt) {
			utils.LogInfof("Got registed id commitment, %v", identityInt)
			continue
		}
		_, err = v.Insert(identityInt)
		if nil != err {
			utils.LogWarningf("Insert id from pubsub error, %v", err.Error())
		}
	}
}

func (v *Voter) voteSubHandler(sub *pubsub.Subscription) {
	for {
		m, err := sub.Next(*v.Ctx)
		if err != nil {
			utils.LogErrorf("Failed to get vote subscription, %v", err.Error())
			return
		}
		v.Mutex.Lock()
		msgs := v.pubMsg[sub.Topic()]
		v.pubMsg[sub.Topic()] = append(msgs, m)
		v.Mutex.Unlock()
	}
}
