package subjectmanager

import (
	"encoding/hex"
	"fmt"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/manifoldco/promptui"

	localContext "github.com/unitychain/zkvote-node/zkvote/model/context"
	"github.com/unitychain/zkvote-node/zkvote/model/subject"
	id "github.com/unitychain/zkvote-node/zkvote/model/identity"
)

// Voter ...
type Voter struct {
	collector  *Collector
	idProtocol *IdentityProtocol
	*localContext.Context

	ps            *pubsub.PubSub
	subscriptions map[subject.HashHex]*VoterSubscription
	messages      map[string][]*pubsub.Message
}

// VoterSubscription ...
type VoterSubscription struct {
	identitySubscription *pubsub.Subscription
	voteSubscription     *pubsub.Subscription
}

// GetIdentitySub ...
func (v *VoterSubscription) GetIdentitySub() *pubsub.Subscription {
	return v.identitySubscription
}

// GetVoteSub ...
func (v *VoterSubscription) GetVoteSub() *pubsub.Subscription {
	return v.voteSubscription
}

// NewVoter ...
func NewVoter(
	collector *Collector,
	ps *pubsub.PubSub,
	lc *localContext.Context,
) (*Voter, error) {
	v := &Voter{
		Context:       lc,
		ps:            ps,
		collector:     collector,
		subscriptions: make(map[subject.HashHex]*VoterSubscription),
		messages:      make(map[string][]*pubsub.Message),
	}
	v.idProtocol = NewIdentityProtocol(v, make(chan bool, 1))
	return v, nil
}

// Propose ...
func (v *Voter) Propose(subjectTitle string) error {
	// Store the new subject locally
	subject := subject.NewSubject(subjectTitle, "Description foobar")

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
	voterSub := &VoterSubscription{identitySubscription: identitySub, voteSubscription: voteSub}
	v.subscriptions[subject.Hash().Hex()] = voterSub

	go identitySubHandler(v, subject.Hash(), identitySub)
	// go voteSubHandler(v, voteSub)

	// Announce
	v.collector.Announce()

	return nil
}

// Join ...
func (v *Voter) Join(subjectHashHex string) error {
	hash, err := hex.DecodeString(subjectHashHex)
	if err != nil {
		return err
	}
	subjectHash := subject.Hash(hash)

	identitySub, err := v.ps.Subscribe("identity/" + subjectHashHex)
	if err != nil {
		return err
	}

	voteSub, err := v.ps.Subscribe("vote/" + subjectHashHex)
	if err != nil {
		return err
	}

	// Store the subscriptions
	voterSub := &VoterSubscription{identitySubscription: identitySub, voteSubscription: voteSub}
	v.subscriptions[subjectHash.Hex()] = voterSub

	go identitySubHandler(v, &subjectHash, identitySub)

	return nil
}

// Register ...
func (v *Voter) Register(subjectHashHex string, identityCommitmentHex string) error {
	subHash, err := hex.DecodeString(subjectHashHex)
	if err != nil {
		return err
	}
	subjectHash := subject.Hash(subHash)

	// TODO : integrate identity_pool
	identity := id.NewIdentity(identityCommitmentHex)

	voterSubscription := v.subscriptions[subjectHash.Hex()]
	identitySubscription := voterSubscription.identitySubscription
	identityTopic := identitySubscription.Topic()

	// Publish identity hash
	v.ps.Publish(identityTopic, identity.Hash().Byte())

	return nil
}

// Vote ...
func (v *Voter) Vote() error {
	return nil
}

// Open ...
func (v *Voter) Open() error {
	return nil
}

// Broadcast ...
func (v *Voter) Broadcast() error {
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
func (v *Voter) PrintInboundMessages() error {
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
func (v *Voter) SyncIdentityIndex() error {
	for subjectHashHex, voterSub := range v.subscriptions {
		h, _ := hex.DecodeString(subjectHashHex.String())
		subjectHash := subject.Hash(h)
		// Get peers from the same pubsub
		peers := v.ps.ListPeers(voterSub.identitySubscription.Topic())
		fmt.Println(peers)
		// Request for registry
		for _, peer := range peers {
			v.idProtocol.GetIdentityIndexFromPeer(peer, &subjectHash)
		}
	}

	return nil
}

// GetSubscriptions .
func (v *Voter) GetSubscriptions() map[subject.HashHex]*VoterSubscription {
	return v.subscriptions
}

// GetIdentityHashes ...
func (v *Voter) GetIdentityHashes(subjectHash *subject.Hash) []id.Hash {
	identityHashSet := v.Cache.GetAIDIndex(subjectHash.Hex())
	if nil == identityHashSet {
		identityHashSet = id.NewHashSet()
	}
	list := make([]id.Hash, 0)
	for hx := range identityHashSet {
		h, err := hex.DecodeString(hx.String())
		if err != nil {
			fmt.Println(err)
		}
		list = append(list, id.Hash(h))
	}
	return list
}

// InsertIdentity .
func (v *Voter) InsertIdentity(subjectHash *subject.Hash, identityHash id.Hash) {
	v.Mutex.Lock()
	// identityHash := identity.Hash(msg.GetData())
	fmt.Println("identitySubHandler: Received message")

	identityHashSet := v.Cache.GetAIDIndex(subjectHash.Hex())
	if nil == identityHashSet {
		identityHashSet = id.NewHashSet()
	}
	identityHashSet[identityHash.Hex()] = "ID"
	v.Cache.InsertIDIndex(subjectHash.Hex(), identityHashSet)
	v.Mutex.Unlock()
}

func pubsubHandler(v*Voter, sub *pubsub.Subscription) {
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

func identitySubHandler(v *Voter, subjectHash *subject.Hash, subscription *pubsub.Subscription) {
	for {
		m, err := subscription.Next(*v.Ctx)
		if err != nil {
			fmt.Println(err)
			return
		}
		_ = m
		identityHash := id.Hash(m.GetData())
		v.InsertIdentity(subjectHash, identityHash)
		// v.Mutex.Lock()
		// identityHash := identity.Hash(m.GetData())

		// fmt.Println("identitySubHandler: Received message")

		// identityHashSet := v.Cache.GetAIDIndex(string(subjectHash.Hex()))
		// if nil == identityHashSet {
		// 	identityHashSet = identity.NewHashSet()
		// }
		// identityHashSet[identityHash.Hex()] = "ID"
		// v.Cache.InsertIDIndex(string(subjectHash.Hex()), identityHashSet)
		// v.Mutex.Unlock()
	}
}

func voteSubHandler(v *Voter, sub *pubsub.Subscription) {
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
