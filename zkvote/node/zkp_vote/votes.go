package zkp_vote

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/unitychain/zkvote-node/zkvote/common/utils"
	s "github.com/unitychain/zkvote-node/zkvote/model/subject"
)

type VoteLeaf struct {
	Subject *s.Subject `json:"subject"`
	Ballots []int      `json:"ballots"`
	hash    s.HashHex  `json:"hash"`
}

type Votes struct {
	Votes []*VoteLeaf `json: "VoteLeaves"`
	Root  *big.Int    `json: "root"`
}

func NewVotes() (*Votes, error) {
	return &Votes{
		Votes: []*VoteLeaf{
			&VoteLeaf{
				Subject: nil,
				Ballots: []int{0, 0},
				hash:    s.HashHex(""),
			}},
		Root: big.NewInt(0),
	}, nil
}

func NewVotesWithSerializedString(jsonStr string) (*Votes, error) {
	var v Votes
	err := json.Unmarshal([]byte(jsonStr), &v)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (v *Votes) Serialize() (string, error) {
	jsonStrB, err := json.Marshal(v)
	if err != nil {
		utils.LogErrorf("Marshal error %v", err.Error())
		return "", err
	}
	return string(jsonStrB), nil
}

func (v *Votes) Update(subHashHex s.HashHex, newBallot []int) error {
	leaf := v.getVoteLeaf(subHashHex)
	if leaf == nil {
		return fmt.Errorf("Can't find this subject")
	}
	leaf.Ballots[0] = newBallot[0]
	leaf.Ballots[1] = newBallot[1]

	newRoot, err := v.calcRoot()
	if err != nil {
		return err
	}
	v.Root = newRoot
	return nil
}

func (v *Votes) IsValidBallotNumber(subHashHex s.HashHex, newBallot []int) (bool, error) {
	leaf := v.getVoteLeaf(subHashHex)
	if leaf == nil {
		return false, fmt.Errorf("Can't find this subject")
	}

	if leaf.Ballots[0] > newBallot[0] || leaf.Ballots[1] > newBallot[1] {
		return false, fmt.Errorf("number of coming ballots is wrong, current:%v, coming:%v", leaf.Ballots, newBallot)
	}

	return true, nil
}

func (v *Votes) IsRootMatched(root *big.Int) bool {
	if 0 == v.Root.Cmp(root) {
		return true
	}
	return false
}

func (v *Votes) calcRoot() (*big.Int, error) {
	return big.NewInt(0), nil
}

func (v *Votes) getVoteLeaf(subHashHex s.HashHex) *VoteLeaf {
	for _, l := range v.Votes {
		if strings.EqualFold(utils.Prepend0x(subHashHex.String()), l.hash.String()) {
			return l
		}
	}
	utils.LogWarningf("Can't find subject hash, %v", subHashHex)
	return nil
}
