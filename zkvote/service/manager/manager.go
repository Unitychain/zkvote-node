package manager

import (
	"context"
	"encoding/json"
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
	pro "github.com/unitychain/zkvote-node/zkvote/service/manager/protocol"
	"github.com/unitychain/zkvote-node/zkvote/service/manager/voter"
	"github.com/unitychain/zkvote-node/zkvote/service/utils"
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
}

type storeObject struct {
	Subject   subject.Subject `json:"subject"`
	Ids       []id.Identity   `json:"ids"`
	BallotMap ba.Map          `json:"ballots"`
}

func (m *Manager) save(key string, v interface{}) error {
	jsonStr, err := json.Marshal(v)
	if err != nil {
		utils.LogErrorf("Marshal error %v", err.Error())
		return err
	}
	// utils.LogDebugf("save, key: %v, value:%v", key, string(jsonStr))
	err = m.Store.PutLocal(key, string(jsonStr))
	if err != nil {
		utils.LogErrorf("Put local db error, %v", err)
		return err
	}

	return nil
}

func (m *Manager) saveSubjects() error {
	subs := make([]subject.HashHex, len(m.voters))
	i := 0
	for k := range m.voters {
		subs[i] = k
		i++
	}

	return m.save(KEY_SUBJECTS, subs)
}

func (m *Manager) saveSubjectContent(subHex subject.HashHex) error {
	voter, ok := m.voters[subHex]
	if !ok {
		return fmt.Errorf("Can't get voter with subject hash: %v", subHex)
	}
	ids := voter.GetAllIdentities()
	ballotMap := voter.GetBallotMap()
	subj := voter.GetSubject()

	s := &storeObject{
		Subject:   *subj,
		Ids:       ids,
		BallotMap: ballotMap,
	}
	return m.save(subHex.Hash().Hex().String(), s)
}

func (m *Manager) loadSubjects() ([]subject.HashHex, error) {
	value, err := m.Store.GetLocal(KEY_SUBJECTS)
	if err != nil {
		utils.LogErrorf("Get local db error, %v", err)
		return nil, err
	}

	var subs []subject.HashHex
	err = json.Unmarshal([]byte(value), &subs)
	if err != nil {
		utils.LogErrorf("unmarshal subjects error, %v", err)
	}

	utils.LogDebugf("loaded subjects hex: %v", subs)
	return subs, err
}

func (m *Manager) loadSubjectContent(subHex subject.HashHex) (*storeObject, error) {

	utils.LogDebugf("load subject: %s", subHex.String())
	value, err := m.Store.GetLocal(subHex.Hash().Hex().String())
	if err != nil {
		utils.LogErrorf("Get local db error, %v", err)
		return nil, err
	}

	var obj storeObject
	err = json.Unmarshal([]byte(value), &obj)
	if err != nil {
		utils.LogErrorf("unmarshal content of subject error, %v", err)
		return nil, err
	}
	return &obj, nil
}

func (m *Manager) loadDB() {
	subsHex, _ := m.loadSubjects()
	for _, s := range subsHex {
		if 0 == len(s.String()) {
			continue
		}

		obj, err := m.loadSubjectContent(s)
		if err != nil {
			continue
		}

		m.propose(obj.Subject.GetTitle(), obj.Subject.GetDescription(), obj.Subject.GetProposer().String())

		for _, id := range obj.Ids {
			if id.Equal(obj.Subject.GetProposer()) {
				continue
			}
			m.InsertIdentity(utils.Remove0x(obj.Subject.HashHex().String()), id.String())
		}

		go func() {
			for _, b := range obj.BallotMap {
				jStr, err := b.JSON()
				if err != nil {
					utils.LogWarningf("get json ballot error, %v", err)
					break
				}
				m.silentVote(utils.Remove0x(obj.Subject.HashHex().String()), jStr, true)
			}
		}()
	}
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
	}
	m.subjProtocol = pro.NewProtocol(pro.SubjectProtocolType, lc)
	m.idProtocol = pro.NewProtocol(pro.IdentityProtocolType, lc)
	m.ballotProtocol = pro.NewProtocol(pro.BallotProtocolType, lc)

	go m.announce()
	m.loadDB()

	go m.syncSubjectWorker()

	return m, nil
}

