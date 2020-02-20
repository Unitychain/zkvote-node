package manager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/peer"
	routingDiscovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/unitychain/zkvote-node/zkvote/common/utils"
	ba "github.com/unitychain/zkvote-node/zkvote/model/ballot"
	localContext "github.com/unitychain/zkvote-node/zkvote/model/context"
	id "github.com/unitychain/zkvote-node/zkvote/model/identity"
	"github.com/unitychain/zkvote-node/zkvote/model/subject"
	pro "github.com/unitychain/zkvote-node/zkvote/operator/service/manager/protocol"
	"github.com/unitychain/zkvote-node/zkvote/operator/service/manager/voter"
)

const KEY_SUBJECTS = "subjects"

// Manager ...
type Manager struct {
	*localContext.Context
	subjProtocol   pro.Protocol
	idProtocol     pro.Protocol
	ballotProtocol pro.Protocol

	ps                *pubsub.PubSub
	dht               *dht.IpfsDHT
	discovery         discovery.Discovery
	providers         map[peer.ID]string
	subjectProtocolCh chan []*subject.Subject
	voters            map[subject.HashHex]*voter.Voter
	chAnnounce        chan bool

	zkVerificationKey string

	idLock     sync.Mutex
	ballotLock sync.Mutex
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
		chAnnounce:        make(chan bool),
		zkVerificationKey: zkVerificationKey,
		idLock:            sync.Mutex{},
		ballotLock:        sync.Mutex{},
	}
	m.subjProtocol = pro.NewProtocol(pro.SubjectProtocolType, lc)
	m.idProtocol = pro.NewProtocol(pro.IdentityProtocolType, lc)
	m.ballotProtocol = pro.NewProtocol(pro.BallotProtocolType, lc)

	go m.announce()
	m.loadDB()

	go m.syncSubjectWorker()

	return m, nil
}

func finally() {
	err := recover()
	if err != nil {
		utils.LogErrorf("PANIC: %v", err)
	}
}

//
// vote/identity function
//
// Propose a new subject
func (m *Manager) Propose(title string, description string, identityCommitmentHex string) error {
	defer finally()

	utils.LogInfof("Propose, title:%v, desc:%v, id:%v", title, description, identityCommitmentHex)
	if 0 == len(title) || 0 == len(identityCommitmentHex) {
		utils.LogErrorf("Invalid input")
		return fmt.Errorf("invalid input")
	}

	voter, err := m.propose(title, description, identityCommitmentHex)
	if err != nil {
		utils.LogErrorf("Propose error, %v", err)
		return err
	}
	m.saveSubjects()
	m.saveSubjectContent(*voter.GetSubject().HashHex())
	return nil
}

// Vote ...
func (m *Manager) Vote(subjectHashHex string, proof string) error {
	defer finally()
	return m.silentVote(subjectHashHex, proof, false)
}

// Open ...
func (m *Manager) Open(subjectHashHex string) (int, int) {
	defer finally()

	utils.LogInfof("Open subject: %v", subjectHashHex)
	voter, ok := m.voters[subject.HashHex(utils.Remove0x(subjectHashHex))]
	if !ok {
		utils.LogErrorf("Can't get voter with subject hash: %v", subject.HashHex(utils.Remove0x(subjectHashHex)))
		return -1, -1
	}
	return voter.Open()
}

// InsertIdentity ...
func (m *Manager) InsertIdentity(subjectHashHex string, identityCommitmentHex string) error {
	defer finally()
	return m.insertIdentity(subjectHashHex, identityCommitmentHex, true)
}

// OverwriteIdentities ...
func (m *Manager) OverwriteIdentities(subjectHashHex string, identitySet []string) error {
	defer finally()

	utils.LogInfof("overwrite, subject:%s", subjectHashHex)
	if 0 == len(subjectHashHex) || 0 == len(identitySet) {
		utils.LogErrorf("Invalid input")
		return fmt.Errorf("invalid input")
	}
	subjHex := subject.HashHex(utils.Remove0x(subjectHashHex))
	voter, ok := m.voters[subjHex]
	if !ok {
		utils.LogErrorf("can't get voter with subject hash:%v", subjHex)
		return fmt.Errorf("can't get voter with subject hash:%v", subjHex)
	}

	// Convert to Identity
	set := make([]*id.Identity, 0)
	for _, idStr := range identitySet {
		set = append(set, id.NewIdentity(idStr))
	}

	_, err := voter.OverwriteIds(set)
	if nil != err {
		utils.LogErrorf("identity pool registration error, %v", err.Error())
		return err
	}

	m.saveSubjectContent(subjHex)
	return nil
}

// Join an existing subject
func (m *Manager) Join(subjectHashHex string, identityCommitmentHex string) error {
	defer finally()

	utils.LogInfof("Join, subject:%s, id:%s", subjectHashHex, identityCommitmentHex)
	if 0 == len(subjectHashHex) || 0 == len(identityCommitmentHex) {
		utils.LogErrorf("Invalid input")
		return fmt.Errorf("invalid input")
	}

	subjHex := subject.HashHex(utils.Remove0x(subjectHashHex))

	// No need to new a voter if the subjec is created by itself
	createdSubs := m.Cache.GetCreatedSubjects()
	if _, ok := createdSubs[subjHex]; ok {
		return m.insertIdentity(subjectHashHex, identityCommitmentHex, true)
	}

	collectedSubs := m.Cache.GetCollectedSubjects()
	if sub, ok := collectedSubs[subjHex]; ok {
		_, err := m.initAVoter(sub, identityCommitmentHex, false)
		if nil != err {
			utils.LogErrorf("Join, init voter error: %v", err)
			return err
		}

		// Sync identities
		ch, _ := m.SyncIdentities(subjHex)

		// Sync ballots
		go func(ch chan bool) {
			<-ch

			finished, err := m.SyncBallots(subjHex)
			if err != nil {
				utils.LogErrorf("SyncBallotIndex error, %v", err)
			}

			<-finished
			m.saveSubjects()
			m.saveSubjectContent(subjHex)
		}(ch)

		// TODO: return sync error
		return err
	}

	return fmt.Errorf("Can NOT find subject, %s", subjectHashHex)
}

