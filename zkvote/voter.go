package zkvote

import (
	"fmt"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/manifoldco/promptui"
)

// Voter ...
type Voter struct {
	*Node
	*IdentityProtocol
	proposals map[[32]byte]*Subject
	messages  map[string][]*pubsub.Message
}

// NewVoter ...
func NewVoter(node *Node) (*Voter, error) {
	voter := &Voter{
		Node:      node,
		proposals: make(map[[32]byte]*Subject),
		messages:  make(map[string][]*pubsub.Message),
	}

	done := make(chan bool, 1)
	voter.IdentityProtocol = NewIdentityProtocol(node, done)

	return voter, nil
}

// Propose ...
func (voter *Voter) Propose() error {
	p := promptui.Prompt{
		Label: "Subject title",
	}
	subjectTitle, err := p.Run()
	if err != nil {
		return err
	}

	// Store the new subject locally
	subject := &Subject{
		title:       subjectTitle,
		description: "Description foobar",
	}

	// Store the created subject
	voter.createdSubjects[subject.hash()] = subject
	// voter.proposals[subject.hash()] = subject

	fmt.Println(voter.proposals)

	sub, err := voter.pubsub.Subscribe(subjectTitle)
	if err != nil {
		return err
	}

	go pubsubHandler(voter, sub)

	// Announce
	voter.Announce()

	return nil
}

// Join ...
func (voter *Voter) Join() error {
	p := promptui.Prompt{
		Label: "topic name",
	}
	topic, err := p.Run()
	if err != nil {
		return err
	}

	sub, err := voter.pubsub.Subscribe(topic)
	if err != nil {
		return err
	}

	go pubsubHandler(voter, sub)

	return nil
}

// Register ...
func (voter *Voter) Register(subjectTitle string) error {
	voter.pubsub.Publish(subjectTitle, []byte("foo"))
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
	for _, title := range voter.GetJoinedSubjectTitles() {
		// Get peers from the same pubsub
		peers := voter.pubsub.ListPeers(title)
		fmt.Println(peers)
		// Request for registry
		for _, peer := range peers {
			voter.GetIdentityIndexFromPeer(peer)
		}
	}

	return nil
}

// GetIdentityIndex ...
func (voter *Voter) GetIdentityIndex() [][32]byte {
	identityIndex := make([][32]byte, 0)
	for i := range voter.identityIndex {
		identityIndex = append(identityIndex, i)
	}
	return identityIndex
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
