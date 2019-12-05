package zkvote

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/peer"
	routingDiscovery "github.com/libp2p/go-libp2p-discovery"
)

// Collector ...
type Collector struct {
	*Node
	*SubjectProtocol
	discovery discovery.Discovery
	providers map[peer.ID]string
}

// NewCollector ...
func NewCollector(node *Node) (*Collector, error) {
	// Discovery
	rd := routingDiscovery.NewRoutingDiscovery(node.dht)

	collector := &Collector{
		Node:      node,
		discovery: rd,
		providers: make(map[peer.ID]string),
	}

	done := make(chan bool, 1)
	collector.SubjectProtocol = NewSubjectProtocol(node, done)

	return collector, nil
}

// Announce ...
func (collector *Collector) Announce() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Before advertising, make sure the host has a subscription
	if len(collector.pubsub.GetTopics()) != 0 {
		fmt.Println("Announce")

		_, err := collector.discovery.Advertise(ctx, "subjects", routingDiscovery.TTL(10*time.Minute))
		return err
	}
	return fmt.Errorf("zknode hasn't subscribed to any topic")
}

// FindProposers ...
func (collector *Collector) FindProposers() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	peers, err := collector.discovery.FindPeers(ctx, "subjects")
	if err != nil {
		return err
	}

	for p := range peers {
		fmt.Println("found peer", p)
		collector.Peerstore().AddAddrs(p.ID, p.Addrs, 24*time.Hour)
		collector.providers[p.ID] = ""
	}

	fmt.Println("Subject creators: ")
	fmt.Println(collector.providers)
	return err
}

// Collect ...
func (collector *Collector) Collect() error {
	// Find proposers first
	collector.FindProposers()

	for peer := range collector.providers {
		// Ignore self ID
		if peer == collector.ID() {
			continue
		}
		collector.GetCreatedSubjects(peer)
	}

	return nil
}

// GetJoinedSubjectTitles ...
func (collector *Collector) GetJoinedSubjectTitles() []string {
	topics := collector.pubsub.GetTopics()
	fmt.Println(topics)

	return topics
}

// GetCollectedSubjects ...
func (collector *Collector) GetCollectedSubjects() *SubjectMap {
	return collector.collectedSubjects
}

// GetCollectedSubjectTitles ...
func (collector *Collector) GetCollectedSubjectTitles() []string {
	titles := make([]string, 0)
	for _, subject := range collector.collectedSubjects.Map {
		titles = append(titles, subject.title)
	}

	return titles
}
