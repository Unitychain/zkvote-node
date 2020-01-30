package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/unitychain/zkvote-node/zkvote/model/subject"
	"github.com/unitychain/zkvote-node/zkvote/service/manager/voter"
	"github.com/unitychain/zkvote-node/zkvote/service/utils"
)

// Join an existing subject
func (m *Manager) Join(subjectHashHex string, identityCommitmentHex string) error {
	utils.LogInfof("Join, subject:%s, id:%s", subjectHashHex, identityCommitmentHex)
	if 0 == len(subjectHashHex) || 0 == len(identityCommitmentHex) {
		utils.LogErrorf("Invalid input")
		return fmt.Errorf("invalid input")
	}

	subjHex := subject.HashHex(utils.Remove0x(subjectHashHex))

	// No need to new a voter if the subjec is created by itself
	createdSubs := m.Cache.GetCreatedSubjects()
	if _, ok := createdSubs[subjHex]; ok {
		return m.InsertIdentity(subjectHashHex, identityCommitmentHex)
	}

	collectedSubs := m.Cache.GetCollectedSubjects()
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

func (m *Manager) syncSubjectWorker() {
	for {
		m.SyncSubjects()
		time.Sleep(60 * time.Second)
	}
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
		m.OverwriteIdentities(subjHex.String(), idStrSet)
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
