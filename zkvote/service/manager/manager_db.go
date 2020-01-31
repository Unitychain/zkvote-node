package manager

import (
	"encoding/json"
	"fmt"

	ba "github.com/unitychain/zkvote-node/zkvote/model/ballot"
	id "github.com/unitychain/zkvote-node/zkvote/model/identity"
	"github.com/unitychain/zkvote-node/zkvote/model/subject"

	"github.com/unitychain/zkvote-node/zkvote/service/utils"
)

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
			m.insertIdentity(utils.Remove0x(obj.Subject.HashHex().String()), id.String(), true)
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
