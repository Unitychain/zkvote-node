package manager

import (
	"context"
	"encoding/hex"
	"encoding/json"
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
	"github.com/unitychain/zkvote-node/zkvote/service/utils"
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
	voters            map[subject.HashHex]*voter.Voter
	// voterMap          map[subject.HashHex]*voter.VoterOrg
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
		voters:            make(map[subject.HashHex]*voter.Voter),
	}
	m.subjProtocol = NewSubjectProtocol(m)
	m.idProtocol = NewIdentityProtocol(m, make(chan bool, 1))

	return m, nil
}

// Propose a new subject
func (m *Manager) Propose(title string, description string, identityCommitmentHex string) error {
	// Store the new subject locally
	subject := subject.NewSubject(title, description)
	if _, ok := m.voters[subject.Hash().Hex()]; ok {
		return fmt.Errorf("subject already existed")
	}

	_, err := m.newAVoter(subject, identityCommitmentHex)
	if nil != err {
		return err
	}

	// Store the created subject
	m.Cache.InsertCreatedSubject(subject.Hash().Hex(), subject)
	fmt.Println(m.Cache.GetCreatedSubjects())

	m.announce()

	return nil
}

// Register ...
func (m *Manager) Register(subjectHashHex string, identityCommitmentHex string) error {
	hash, err := hex.DecodeString(subjectHashHex)
	if err != nil {
		return err
	}
	subjectHash := subject.Hash(hash)

	////
	voter := m.voters[subjectHash.Hex()]
	idx, err := voter.Register(id.HashHex(identityCommitmentHex))
	if nil != err {
		utils.LogWarningf("identity pool registration error, %v", err.Error())
		return err
	}
	_ = idx
	return nil
}

// Join an existing subject
func (m *Manager) Join(subjectHashHex string, identityCommitmentHex string) error {
	// subjectHash := subject.HashHex(subjectHashHex).Hash()
	// TODO: create a subject object and sync data with existing subject
	m.newAVoter(nil, identityCommitmentHex)

	return nil
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
	for _, voter := range m.voters {
		subjectHash := voter.GetSubject().Hash()
		// Get peers from the same pubsub
		peers := m.ps.ListPeers(voter.GetIdentitySub().Topic())
		utils.LogDebugf("%v", peers)
		// Request for registry
		for _, peer := range peers {
			m.idProtocol.GetIdentityIndexFromPeer(peer, subjectHash)
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

// GetIdentityHashes ...
func (m *Manager) GetIdentityHashes(subjectHash *subject.Hash) ([]id.Hash, error) {
	v, ok := m.voters[subjectHash.Hex()]
	if !ok {
		return nil, fmt.Errorf("voter is not instantiated")
	}

	set := v.GetAllIds()
	hashSet := make([]id.Hash, len(set))
	for i, v := range set {
		hashSet[i] = v.Bytes()
	}

	return hashSet, nil
}

func (m *Manager) newAVoter(sub *subject.Subject, idc string) (*voter.Voter, error) {
	// New a voter including proposal/id tree
	voter, err := voter.NewVoter(sub, m.ps, m.Context)
	if nil != err {
		return nil, err
	}
	voter.Register(id.HashHex(idc))

	jsonStr, err := json.Marshal(sub.JSON())
	if nil != err {
		return nil, err
	}
	pid := voter.Propose(string(jsonStr))
	_ = pid

	m.voters[sub.Hash().Hex()] = voter
	return m.voters[sub.Hash().Hex()], nil
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

// GetIdentityIndex ...
func (m *Manager) GetIdentityIndex() map[subject.HashHex][]id.HashHex {
	var index map[subject.HashHex][]id.HashHex
	// index := id.NewIndex()
	for k, v := range m.voters {
		index[k] = v.GetAllIdentities()
	}
	return index
}
