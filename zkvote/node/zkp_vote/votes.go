package zkp_vote

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/unitychain/zkvote-node/zkvote/common/utils"
	tree "github.com/unitychain/zkvote-node/zkvote/model/identity"
	s "github.com/unitychain/zkvote-node/zkvote/model/subject"
)

type VoteLeaf struct {
	Subject *s.Subject `json:"subject"`
	Ballots []int      `json:"ballots"`
	Hash    s.HashHex  `json:"hash"`
}

type Votes struct {
	VoteLeaves []*VoteLeaf `json:"VoteLeaves"`
	Root       *big.Int    `json:"root"`
	voteTree   *tree.MerkleTree
}

func NewVotes() (*Votes, error) {
	return &Votes{
		VoteLeaves: []*VoteLeaf{
			&VoteLeaf{
				Subject: nil,
				Ballots: []int{0, 0},
				Hash:    s.HashHex(""),
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

func (v *Votes) CreateAVote(subject *s.Subject) error {
	// Or prevent from the same title
	leaf := v.getVoteLeaf(*subject.HashHex())
	if leaf != nil {
		return fmt.Errorf("Subject is existed, hash:%v", subject.HashHex())
	}

	l := &VoteLeaf{
		Subject: subject,
		Ballots: []int{0, 0},
		Hash:    *subject.HashHex(),
	}
	return v.insertLeaf(l)
}

func (v *Votes) Update(subHashHex s.HashHex, newBallot []int) error {
	leaf := v.getVoteLeaf(subHashHex)
	if leaf == nil {
		return fmt.Errorf("Can't find this subject")
	}
	leaf.Ballots[0] = newBallot[0]
	leaf.Ballots[1] = newBallot[1]

	v.Root = v.calcRoot()
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

//
// Internals
//
func (v *Votes) calcRoot() *big.Int {
	return v.voteTree.GetRoot().BigInt()
}

func (v *Votes) getVoteLeaf(subHashHex s.HashHex) *VoteLeaf {
	for _, l := range v.VoteLeaves {
		if strings.EqualFold(utils.Prepend0x(subHashHex.String()), l.Hash.String()) {
			return l
		}
	}
	utils.LogWarningf("Can't find subject hash, %v", subHashHex)
	return nil
}

func (v *Votes) insertLeaf(leaf *VoteLeaf) error {

	b, e := json.Marshal(v.VoteLeaves)
	if e != nil {
		return e
	}

	x := big.NewInt(0).SetBytes(crypto.Keccak256(b))
	_, e = v.voteTree.Insert(*tree.NewTreeContent(x))
	if e != nil {
		return e
	}

	v.VoteLeaves = append(v.VoteLeaves, leaf)
	return nil
}
