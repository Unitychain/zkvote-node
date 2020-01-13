package manager

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/peer"
	routingDiscovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	ba "github.com/unitychain/zkvote-node/zkvote/model/ballot"
	localContext "github.com/unitychain/zkvote-node/zkvote/model/context"
	id "github.com/unitychain/zkvote-node/zkvote/model/identity"
	"github.com/unitychain/zkvote-node/zkvote/model/subject"
	"github.com/unitychain/zkvote-node/zkvote/service/manager/voter"
	"github.com/unitychain/zkvote-node/zkvote/service/utils"
)

// Manager ...
type Manager struct {
	*localContext.Context
	subjProtocol *SubjectProtocol
	idProtocol   *IdentityProtocol

	ps                *pubsub.PubSub
	dht               *dht.IpfsDHT
	discovery         discovery.Discovery
	providers         map[peer.ID]string
	subjectProtocolCh chan []*subject.Subject
	voters            map[subject.HashHex]*voter.Voter

	zkVerificationKey string
}

// NewManager ...
func NewManager(
	pubsub *pubsub.PubSub,
	dht *dht.IpfsDHT,
	lc *localContext.Context,
	zkVerificationKey string,
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
		zkVerificationKey: zkVerificationKey,
	}
	m.subjProtocol = NewSubjectProtocol(m)
	m.idProtocol = NewIdentityProtocol(m, make(chan bool, 1))

	return m, nil
}

//
// vote/identity function
//
// Propose a new subject
func (m *Manager) Propose(title string, description string, identityCommitmentHex string) error {
	// Store the new subject locally
	identity := id.NewIdentity(identityCommitmentHex)
	subject := subject.NewSubject(title, description, identity)
	if _, ok := m.voters[subject.Hash().Hex()]; ok {
		return fmt.Errorf("subject already existed")
	}

	_, err := m.newAVoter(subject, identityCommitmentHex)
	if nil != err {
		return err
	}

	// Store the created subject
	m.Cache.InsertCreatedSubject(subject.Hash().Hex(), subject)

	m.announce()

	return nil
}

//
// Protocol Handler
//
// Insert ...
func (m *Manager) Insert(subjectHashHex string, identityCommitmentHex string) error {
	voter := m.voters[subject.HashHex(subjectHashHex)]
	idx, err := voter.Insert(*id.NewIdentity(identityCommitmentHex))
	if nil != err {
		utils.LogWarningf("identity pool registration error, %v", err.Error())
		return err
	}
	_ = idx
	return nil
}

// Overwrite ...
func (m *Manager) Overwrite(subjectHashHex string, identitySet []string) error {
	voter := m.voters[subject.HashHex(subjectHashHex)]

	// Convert to idPathElement
	set := make([]*id.IdPathElement, 0)
	for _, idStr := range identitySet {
		set = append(set, id.NewIdentity(idStr).PathElement())
	}

	_, err := voter.Overwrite(set)
	if nil != err {
		utils.LogWarningf("identity pool registration error, %v", err.Error())
		return err
	}

	return nil
}

//
// pubsub function
//
// Join an existing subject
func (m *Manager) Join(subjectHashHex string, identityCommitmentHex string) error {
	// Check if subjectHash is created by itself
	createdSubs := m.GetCreatedSubjects()
	if _, ok := createdSubs[subject.HashHex(subjectHashHex)]; ok {
		return m.Insert(subjectHashHex, identityCommitmentHex)
	}

	collectedSubs := m.GetCollectedSubjects()
	if sub, ok := collectedSubs[subject.HashHex(subjectHashHex)]; ok {
		_, err := m.newAVoter(sub, identityCommitmentHex)
		return err
	}
	return fmt.Errorf("Can NOT find subject, %s", subjectHashHex)
}

// Vote ...
func (m *Manager) Vote(subjectHashHex string, proof string) error {
	voter := m.voters[subject.HashHex(subjectHashHex)]
	ballot, err := ba.NewBallot(proof)
	if err != nil {
		return err
	}
	return voter.Vote(ballot)
}

// Open ...
func (m *Manager) Open(subjectHashHex string) (int, int) {
	voter := m.voters[subject.HashHex(subjectHashHex)]
	return voter.Open()
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

//
// identity/subject getters
//
// GetSubjectList ...
func (m *Manager) GetSubjectList() ([]*subject.Subject, error) {
	result := make([]*subject.Subject, 0)
	collections, _ := m.Collect()
	for s := range collections {
		result = append(result, s)
	}
	for _, s := range m.Cache.GetCreatedSubjects() {
		result = append(result, s)
	}
	return result, nil
}

// GetJoinedSubjectTitles ...
func (m *Manager) GetJoinedSubjectTitles() []string {
	topics := m.ps.GetTopics()
	fmt.Println(topics)

	return topics
}

// GetCreatedSubjects ...
func (m *Manager) GetCreatedSubjects() subject.Map {
	return m.Cache.GetCreatedSubjects()
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

// GetIdentityIndex ...
func (m *Manager) GetIdentityIndex() map[subject.HashHex][]id.Identity {
	index := make(map[subject.HashHex][]id.Identity)
	for k, v := range m.voters {
		index[k] = v.GetAllIdentities()
	}
	return index
}

// GetIdentityPath .
// return intermediate values and merkle root in hex string
func (m *Manager) GetIdentityPath(
	subjectHashHex string,
	identityCommitmentHex string) (
	[]string, []int, string, error) {

	voter := m.voters[subject.HashHex(subjectHashHex)]
	idPaths, idPathIndexes, root, err := voter.GetIdentityPath(*id.NewIdentity(identityCommitmentHex))
	if err != nil {
		return nil, nil, "", err
	}
	hexIDPaths := make([]string, len(idPaths))
	for i, p := range idPaths {
		hexIDPaths[i] = p.Hex()
	}
	return hexIDPaths, idPathIndexes, root.Hex(), nil
}

// GetIdentitySet ...
func (m *Manager) GetIdentitySet(subjectHash *subject.Hash) ([]id.Identity, error) {
	v, ok := m.voters[subjectHash.Hex()]
	if !ok {
		return nil, fmt.Errorf("voter is not instantiated")
	}

	set := v.GetAllIds()
	hashSet := make([]id.Identity, len(set))
	for i, v := range set {
		hashSet[i] = *id.NewIdentity(v.Hex())
	}

	return hashSet, nil
}

//
// CLI Debugger
//
// GetVoterIdentities ...
func (m *Manager) GetVoterIdentities() map[subject.HashHex][]*id.IdPathElement {
	result := make(map[subject.HashHex][]*id.IdPathElement)

	for k, v := range m.voters {
		result[k] = v.GetAllIds()
	}
	return result
}

// GetBallotMaps ...
func (m *Manager) GetBallotMaps() map[subject.HashHex]ba.Map {
	result := make(map[subject.HashHex]ba.Map)

	for k, v := range m.voters {
		result[k] = v.GetBallotMap()
	}
	return result
}

//
// internal functions
//
func (m *Manager) newAVoter(sub *subject.Subject, idc string) (*voter.Voter, error) {
	// New a voter including proposal/id tree
	voter, err := voter.NewVoter(sub, m.ps, m.Context, m.zkVerificationKey)
	if nil != err {
		return nil, err
	}
	_, err = voter.Register(*id.NewIdentity(idc))
	if nil != err {
		return nil, err
	}

	pid := voter.Propose(sub.Hash().Hex().String())
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
