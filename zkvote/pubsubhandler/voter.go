package pubsubhandler

import (
	"context"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/libp2p/go-libp2p-core/host"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/manifoldco/promptui"

	identity "github.com/unitychain/zkvote-node/zkvote/pubsubhandler/identity"
	subject "github.com/unitychain/zkvote-node/zkvote/pubsubhandler/subject"
	"github.com/unitychain/zkvote-node/zkvote/store"
)

// Voter ...
type Voter struct {
	host          host.Host
	collector     *Collector
	ctx           *context.Context
	ps            *pubsub.PubSub
	idProtocol    *IdentityProtocol
	cache         *store.Cache
	mu            *sync.RWMutex
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
	host host.Host,
	ctx *context.Context,
	collector *Collector,
	ps *pubsub.PubSub,
	cache *store.Cache,
	mu *sync.RWMutex,
) (*Voter, error) {
	v := &Voter{
		host:          host,
		ctx:           ctx,
		ps:            ps,
		collector:     collector,
		cache:         cache,
		mu:            mu,
		subscriptions: make(map[subject.HashHex]*VoterSubscription),
		messages:      make(map[string][]*pubsub.Message),
	}
	v.idProtocol = NewIdentityProtocol(host, v, cache, make(chan bool, 1))
	return v, nil
}

// Propose ...
func (voter *Voter) Propose(subjectTitle string) error {
	// Store the new subject locally
	subject := subject.NewSubject(subjectTitle, "Description foobar")

	// Store the created subject
	voter.cache.InsertCreatedSubject(subject.Hash().Hex(), subject)
	fmt.Println(voter.cache.GetCreatedSubjects())

	// Subscribe to two topics: one for identities, one for votes
	identitySub, err := voter.ps.Subscribe("identity/" + subject.Hash().Hex().String())
	if err != nil {
		return err
	}

	voteSub, err := voter.ps.Subscribe("vote/" + subject.Hash().Hex().String())
	if err != nil {
		return err
	}

	// Store the subscriptions
	voterSub := &VoterSubscription{identitySubscription: identitySub, voteSubscription: voteSub}
	voter.subscriptions[subject.Hash().Hex()] = voterSub

	go identitySubHandler(voter, subject.Hash(), identitySub)
	// go voteSubHandler(voter, voteSub)

	// Announce
	voter.collector.Announce()

	return nil
}

// Join ...
func (voter *Voter) Join(subjectHashHex string) error {
	hash, err := hex.DecodeString(subjectHashHex)
	if err != nil {
		return err
	}
	subjectHash := subject.Hash(hash)

	identitySub, err := voter.ps.Subscribe("identity/" + subjectHashHex)
	if err != nil {
		return err
	}

	voteSub, err := voter.ps.Subscribe("vote/" + subjectHashHex)
	if err != nil {
		return err
	}

	// Store the subscriptions
	voterSub := &VoterSubscription{identitySubscription: identitySub, voteSubscription: voteSub}
	voter.subscriptions[subjectHash.Hex()] = voterSub

	go identitySubHandler(voter, &subjectHash, identitySub)

	return nil
}

// Register ...
func (voter *Voter) Register(subjectHashHex string, identityCommitmentHex string) error {
	subHash, err := hex.DecodeString(subjectHashHex)
	if err != nil {
		return err
	}
	subjectHash := subject.Hash(subHash)

	identity := identity.NewIdentity(identityCommitmentHex)

	voterSubscription := voter.subscriptions[subjectHash.Hex()]
	identitySubscription := voterSubscription.identitySubscription
	identityTopic := identitySubscription.Topic()

	// Publish identity hash
	voter.ps.Publish(identityTopic, identity.Hash().Byte())

	return nil
}

// Vote ...
func (voter *Voter) Vote() error {
	return nil
}

// Open ...
func (voter *Voter) Open() error {
	return nil
}

// Broadcast ...
func (voter *Voter) Broadcast() error {
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

	return voter.ps.Publish(topic, []byte(data))
}

// PrintInboundMessages ...
func (voter *Voter) PrintInboundMessages() error {
	voter.mu.RLock()
	topics := make([]string, 0, len(voter.messages))
	for k := range voter.messages {
		topics = append(topics, k)
	}
	voter.mu.RUnlock()

	s := promptui.Select{
		Label: "topic",
		Items: topics,
	}

	_, topic, err := s.Run()
	if err != nil {
		return err
	}

	voter.mu.Lock()
	defer voter.mu.Unlock()
	for _, m := range voter.messages[topic] {
		fmt.Printf("<<< from: %s >>>: %s\n", m.GetFrom(), string(m.GetData()))
	}
	voter.messages[topic] = nil
	return nil
}

// SyncIdentityIndex ...
func (voter *Voter) SyncIdentityIndex() error {
	for subjectHashHex, voterSub := range voter.subscriptions {
		h, _ := hex.DecodeString(subjectHashHex.String())
		subjectHash := subject.Hash(h)
		// Get peers from the same pubsub
		peers := voter.ps.ListPeers(voterSub.identitySubscription.Topic())
		fmt.Println(peers)
		// Request for registry
		for _, peer := range peers {
			voter.idProtocol.GetIdentityIndexFromPeer(peer, &subjectHash)
		}
	}

	return nil
}

// GetSubscriptions .
func (voter *Voter) GetSubscriptions() map[subject.HashHex]*VoterSubscription {
	return voter.subscriptions
}

// GetIdentityHashes ...
func (voter *Voter) GetIdentityHashes(subjectHash *subject.Hash) []identity.Hash {
	identityHashSet := voter.cache.GetAIDIndex(string(subjectHash.Hex()))
	if nil == identityHashSet {
		identityHashSet = identity.NewHashSet()
	}
	list := make([]identity.Hash, 0)
	for hx := range identityHashSet {
		h, err := hex.DecodeString(hx.String())
		if err != nil {
			fmt.Println(err)
		}
		list = append(list, identity.Hash(h))
	}
	return list
}

func pubsubHandler(voter *Voter, sub *pubsub.Subscription) {
	for {
		m, err := sub.Next(*voter.ctx)
		if err != nil {
			fmt.Println(err)
			return
		}

		voter.mu.Lock()
		msgs := voter.messages[sub.Topic()]
		voter.messages[sub.Topic()] = append(msgs, m)
		voter.mu.Unlock()
	}
}

func identitySubHandler(voter *Voter, subjectHash *subject.Hash, subscription *pubsub.Subscription) {
	for {
		m, err := subscription.Next(*voter.ctx)
		if err != nil {
			fmt.Println(err)
			return
		}
		_ = m

		voter.mu.Lock()
		identityHash := identity.Hash(m.GetData())

		fmt.Println("identitySubHandler: Received message")

		identityHashSet := voter.cache.GetAIDIndex(string(subjectHash.Hex()))
		if nil == identityHashSet {
			identityHashSet = identity.NewHashSet()
		}
		identityHashSet[identityHash.Hex()] = "ID"
		voter.cache.InsertIDIndex(string(subjectHash.Hex()), identityHashSet)
		voter.mu.Unlock()
	}
}

func voteSubHandler(voter *Voter, sub *pubsub.Subscription) {
	for {
		m, err := sub.Next(*voter.ctx)
		if err != nil {
			fmt.Println(err)
			return
		}
		voter.mu.Lock()
		msgs := voter.messages[sub.Topic()]
		voter.messages[sub.Topic()] = append(msgs, m)
		voter.mu.Unlock()
	}
}
