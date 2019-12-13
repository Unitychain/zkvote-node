package subjectmanager

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/peer"
	routingDiscovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	localContext "github.com/unitychain/zkvote-node/zkvote/model/context"
	"github.com/unitychain/zkvote-node/zkvote/model/subject"
)

// Collector ...
type Collector struct {
	subjProtocol *SubjectProtocol
	*localContext.Context

	ps                *pubsub.PubSub
	dht               *dht.IpfsDHT
	discovery         discovery.Discovery
	providers         map[peer.ID]string
	subjectProtocolCh chan []*subject.Subject
}

// NewCollector ...
func NewCollector(
	pubsub *pubsub.PubSub,
	dht *dht.IpfsDHT,
	lc *localContext.Context,
) (*Collector, error) {
	// Discovery
	rd := routingDiscovery.NewRoutingDiscovery(dht)

	c := &Collector{
		ps:                pubsub,
		dht:               dht,
		discovery:         rd,
		Context:           lc,
		providers:         make(map[peer.ID]string),
		subjectProtocolCh: make(chan []*subject.Subject, 10),
	}
	c.subjProtocol = NewSubjectProtocol(c)

	return c, nil
}

// Announce ...
func (collector *Collector) Announce() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Before advertising, make sure the host has a subscription
	if len(collector.ps.GetTopics()) != 0 {
		fmt.Println("Announce")

		_, err := collector.discovery.Advertise(ctx, "subjects", routingDiscovery.TTL(10*time.Minute))
		return err
	}
	return fmt.Errorf("zknode hasn't subscribed to any topic")
}

// FindProposers ...
func (collector *Collector) FindProposers() (<-chan peer.AddrInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	// defer cancel()
	_ = cancel

	peers, err := collector.discovery.FindPeers(ctx, "subjects")
	if err != nil {
		return nil, err
	}

	return peers, err
}

// Collect ...
func (collector *Collector) Collect() (<-chan *subject.Subject, error) {
	out := make(chan *subject.Subject, 100)
	defer close(out)

	proposers, err := collector.FindProposers()
	if err != nil {
		fmt.Println(err)
	}

	var resultCount int

	for peer := range proposers {
		// Ignore self ID
		if peer.ID == collector.Host.ID() {
			continue
		}
		fmt.Println("found peer", peer)
		collector.Host.Peerstore().AddAddrs(peer.ID, peer.Addrs, 24*time.Hour)
		collector.subjProtocol.GetCreatedSubjects(peer.ID)

		resultCount++
	}

	// TODO: refactor to non-blocking
	for i := 0; i < resultCount; i++ {
		// Block here
		results := <-collector.subjectProtocolCh
		for _, subject := range results {
			out <- subject
		}
	}
	return out, nil
}

// SetProvider ...
func (c *Collector) SetProvider(key peer.ID, value string) {
	c.providers[key] = value
}

// GetProviders ...
func (c *Collector) GetProviders() map[peer.ID]string {
	return c.providers
}

// GetProvider ...
func (c *Collector) GetProvider(key peer.ID) string {
	return c.providers[key]
}

// GetJoinedSubjectTitles ...
func (collector *Collector) GetJoinedSubjectTitles() []string {
	topics := collector.ps.GetTopics()
	fmt.Println(topics)

	return topics
}

// GetCollectedSubjects ...
func (collector *Collector) GetCollectedSubjects() subject.Map {
	return collector.Cache.GetCollectedSubjects()
}

// GetCollectedSubjectTitles ...
func (collector *Collector) GetCollectedSubjectTitles() []string {
	titles := make([]string, 0)
	for _, s := range collector.Cache.GetCollectedSubjects() {
		titles = append(titles, s.GetTitle())
	}

	return titles
}
