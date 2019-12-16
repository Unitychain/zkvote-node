package manager

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/peer"
	routingDiscovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	localContext "github.com/unitychain/zkvote-node/zkvote/model/context"
	id "github.com/unitychain/zkvote-node/zkvote/model/identity"
	"github.com/unitychain/zkvote-node/zkvote/model/subject"
	"github.com/unitychain/zkvote-node/zkvote/service/manager/voter"
)

// Manager ...
type Manager struct {
	subjProtocol *SubjectProtocol
	idProtocol   *IdentityProtocol
	*localContext.Context

	ps                *pubsub.PubSub
	dht               *dht.IpfsDHT
	discovery         discovery.Discovery
	providers         map[peer.ID]string
	subjectProtocolCh chan []*subject.Subject
	voterMap          map[subject.HashHex]*voter.Voter
}

// NewManager ...
func NewManager(
	pubsub *pubsub.PubSub,
	dht *dht.IpfsDHT,
	lc *localContext.Context,
) (*Manager, error) {
	// Discovery
	rd := routingDiscovery.NewRoutingDiscovery(dht)

	m := &Manager{
		ps:                pubsub,
		dht:               dht,
		discovery:         rd,
		Context:           lc,
		providers:         make(map[peer.ID]string),
		subjectProtocolCh: make(chan []*subject.Subject, 10),
		voterMap:          make(map[subject.HashHex]*voter.Voter),
	}
	m.subjProtocol = NewSubjectProtocol(m)
	m.idProtocol = NewIdentityProtocol(m, make(chan bool, 1))

	// TODO: Manage multiple voters

	return m, nil
}

// Propose ...
func (m *Manager) Propose(title string, description string, identityCommitmentHex string) error {
	// Store the new subject locally
	subject := subject.NewSubject(title, description)

	// Store the created subject
	m.Cache.InsertCreatedSubject(subject.Hash().Hex(), subject)
	fmt.Println(m.Cache.GetCreatedSubjects())

	// Create a new voter
	voter, _ := voter.NewVoter(subject, m.ps, m.Context, subject.Hash())
	m.voterMap[subject.Hash().Hex()] = voter

	// TODO: Insert identity
	// identityTopic := identitySub.Topic()
	identity := id.NewIdentity(identityCommitmentHex)

	voter.InsertIdentity(*identity.Hash())

	m.announce()

	return nil
}

// Announce that the node has a proposal to be discovered
func (m *Manager) announce() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// TODO: Check if the voter is ready for announcement
	fmt.Println("Announce")
	_, err := m.discovery.Advertise(ctx, "subjects", routingDiscovery.TTL(10*time.Minute))
	return err
}

// FindProposers ...
func (m *Manager) FindProposers() (<-chan peer.AddrInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	// defer cancel()
	_ = cancel

	peers, err := m.discovery.FindPeers(ctx, "subjects")
	if err != nil {
		return nil, err
	}

	return peers, err
}

// Collect ...
func (m *Manager) Collect() (<-chan *subject.Subject, error) {
	out := make(chan *subject.Subject, 100)
	defer close(out)

	proposers, err := m.FindProposers()
	if err != nil {
		fmt.Println(err)
	}

	var resultCount int

	for peer := range proposers {
		// Ignore self ID
		if peer.ID == m.Host.ID() {
			continue
		}
		fmt.Println("found peer", peer)
		m.Host.Peerstore().AddAddrs(peer.ID, peer.Addrs, 24*time.Hour)
		m.subjProtocol.GetCreatedSubjects(peer.ID)

		resultCount++
	}

	// TODO: refactor to non-blocking
	for i := 0; i < resultCount; i++ {
		// Block here
		results := <-m.subjectProtocolCh
		for _, subject := range results {
			out <- subject
		}
	}
	return out, nil
}

// SyncIdentityIndex ...
func (m *Manager) SyncIdentityIndex() error {
	for _, voter := range m.voterMap {
		for subjectHashHex, voterSub := range voter.Subscriptions {
			h, _ := hex.DecodeString(subjectHashHex.String())
			subjectHash := subject.Hash(h)
			// Get peers from the same pubsub
			peers := voter.Ps.ListPeers(voterSub.GetIdentitySub().Topic())
			fmt.Println(peers)
			// Request for registry
			for _, peer := range peers {
				m.idProtocol.GetIdentityIndexFromPeer(peer, &subjectHash)
			}
		}
	}

	return nil
}

// SetProvider ...
func (m *Manager) SetProvider(key peer.ID, value string) {
	m.providers[key] = value
}

// GetProviders ...
func (m *Manager) GetProviders() map[peer.ID]string {
	return m.providers
}

// GetProvider ...
func (m *Manager) GetProvider(key peer.ID) string {
	return m.providers[key]
}

// GetJoinedSubjectTitles ...
func (m *Manager) GetJoinedSubjectTitles() []string {
	topics := m.ps.GetTopics()
	fmt.Println(topics)

	return topics
}

// GetCollectedSubjects ...
func (m *Manager) GetCollectedSubjects() subject.Map {
	return m.Cache.GetCollectedSubjects()
}

// GetCollectedSubjectTitles ...
func (m *Manager) GetCollectedSubjectTitles() []string {
	titles := make([]string, 0)
	for _, s := range m.Cache.GetCollectedSubjects() {
		titles = append(titles, s.GetTitle())
	}

	return titles
}

// InsertIdentity .
func (m *Manager) InsertIdentity(subjectHash *subject.Hash, identityHash id.Hash) error {
	v, ok := m.voterMap[subjectHash.Hex()]
	if !ok {
		return fmt.Errorf("voter is not instantiated")
	}

	v.InsertIdentity(identityHash)

	return nil

	// m.Mutex.Lock()
	// identityHashSet := m.Cache.GetAIDIndex(subjectHash.Hex())
	// if nil == identityHashSet {
	// 	identityHashSet = id.NewHashSet()
	// }
	// identityHashSet[identityHash.Hex()] = "ID"
	// m.Cache.InsertIDIndex(subjectHash.Hex(), identityHashSet)
	// m.Mutex.Unlock()
}

// GetIdentityHashes ...
func (m *Manager) GetIdentityHashes(subjectHash *subject.Hash) ([]id.Hash, error) {
	v, ok := m.voterMap[subjectHash.Hex()]
	if !ok {
		return nil, fmt.Errorf("voter is not instantiated")
	}

	// identityHashSet := m.Cache.GetAIDIndex(subjectHash.Hex())
	// if nil == identityHashSet {
	// 	identityHashSet = id.NewHashSet()
	// }
	set := v.GetIdentityHashes()

	// list := make([]id.Hash, 0)
	// for hx := range set {
	// 	h, err := hex.DecodeString(hx.String())
	// 	if err != nil {
	// 		fmt.Println(err)
	// 	}
	// 	list = append(list, id.Hash(h))
	// }
	return set, nil
}
