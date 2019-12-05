package zkvote

import (
	"encoding/hex"
	"fmt"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/manifoldco/promptui"
)

// Voter ...
type Voter struct {
	*Node
	*IdentityProtocol
	proposals     map[SubjectHashHex]*Subject
	subscriptions map[SubjectHashHex]*VoterSubscription
	messages      map[string][]*pubsub.Message
}

// VoterSubscription ...
type VoterSubscription struct {
	identitySubscription *pubsub.Subscription
	voteSubscription     *pubsub.Subscription
}

// NewVoter ...
func NewVoter(node *Node) (*Voter, error) {
	voter := &Voter{
		Node:          node,
		proposals:     make(map[SubjectHashHex]*Subject),
		subscriptions: make(map[SubjectHashHex]*VoterSubscription),
		messages:      make(map[string][]*pubsub.Message),
	}

	done := make(chan bool, 1)
	voter.IdentityProtocol = NewIdentityProtocol(node, done)

	return voter, nil
}

// Propose ...
func (voter *Voter) Propose(subjectTitle string) error {
	// Store the new subject locally
	subject := &Subject{
		title:       subjectTitle,
		description: "Description foobar",
	}

	// Store the created subject
	voter.createdSubjects.Map[subject.hash().hex()] = subject

	fmt.Println(voter.proposals)

	// Subscribe to two topics: one for identities, one for votes
	identitySub, err := voter.pubsub.Subscribe("identity/" + subject.hash().hex().hex)
	if err != nil {
		return err
	}

	voteSub, err := voter.pubsub.Subscribe("vote/" + subject.hash().hex().hex)
	if err != nil {
		return err
	}

	// Store the subscriptions
	voterSub := &VoterSubscription{identitySubscription: identitySub, voteSubscription: voteSub}
	voter.subscriptions[subject.hash().hex()] = voterSub

	go identitySubHandler(voter, subject.hash(), identitySub)
	// go voteSubHandler(voter, voteSub)

	// Announce
	voter.Announce()

	return nil
}

// Join ...
func (voter *Voter) Join(subjectHashHex string) error {
	hash, err := hex.DecodeString(subjectHashHex)
	if err != nil {
		return err
	}
	subjectHash := &SubjectHash{hash: hash}

	identitySub, err := voter.pubsub.Subscribe("identity/" + subjectHashHex)
	if err != nil {
		return err
	}

	voteSub, err := voter.pubsub.Subscribe("vote/" + subjectHashHex)
	if err != nil {
		return err
	}

	// Store the subscriptions
	voterSub := &VoterSubscription{identitySubscription: identitySub, voteSubscription: voteSub}
	voter.subscriptions[subjectHash.hex()] = voterSub

	go identitySubHandler(voter, subjectHash, identitySub)

	return nil
}

// Register ...
func (voter *Voter) Register(subjectHashHex string, identityCommitmentHex string) error {
	subHash, err := hex.DecodeString(subjectHashHex)
	if err != nil {
		return err
	}
	subjectHash := &SubjectHash{hash: subHash}

	idHash, err := hex.DecodeString(identityCommitmentHex)
	if err != nil {
		return err
	}
	identity := &Identity{commitment: idHash}

	voterSubscription := voter.subscriptions[subjectHash.hex()]
	identitySubscription := voterSubscription.identitySubscription
	identityTopic := identitySubscription.Topic()

	// Publish identity hash
	voter.pubsub.Publish(identityTopic, identity.hash().hash)

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

	return voter.pubsub.Publish(topic, []byte(data))
}

// PrintInboundMessages ...
func (voter *Voter) PrintInboundMessages() error {
	voter.RLock()
	topics := make([]string, 0, len(voter.messages))
	for k := range voter.messages {
		topics = append(topics, k)
	}
	voter.RUnlock()

	s := promptui.Select{
		Label: "topic",
		Items: topics,
	}

	_, topic, err := s.Run()
	if err != nil {
		return err
	}

	voter.Lock()
	defer voter.Unlock()
	for _, m := range voter.messages[topic] {
		fmt.Printf("<<< from: %s >>>: %s\n", m.GetFrom(), string(m.GetData()))
	}
	voter.messages[topic] = nil
	return nil
}

// SyncIdentityIndex ...
func (voter *Voter) SyncIdentityIndex() error {
	for subjectHashHex, voterSub := range voter.subscriptions {
		h, _ := hex.DecodeString(subjectHashHex.hex)
		subjectHash := &SubjectHash{hash: h}
		// Get peers from the same pubsub
		peers := voter.pubsub.ListPeers(voterSub.identitySubscription.Topic())
		fmt.Println(peers)
		// Request for registry
		for _, peer := range peers {
			voter.GetIdentityIndexFromPeer(peer, subjectHash)
		}
	}

	return nil
}

// GetIdentityHashes ...
func (voter *Voter) GetIdentityHashes(subjectHash *SubjectHash) []*IdentityHash {
	identityHashSet, ok := voter.identityIndex.Index[subjectHash.hex()]
	if !ok {
		identityHashSet = &IdentityHashSet{set: make(map[IdentityHashHex]string)}
	}
	list := make([]*IdentityHash, 0)
	for hx := range identityHashSet.set {
		h, err := hex.DecodeString(hx.hex)
		if err != nil {
			fmt.Println(err)
		}
		list = append(list, &IdentityHash{hash: h})
	}
	return list
}

func pubsubHandler(voter *Voter, sub *pubsub.Subscription) {
	for {
		m, err := sub.Next(voter.ctx)
		if err != nil {
			fmt.Println(err)
			return
		}
		voter.Lock()
		msgs := voter.messages[sub.Topic()]
		voter.messages[sub.Topic()] = append(msgs, m)
		voter.Unlock()
	}
}

func identitySubHandler(voter *Voter, subjectHash *SubjectHash, subscription *pubsub.Subscription) {
	for {
		m, err := subscription.Next(voter.ctx)
		if err != nil {
			fmt.Println(err)
			return
		}
		voter.Lock()
		identityHash := &IdentityHash{hash: m.GetData()}

		fmt.Println("identitySubHandler: Received message")

		identityHashSet, ok := voter.identityIndex.Index[subjectHash.hex()]
		if !ok {
			identityHashSet = &IdentityHashSet{set: make(map[IdentityHashHex]string)}
		}
		identityHashSet.set[identityHash.hex()] = ""
		voter.identityIndex.Index[subjectHash.hex()] = identityHashSet
		voter.Unlock()
	}
}

func voteSubHandler(voter *Voter, sub *pubsub.Subscription) {
	for {
		m, err := sub.Next(voter.ctx)
		if err != nil {
			fmt.Println(err)
			return
		}
		voter.Lock()
		msgs := voter.messages[sub.Topic()]
		voter.messages[sub.Topic()] = append(msgs, m)
		voter.Unlock()
	}
}
