package zkvote

import (
	"fmt"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/manifoldco/promptui"
)

// Voter ...
type Voter struct {
	*Node
	proposals map[string]string
	messages  map[string][]*pubsub.Message
}

// NewVoter ...
func NewVoter(node *Node) (*Voter, error) {
	voter := &Voter{
		Node:      node,
		proposals: make(map[string]string),
		messages:  make(map[string][]*pubsub.Message),
	}

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
	voter.proposals[subjectTitle] = "Description foobar"

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
func (voter *Voter) Register() error {
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