//
// vote/identity function
//
// Propose a new subject
func (m *Manager) Propose(title string, description string, identityCommitmentHex string) error {
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

// Vote ...
func (m *Manager) Vote(subjectHashHex string, proof string) error {
	return m.silentVote(subjectHashHex, proof, false)
}

// Open ...
func (m *Manager) Open(subjectHashHex string) (int, int) {
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
	utils.LogInfof("Insert, subject:%s, id:%v", subjectHashHex, identityCommitmentHex)
	if 0 == len(subjectHashHex) || 0 == len(identityCommitmentHex) {
		utils.LogWarningf("Invalid input")
		return fmt.Errorf("invalid input")
	}

	voter, ok := m.voters[subject.HashHex(utils.Remove0x(subjectHashHex))]
	if !ok {
		return fmt.Errorf("Can't get voter with subject hash: %v", subject.HashHex(utils.Remove0x(subjectHashHex)))
	}

	_, err := voter.InsertIdentity(id.NewIdentity(identityCommitmentHex))
	if nil != err {
		utils.LogWarningf("identity pool registration error, %v", err.Error())
		return err
	}

	m.saveSubjectContent(subject.HashHex(subjectHashHex))
	return nil

}

// OverwriteIds ...
func (m *Manager) OverwriteIds(subjectHashHex string, identitySet []string) error {
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

// InsertBallot ...
// TODO: Integrate with InsertIdentity
// func (m *Manager) InsertBallots(subjectHashHex string, ballotStrSet []string) error {
// 	utils.LogInfof("Insert, subject:%s, ballot: %v", subjectHashHex, ballotStrSet)
// 	if 0 == len(subjectHashHex) || 0 == len(ballotStrSet) {
// 		utils.LogErrorf("Invalid input")
// 		return fmt.Errorf("invalid input")
// 	}

// 	voter, ok := m.voters[subject.HashHex(utils.Remove0x(subjectHashHex))]
// 	if !ok {
// 		return fmt.Errorf("Can't get voter with subject hash: %v", subject.HashHex(utils.Remove0x(subjectHashHex)))
// 	}

// 	for _, bs := range ballotStrSet {
// 		ba, err := ba.NewBallot(bs)
// 		if nil != err {
// 			utils.LogWarningf("Ballot insertion error, %v", err.Error())
// 			return err
// 		}

// 		err = voter.Vote(ba)
// 		if nil != err {
// 			utils.LogWarningf("Ballot insertion error, %v", err.Error())
// 			return err
// 		}
// 	}
// 	return nil
// }

//
// pubsub function
//
// Join an existing subject
func (m *Manager) Join(subjectHashHex string, identityCommitmentHex string) error {
	utils.LogInfof("Join, subject:%s, id:%s", subjectHashHex, identityCommitmentHex)
	if 0 == len(subjectHashHex) || 0 == len(identityCommitmentHex) {
		utils.LogErrorf("Invalid input")
		return fmt.Errorf("invalid input")
	}

	subjHex := subject.HashHex(utils.Remove0x(subjectHashHex))

	// No need to new a voter if the subjec is created by itself
	createdSubs := m.GetCreatedSubjects()
	if _, ok := createdSubs[subjHex]; ok {
		return m.InsertIdentity(subjectHashHex, identityCommitmentHex)
	}

	collectedSubs := m.GetCollectedSubjects()
	if sub, ok := collectedSubs[subjHex]; ok {
		// _, err := m.newAVoter(sub, identityCommitmentHex)
		voter, err := voter.NewVoter(sub, m.ps, m.Context, m.zkVerificationKey)
		if nil != err {
			utils.LogErrorf("Join, new voter error: %v", err)
			return err
		}
		m.voters[*sub.HashHex()] = voter

		// Sync identities
		ch, _ := m.SyncIdentities(subjHex)

		// Sync ballots
		go func(ch chan bool) {
			<-ch

			finished, err := m.SyncBallots(subjHex)
			if err != nil {
				utils.LogErrorf("SyncBallotIndex error, %v", err)
			}

			err = m.InsertIdentity(sub.HashHex().String(), identityCommitmentHex)
			if err != nil {
				utils.LogErrorf("insert ID when join error, %v", err)
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

func (m *Manager) waitSubject(ch chan []string) {
	select {
	case results := <-ch:
		for _, ret := range results {
			var s subject.Subject
			err := json.Unmarshal([]byte(ret), &s)
			if err != nil {
				utils.LogWarningf("Unmarshal error, %v", err)
				continue
			}
			m.Cache.InsertColletedSubject(*s.HashHex(), &s)
		}
	case <-time.After(10 * time.Second):
		utils.LogWarning("Collect timeout")
	}

	close(ch)
}

func (m *Manager) syncSubjectWorker() {
	for {
		m.SyncSubjects()
		time.Sleep(60 * time.Second)
	}
}

// SyncSubject ...
func (m *Manager) SyncSubjects() {

	proposers, err := m.FindProposers()
	if err != nil {
		utils.LogErrorf("find peers error, %v", err)
		return
	}

	// TODO: store peers
	for peer := range proposers {
		// Ignore self ID
		if peer.ID == m.Host.ID() {
			continue
		}
		utils.LogInfof("found peer, %v", peer)
		m.Host.Peerstore().AddAddrs(peer.ID, peer.Addrs, 24*time.Hour)

		ch := make(chan []string)
		m.subjProtocol.SubmitRequest(peer.ID, nil, ch)
		go m.waitSubject(ch)
	}
}

//
// Synchronizing functions
//

func (m *Manager) waitIdentities(subjHex subject.HashHex, chIDStrSet chan []string, finished chan bool) {
	select {
	case idStrSet := <-chIDStrSet:
		// CAUTION!
		// Manager needs to overwrite the whole identity pool
		// to keep the order of the tree the same
		m.OverwriteIds(subjHex.String(), idStrSet)
	case <-time.After(10 * time.Second):
		utils.LogWarning("waitIdentities timeout")
	}

	close(chIDStrSet)
	finished <- true
}

// TODO: move to voter.go
// SyncIdentity ...
func (m *Manager) SyncIdentities(subjHex subject.HashHex) (chan bool, error) {
	voter := m.voters[subjHex]
	subjHash := subjHex.Hash()

	// Get peers from the same pubsub
	strTopic := voter.GetIdentitySub().Topic()
	peers := m.ps.ListPeers(strTopic)
	utils.LogDebugf("SyncIdentityIndex peers: %v", peers)

	chPeers := make(chan bool, len(peers))
	for _, peer := range peers {
		ch := make(chan []string)
		m.idProtocol.SubmitRequest(peer, &subjHash, ch)
		go m.waitIdentities(subjHash.Hex(), ch, chPeers)
	}
	return chPeers, nil
}

func (m *Manager) waitBallots(subjHex subject.HashHex, chBallotStrSet chan []string, finished chan bool) {
	select {
	case ballotStrSet := <-chBallotStrSet:
		for _, bs := range ballotStrSet {
			m.silentVote(subjHex.String(), bs, true)
		}
	case <-time.After(10 * time.Second):
		utils.LogWarning("waitBallots timeout")
	}

	finished <- true
	close(chBallotStrSet)
}

// SyncBallot ...
func (m *Manager) SyncBallots(subjHex subject.HashHex) (chan bool, error) {
	voter := m.voters[subjHex]
	subjHash := subjHex.Hash()
	// Get peers from the same pubsub
	peers := m.ps.ListPeers(voter.GetVoteSub().Topic())
	utils.LogDebugf("SyncBallotIndex peers: %v", peers)

	chPeers := make(chan bool, len(peers))
	for _, peer := range peers {
		ch := make(chan []string)
		m.ballotProtocol.SubmitRequest(peer, &subjHash, ch)
		go m.waitBallots(subjHash.Hex(), ch, chPeers)
	}
	return chPeers, nil
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
	// m.Collect()
	for _, s := range m.Cache.GetCollectedSubjects() {
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
// func (m *Manager) GetCollectedSubjectTitles() []string {
// 	titles := make([]string, 0)
// 	for _, s := range m.Cache.GetCollectedSubjects() {
// 		titles = append(titles, s.GetTitle())
// 	}
// 	return titles
// }

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

// GetBallotSet ...
func (m *Manager) GetBallotSet(subjectHashHex *subject.HashHex) ([]*ba.Ballot, error) {
	v, ok := m.voters[*subjectHashHex]
	if !ok {
		return nil, fmt.Errorf("voter is not instantiated")
	}

	ballotSet := make([]*ba.Ballot, 0)
	for _, b := range v.GetBallotMap() {
		ballotSet = append(ballotSet, b)
	}

	return ballotSet, nil
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

	voter, err := m.newAVoter(subject, identityCommitmentHex)
	if nil != err {
		return nil, err
	}

	// Store the created subject
	m.Cache.InsertCreatedSubject(*subject.HashHex(), subject)

	return voter, nil
}

func (m *Manager) newAVoter(sub *subject.Subject, idc string) (*voter.Voter, error) {
	// New a voter including proposal/id tree
	utils.LogInfo("New voter")
	voter, err := voter.NewVoter(sub, m.ps, m.Context, m.zkVerificationKey)
	if nil != err {
		return nil, err
	}

	utils.LogInfof("Register, subject:%s, id:%v", sub.HashHex().String(), idc)
	identity := id.NewIdentity(idc)
	// Insert idenitty to identity pool
	_, err = voter.InsertIdentity(identity)
	if nil != err {
		return nil, err
	}

	// Publish identity to other pubsub peers
	err = voter.Join(identity)
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
