package manager

import (
	"encoding/json"
	"time"

	"github.com/unitychain/zkvote-node/zkvote/model/subject"
	"github.com/unitychain/zkvote-node/zkvote/service/utils"
)

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
	case <-time.After(30 * time.Second):
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
	defer func() {
		err := recover()
		if err != nil {
			utils.LogWarningf("PANIC: %v", err)
		}
		close(chIDStrSet)
		m.idLock.Unlock()
	}()

	m.idLock.Lock()
	select {
	case idStrSet := <-chIDStrSet:
		// CAUTION!
		// Manager needs to overwrite the whole identity pool
		// to keep the order of the tree the same
		m.OverwriteIdentities(subjHex.String(), idStrSet)
	case <-time.After(30 * time.Second):
		utils.LogWarning("waitIdentities timeout")
	}

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
	defer func() {
		err := recover()
		if err != nil {
			utils.LogWarningf("PANIC: %v", err)
		}
		close(chBallotStrSet)
		m.ballotLock.Unlock()
	}()

	m.ballotLock.Lock()

	select {
	case ballotStrSet := <-chBallotStrSet:
		utils.LogDebugf("ballot num: %d", len(ballotStrSet))
		for _, bs := range ballotStrSet {
			err := m.silentVote(subjHex.String(), bs, true)
			if err != nil {
				utils.LogErrorf("waitBallots, vote error, %v", err.Error())
			}
		}
	case <-time.After(30 * time.Second):
		utils.LogWarning("waitBallots timeout")
	}

	finished <- true
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
