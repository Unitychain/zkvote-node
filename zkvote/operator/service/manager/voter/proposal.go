package voter

import (
	"fmt"
	"math/big"

	crypto "github.com/ethereum/go-ethereum/crypto"
	ba "github.com/unitychain/zkvote-node/zkvote/model/ballot"
	"github.com/unitychain/zkvote-node/zkvote/model/subject"
	"github.com/unitychain/zkvote-node/zkvote/operator/service/utils"
	"github.com/unitychain/zkvote-node/zkvote/snark"
)

type state struct {
	records  []*big.Int
	opinion  []bool
	finished bool
}

type nullifier struct {
	content   string
	hash      *big.Int
	voteState state
}

// Proposal ...
// TODO: Rename
type Proposal struct {
	nullifiers map[int]*nullifier
	ballotMap  ba.Map
	index      int
}

const HASH_YES = "43379584054787486383572605962602545002668015983485933488536749112829893476306"
const HASH_NO = "85131057757245807317576516368191972321038229705283732634690444270750521936266"

// NewProposal ...
func NewProposal() (*Proposal, error) {
	nullifiers := map[int]*nullifier{
		0: &nullifier{
			hash:    big.NewInt(0).SetBytes(crypto.Keccak256([]byte("empty"))),
			content: "empty",
			voteState: state{
				records:  []*big.Int{},
				opinion:  []bool{},
				finished: false,
			},
		}}
	index := 0

	return &Proposal{
		nullifiers: nullifiers,
		ballotMap:  ba.NewMap(),
		index:      index,
	}, nil
}

// Propose : propose the hash of a question
func (p *Proposal) ProposeSubject(subHash subject.HashHex) int {
	if 0 == len(subHash) {
		utils.LogWarning("input qeustion in empty")
		return -1
	}

	// bigHashQus := big.NewInt(0).SetBytes(crypto.Keccak256([]byte(q)))
	bigHashQus := utils.GetBigIntFromHexString(subHash.String())
	p.nullifiers[p.index] = &nullifier{
		// TODO: Div(8) is a workaround because a bits conversion issue in circom
		hash:    bigHashQus.Div(bigHashQus, big.NewInt(8)),
		content: subHash.String(),
		voteState: state{
			opinion:  []bool{},
			finished: false,
		},
	}
	p.index++

	return p.index - 1
}

// VoteWithProof : vote with zk proof
func (p *Proposal) VoteWithProof(ballot *ba.Ballot, vkString string) error {
	if b, e := p.isValidVote(ballot, vkString); !b {
		return e
	}

	bigNullHash, _ := big.NewInt(0).SetString(ballot.NullifierHash, 10)
	p.nullifiers[0].voteState.records = append(p.nullifiers[0].voteState.records, bigNullHash)

	if ballot.PublicSignal[2] == HASH_YES {
		p.nullifiers[0].voteState.opinion = append(p.nullifiers[0].voteState.opinion, true)
	} else {
		p.nullifiers[0].voteState.opinion = append(p.nullifiers[0].voteState.opinion, false)
	}

	p.ballotMap[ballot.NullifierHashHex()] = ballot
	return nil
}

// Remove : remove a proposal from the list
func (p *Proposal) Remove(idx int) {
	if !p.checkIndex(idx) {
		return
	}
	delete(p.nullifiers, idx)
}

// Close : close a proposal which means can't vote anymore
func (p *Proposal) Close(idx int) {
	if !p.checkIndex(idx) {
		return
	}
	p.nullifiers[idx].voteState.finished = true
}

// InsertBallot ...
func (p *Proposal) InsertBallot(ballot *ba.Ballot) error {
	if nil == ballot {
		return fmt.Errorf("invalid input")
	}

	p.ballotMap[ballot.NullifierHashHex()] = ballot
	return nil
}

func (p *Proposal) GetBallots() ba.Map {
	return p.ballotMap
}

