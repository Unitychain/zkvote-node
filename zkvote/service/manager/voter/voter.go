package voter

import (
	"fmt"
	"math/big"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	ba "github.com/unitychain/zkvote-node/zkvote/model/ballot"
	localContext "github.com/unitychain/zkvote-node/zkvote/model/context"
	id "github.com/unitychain/zkvote-node/zkvote/model/identity"
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
	ballotMap       ba.Map
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
	p, err := NewProposal()
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
		ballotMap:       ba.NewMap(),
		verificationKey: verificationKey,
		subscription: &voterSubscription{
			idSub:   identitySub,
			voteSub: voteSub,
		},
	}

	go v.identitySubHandler(v.subject.Hash(), v.subscription.idSub)
	go v.voteSubHandler(v.subscription.voteSub)
	return v, nil
}

// Insert .
func (v *Voter) Insert(identity id.Identity) (int, error) {
	i, err := v.InsertIdc(identity.PathElement())
	if nil != err {
		return -1, err
	}
	return i, nil
}

// Register .
func (v *Voter) Register(identity id.Identity) (int, error) {
	i, err := v.Insert(identity)
	if nil != err {
		return -1, err
	}

	err = v.ps.Publish(v.GetIdentitySub().Topic(), identity.Byte())
	if nil != err {
		return -1, err
	}
	return i, nil
}

// GetIdentityIndex .
func (v *Voter) GetIdentityIndex(identity id.Identity) int {
	return v.GetIndex(identity.PathElement())
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

// GetBallotMap ...
func (v *Voter) GetBallotMap() ba.Map {
	return v.ballotMap
}

// Vote .
func (v *Voter) Vote(ballot *ba.Ballot) error {
	bytes, err := ballot.Byte()
	if err != nil {
		return err
	}

	// Check membership
	bigRoot, _ := big.NewInt(0).SetString(ballot.Root, 10)
	if !v.IsMember(id.NewIdPathElement(id.NewTreeContent(bigRoot))) {
		return fmt.Errorf("Not a member")
	}

	// Update voteState
	err = v.VoteWithProof(ballot, v.verificationKey)
	if err != nil {
		return err
	}

	// Store ballot
	v.ballotMap[ballot.Hash().Hex()] = ballot

	return v.ps.Publish(v.GetVoteSub().Topic(), bytes)
}

// Open .
func (v *Voter) Open() (yes, no int) {
	return v.GetVotes(0)
}

// GetAllIdentities .
func (v *Voter) GetAllIdentities() []id.Identity {
	ids := v.GetAllIds()
	hexArray := make([]id.Identity, len(ids))
	for i, _id := range ids {
		hexArray[i] = *id.NewIdentity(_id.Hex())
	}
	return hexArray
}

// GetIdentityPath .
func (v *Voter) GetIdentityPath(identity id.Identity) ([]*id.IdPathElement, []int, *id.IdPathElement, error) {
	elements, paths, root := v.GetIdentityTreePath(identity.PathElement())
	if nil == elements {
		return nil, nil, nil, fmt.Errorf("Can't find the element")
	}
	return elements, paths, root, nil
}

func (v *Voter) identitySubHandler(subjectHash *subject.Hash, subscription *pubsub.Subscription) {
	for {
		m, err := subscription.Next(*v.Ctx)
		if err != nil {
			utils.LogErrorf("Failed to get identity subscription, %v", err.Error())
			continue
		}
		utils.LogDebugf("identitySubHandler: Received message")

		// TODO: Same logic as Register
		identity := id.NewIdentityFromBytes(m.GetData())
		if v.HasRegistered(identity.PathElement()) {
			utils.LogInfof("Got registed id commitment, %v", identity.String())
			continue
		}

		// TODO: Implement consensus for insert
		_, err = v.Insert(*identity)
		if nil != err {
			utils.LogWarningf("Insert id from pubsub error, %v", err.Error())
			continue
		}
	}
}

func (v *Voter) voteSubHandler(sub *pubsub.Subscription) {
	for {
		m, err := sub.Next(*v.Ctx)
		if err != nil {
			utils.LogErrorf("Failed to get vote subscription, %v", err.Error())
			continue
		}
		utils.LogDebugf("voteSubHandler: Received message")

		// Get Ballot
		ballotStr := string(m.GetData())
		ballot, err := ba.NewBallot(ballotStr)
		if err != nil {
			utils.LogWarningf("voteSubHandler: %v", err.Error())
			continue
		}

		// TODO: Same logic as Vote
		// Check membership
		bigRoot, _ := big.NewInt(0).SetString(ballot.Root, 10)
		if !v.IsMember(id.NewIdPathElement(id.NewTreeContent(bigRoot))) {
			err = fmt.Errorf("Not a member")
			utils.LogWarningf("voteSubHandler: %v, %v", err.Error(), bigRoot)
			continue
		}

		// Update voteState
		err = v.VoteWithProof(ballot, v.verificationKey)
		if err != nil {
			utils.LogWarningf("voteSubHandler: %v", err.Error())
			continue
		}

		// Store ballot
		v.ballotMap[ballot.Hash().Hex()] = ballot
	}
}