// FindProposers ...
func (m *Manager) FindProposers() (<-chan peer.AddrInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	_ = cancel

	peers, err := m.discovery.FindPeers(ctx, "subjects")
	if err != nil {
		return nil, err
	}

	return peers, err
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
	defer finally()

	result := make([]*subject.Subject, 0)
	// m.Collect()
	for _, s := range m.Cache.GetCollectedSubjects() {
		result = append(result, s)
	}
	for _, s := range m.Cache.GetCreatedSubjects() {
		result = append(result, s)
	}

	return result, nil
}

// // GetJoinedSubjectTitles ...
// func (m *Manager) GetJoinedSubjectTitles() []string {
// 	topics := m.ps.GetTopics()
// 	fmt.Println(topics)

// 	return topics
// }

// GetIdentityIndex ...
func (m *Manager) GetIdentityIndex() map[subject.HashHex][]id.Identity {
	defer finally()

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
	defer finally()

	voter, ok := m.voters[subject.HashHex(utils.Remove0x(subjectHashHex))]
	if !ok {
		utils.LogWarningf("can't get voter with subject hash:%v", subject.HashHex(utils.Remove0x(subjectHashHex)))
		return nil, nil, "", fmt.Errorf("can't get voter with subject hash:%v", subject.HashHex(utils.Remove0x(subjectHashHex)))
	}
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

func (m *Manager) propose(title string, description string, identityCommitmentHex string) (*voter.Voter, error) {
	// Store the new subject locally
	identity := id.NewIdentity(identityCommitmentHex)
	if nil == identity {
		return nil, fmt.Errorf("Can not get identity object by commitment %v", identityCommitmentHex)
	}
	subject := subject.NewSubject(title, description, identity)
	if _, ok := m.voters[*subject.HashHex()]; ok {
		return nil, fmt.Errorf("subject already existed")
	}

	voter, err := m.initAVoter(subject, identityCommitmentHex, true)
	if nil != err {
		return nil, err
	}

	// Store the created subject
	m.Cache.InsertCreatedSubject(*subject.HashHex(), subject)

	return voter, nil
}

func (m *Manager) silentVote(subjectHashHex string, proof string, silent bool) error {
	utils.LogInfof("Vote, subject:%s", subjectHashHex)
	if 0 == len(subjectHashHex) || 0 == len(proof) {
		utils.LogErrorf("Invalid input")
		return fmt.Errorf("invalid input")
	}

	subjHex := subject.HashHex(utils.Remove0x(subjectHashHex))
	voter, ok := m.voters[subjHex]
	if !ok {
		utils.LogErrorf("Can't get voter with subject hash: %v", subjHex)
		return fmt.Errorf("Can't get voter with subject hash: %v", subjHex)
	}
	ballot, err := ba.NewBallot(proof)
	if err != nil {
		return err
	}

	err = voter.Vote(ballot, silent)
	if err != nil {
		return err
	}

	m.saveSubjectContent(subjHex)
	return nil
}

func (m *Manager) insertIdentity(subjectHashHex string, identityCommitmentHex string, publish bool) error {
	utils.LogInfof("Insert, subject:%s, id:%v", subjectHashHex, identityCommitmentHex)
	if 0 == len(subjectHashHex) || 0 == len(identityCommitmentHex) {
		utils.LogWarningf("Invalid input")
		return fmt.Errorf("invalid input")
	}

	voter, ok := m.voters[subject.HashHex(utils.Remove0x(subjectHashHex))]
	if !ok {
		return fmt.Errorf("Can't get voter with subject hash: %v", subject.HashHex(utils.Remove0x(subjectHashHex)))
	}

	_, err := voter.InsertIdentity(id.NewIdentity(identityCommitmentHex), publish)
	if nil != err {
		utils.LogWarningf("identity pool registration error, %v", err.Error())
		return err
	}

	m.saveSubjectContent(subject.HashHex(subjectHashHex))
	return nil
}

func (m *Manager) initAVoter(sub *subject.Subject, idc string, publish bool) (*voter.Voter, error) {
	// New a voter including proposal/id tree
	utils.LogDebug("New a voter")
	voter, err := voter.NewVoter(sub, m.ps, m.Context, m.zkVerificationKey)
	if nil != err {
		return nil, err
	}

	utils.LogInfof("Register, subject:%s, id:%v", sub.HashHex().String(), idc)
	identity := id.NewIdentity(idc)
	// Insert idenitty to identity pool
	_, err = voter.InsertIdentity(identity, publish)
	if nil != err {
		return nil, err
	}

	m.voters[*sub.HashHex()] = voter

	if nil != m.chAnnounce {
		m.chAnnounce <- true
	}

	return m.voters[*sub.HashHex()], nil
}

// Announce that the node has a proposal to be discovered
func (m *Manager) announce() error {
	<-m.chAnnounce
	close(m.chAnnounce)
	m.chAnnounce = nil

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// TODO: Check if the voter is ready for announcement
	utils.LogInfo("Announce")
	_, err := m.discovery.Advertise(ctx, "subjects", routingDiscovery.TTL(10*time.Minute))
	if err != nil {
		utils.LogWarningf("Advertise error, %v", err)
	}
	return err
}