// HasProposal : check proposal exists or not
// return : -1, not exists, proposal index otherwise
func (p *Proposal) HasProposal(q string) int {
	for i, e := range p.nullifiers {
		if e.content == q {
			return i
		}
	}
	return -1
}

// HasProposalByHash : check proposal exists or not
// return : -1, not exists, proposal index otherwise
func (p *Proposal) HasProposalByHash(hash *big.Int) int {
	for i, e := range p.nullifiers {
		if 0 == e.hash.Cmp(hash) {
			return i
		}
	}
	return -1
}

// GetCurrentIdex : get current index of whole questions
func (p *Proposal) GetCurrentIdex() int {
	return p.index
}

// GetVotes : get total votes
func (p *Proposal) GetVotes(idx int) (yes, no int) {
	nul := p.getProposal(idx)
	if nul == nil {
		return -1, -1
	}
	ops := nul.voteState.opinion

	for _, o := range ops {
		if true == o {
			yes++
		} else {
			no++
		}
	}
	return yes, no
}

// GetProposal : Get a proposal instance
func (p *Proposal) getProposal(idx int) *nullifier {
	if !p.checkIndex(idx) {
		return nil
	}
	return p.nullifiers[idx]
}

// // GetProposalByHash : Get a proposal instance by question hash
// func (p *Proposal) getProposalByHash(hash *big.Int) *nullifier {
// 	idx := p.HasProposalByHash(hash)
// 	return p.getProposal(idx)
// }

//
// Internal functions
//
func (p *Proposal) isFinished() bool {
	return p.nullifiers[0].voteState.finished
}

func (p *Proposal) isValidVote(ballot *ba.Ballot, vkString string) (bool, error) {
	if 0 == len(vkString) {
		utils.LogWarningf("invalid input: %s", vkString)
		return false, fmt.Errorf("vk string is empty")
	}
	if p.isFinished() {
		utils.LogWarningf("this question has been closed")
		return false, fmt.Errorf("this question has been closed")
	}

	nullifierHash := ballot.PublicSignal[1]
	singalHash := ballot.PublicSignal[2]
	externalNullifier := ballot.PublicSignal[3]

	bigExternalNull, _ := big.NewInt(0).SetString(externalNullifier, 10)
	if 0 != p.nullifiers[0].hash.Cmp(bigExternalNull) {
		utils.LogWarningf("question doesn't match (%v)/(%v)", p.nullifiers[0].hash, bigExternalNull)
		return false, fmt.Errorf(fmt.Sprintf("question doesn't match (%v)/(%v)", p.nullifiers[0].hash, bigExternalNull))
	}
	if p.isVoted(nullifierHash) {
		utils.LogWarningf("Voted already, %v", nullifierHash)
		return false, fmt.Errorf("voted already")
	}
	if !isValidOpinion(singalHash) {
		utils.LogWarningf("Not a valid vote hash, %v", singalHash)
		return false, fmt.Errorf(fmt.Sprintf("Not a valid vote hash, %v", singalHash))
	}

	return snark.Verify(vkString, ballot.Proof, ballot.PublicSignal), nil
}

func (p *Proposal) isVoted(nullifierHash string) bool {
	bigNullHash, _ := big.NewInt(0).SetString(nullifierHash, 10)
	for _, r := range p.nullifiers[0].voteState.records {
		if 0 == bigNullHash.Cmp(r) {
			return true
		}
	}
	return false
}

func isValidOpinion(hash string) bool {
	if hash == HASH_NO || hash == HASH_YES {
		return true
	}
	return false
}

func (p *Proposal) checkIndex(idx int) bool {
	if 0 > idx {
		utils.LogWarningf("invalid index, %d", idx)
		return false
	}
	if idx > p.GetCurrentIdex() {
		utils.LogWarningf("index (%d) is incorrect, max index is %d", idx, p.index)
		return false
	}
	if nil == p.nullifiers[idx] {
		utils.LogWarningf("question doesn't exist (index %d)", idx)
		return false
	}
	return true
}
