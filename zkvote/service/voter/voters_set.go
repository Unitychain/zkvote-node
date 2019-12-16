package voter

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/manifoldco/promptui"

	localContext "github.com/unitychain/zkvote-node/zkvote/model/context"
	"github.com/unitychain/zkvote-node/zkvote/model/subject"
	"github.com/unitychain/zkvote-node/zkvote/service/utils"
)

// VoterSet ...
type VoterSet struct {
	*localContext.Context
	manager *Manager
	voters  map[subject.HashHex]*Voter

	ps            *pubsub.PubSub
	subscriptions map[subject.HashHex]*voterSubscription
	messages      map[string][]*pubsub.Message
}

type voterSubscription struct {
	identitySubscription *pubsub.Subscription
	voteSubscription     *pubsub.Subscription
}

// GetIdentitySub ...
func (v *voterSubscription) GetIdentitySub() *pubsub.Subscription {
	return v.identitySubscription
}

// GetVoteSub ...
func (v *voterSubscription) GetVoteSub() *pubsub.Subscription {
	return v.voteSubscription
}

// NewVoterSet ...
func NewVoterSet(
	m *Manager,
	ps *pubsub.PubSub,
	lc *localContext.Context,
) (*VoterSet, error) {
	v := &VoterSet{
		Context:       lc,
		ps:            ps,
		manager:       m,
		voters:        make(map[subject.HashHex]*Voter),
		subscriptions: make(map[subject.HashHex]*voterSubscription),
		messages:      make(map[string][]*pubsub.Message),
	}
	return v, nil
}

func (v *VoterSet) newAVoter(sub *subject.Subject, idc string) (*Voter, error) {
	// New a voter including proposal/id tree
	voter, err := NewVoter(sub)
	if nil != err {
		return nil, err
	}
	voter.Register(utils.GetBigIntFromHexString(idc))

	jsonStr, err := json.Marshal(sub.JSON())
	if nil != err {
		return nil, err
	}
	pid := voter.Propose(string(jsonStr))
	_ = pid

	v.voters[sub.Hash().Hex()] = voter
	return v.voters[sub.Hash().Hex()], nil
}

// Propose ...
func (v *VoterSet) Propose(title string, description string) error {
	// Store the new subject locally
	subject := subject.NewSubject(title, description)
	if _, ok := v.voters[subject.Hash().Hex()]; ok {
		return fmt.Errorf("subject already existed")
	}

	// TODO : get id commitment
	_, err := v.newAVoter(subject, "nil")
	if nil != err {
		return err
	}

	// Store the created subject
	v.Cache.InsertCreatedSubject(subject.Hash().Hex(), subject)
	fmt.Println(v.Cache.GetCreatedSubjects())

	// Subscribe to two topics: one for identities, one for votes
	identitySub, err := v.ps.Subscribe("identity/" + subject.Hash().Hex().String())
	if err != nil {
		return err
	}

	voteSub, err := v.ps.Subscribe("vote/" + subject.Hash().Hex().String())
	if err != nil {
		return err
	}

	// Store the subscriptions
	voterSub := &voterSubscription{identitySubscription: identitySub, voteSubscription: voteSub}
	v.subscriptions[subject.Hash().Hex()] = voterSub

	go identitySubHandler(v, subject.Hash(), identitySub)
	// go voteSubHandler(v, voteSub)

	// Announce
	v.manager.Announce()

	return nil
}

// Join ...
func (v *VoterSet) Join(subjectHashHex string) error {
	// hash, err := hex.DecodeString(subjectHashHex)
	// if err != nil {
	// 	return err
	// }
	// subjectHash := subject.Hash(hash)

	// identitySub, err := v.ps.Subscribe("identity/" + subjectHashHex)
	// if err != nil {
	// 	return err
	// }

	// voteSub, err := v.ps.Subscribe("vote/" + subjectHashHex)
	// if err != nil {
	// 	return err
	// }

	// // Store the subscriptions
	// voterSub := &voterSubscription{identitySubscription: identitySub, voteSubscription: voteSub}
	// v.subscriptions[subjectHash.Hex()] = voterSub

	// go identitySubHandler(v, &subjectHash, identitySub)

	return nil
}

// Register ...
func (v *VoterSet) Register(subjectHashHex string, identityCommitmentHex string) error {
	hash, err := hex.DecodeString(subjectHashHex)
	if err != nil {
		return err
	}
	subjectHash := subject.Hash(hash)
	if _, ok := v.voters[subjectHash.Hex()]; !ok {
		fmt.Errorf("subject hasn't been proposed")
	}

	identitySub, err := v.ps.Subscribe("identity/" + subjectHashHex)
	if err != nil {
		return err
	}

	voteSub, err := v.ps.Subscribe("vote/" + subjectHashHex)
	if err != nil {
		return err
	}

	// Store the subscriptions
	voterSub := &voterSubscription{identitySubscription: identitySub, voteSubscription: voteSub}
	v.subscriptions[subjectHash.Hex()] = voterSub

	go identitySubHandler(v, &subjectHash, identitySub)

	voter := v.voters[subjectHash.Hex()]
	idx, err := voter.Register(utils.GetBigIntFromHexString(identityCommitmentHex))
	if nil != err {
		utils.LogWarningf("identity pool registration error, %v", err.Error())
		return err
	}
	_ = idx

	voterSubscription := v.subscriptions[subjectHash.Hex()]
	identitySubscription := voterSubscription.identitySubscription
	identityTopic := identitySubscription.Topic()

	// Publish identity hash
	v.ps.Publish(identityTopic, utils.GetBytesFromHexString(identityCommitmentHex))

	return nil
}

// Vote ...
func (v *VoterSet) Vote() error {
	return nil
}

// Open ...
func (v *VoterSet) Open() error {
	return nil
}

// Broadcast ...
func (v *VoterSet) Broadcast() error {
	p := promptui.Prompt{
		Label: "topic name",
	}
	topic, err := p.Run()
	if err != nil {
		return err
	}

	p = promptui.Prompt{
		Label: "data",
	}
	data, err := p.Run()
	if err != nil {
		return err
	}

	return v.ps.Publish(topic, []byte(data))
}

// PrintInboundMessages ...
func (v *VoterSet) PrintInboundMessages() error {
	v.Mutex.RLock()
	topics := make([]string, 0, len(v.messages))
	for k := range v.messages {
		topics = append(topics, k)
	}
	v.Mutex.RUnlock()

	s := promptui.Select{
		Label: "topic",
		Items: topics,
	}

	_, topic, err := s.Run()
	if err != nil {
		return err
	}

	v.Mutex.Lock()
	defer v.Mutex.Unlock()
	for _, m := range v.messages[topic] {
		fmt.Printf("<<< from: %s >>>: %s\n", m.GetFrom(), string(m.GetData()))
	}
	v.messages[topic] = nil
	return nil
}

// SyncIdentityIndex ...
func (v *VoterSet) SyncIdentityIndex() error {
	for subjectHashHex, voterSub := range v.subscriptions {
		h, _ := hex.DecodeString(subjectHashHex.String())
		subjectHash := subject.Hash(h)
		// Get peers from the same pubsub
		peers := v.ps.ListPeers(voterSub.identitySubscription.Topic())
		fmt.Println(peers)
		// Request for registry
		for _, peer := range peers {
			v.manager.idProtocol.GetIdentityIndexFromPeer(peer, &subjectHash)
		}
	}

	return nil
}

// GetSubscriptions .
func (v *VoterSet) GetSubscriptions() map[subject.HashHex]*voterSubscription {
	return v.subscriptions
}

func pubsubHandler(v *VoterSet, sub *pubsub.Subscription) {
	for {
		m, err := sub.Next(*v.Ctx)
		if err != nil {
			fmt.Println(err)
			return
		}

		v.Mutex.Lock()
		msgs := v.messages[sub.Topic()]
		v.messages[sub.Topic()] = append(msgs, m)
		v.Mutex.Unlock()
	}
}

func identitySubHandler(v *VoterSet, subjectHash *subject.Hash, subscription *pubsub.Subscription) {
	for {
		m, err := subscription.Next(*v.Ctx)
		if err != nil {
			fmt.Println(err)
			return
		}
		_ = m
		// identityHash := id.Hash(m.GetData())
		// v.manager.InsertIdentity(subjectHash, identityHash)
	}
}

func voteSubHandler(v *VoterSet, sub *pubsub.Subscription) {
	for {
		m, err := sub.Next(*v.Ctx)
		if err != nil {
			fmt.Println(err)
			return
		}
		v.Mutex.Lock()
		msgs := v.messages[sub.Topic()]
		v.messages[sub.Topic()] = append(msgs, m)
		v.Mutex.Unlock()
	}
}
