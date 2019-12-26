package voter

import (
	"fmt"
	"math/big"

	crypto "github.com/ethereum/go-ethereum/crypto"
	. "github.com/unitychain/zkvote-node/zkvote/model/identity"
	"github.com/unitychain/zkvote-node/zkvote/service/utils"
)

type vote struct {
	records  []*big.Int
	opinion  []bool
	finished bool
}

type nullifier struct {
	content string
	hash    *big.Int
	votes   vote
}

// Proposal ...
type Proposal struct {
	nullifiers map[int]*nullifier
	index      int
	members    *IdentityPool
}

const HASH_YES = "43379584054787486383572605962602545002668015983485933488536749112829893476306"
const HASH_NO = "85131057757245807317576516368191972321038229705283732634690444270750521936266"

// NewProposal ...
func NewProposal(identity *IdentityPool) (*Proposal, error) {
	if nil == identity {
		return nil, fmt.Errorf("Invalid input, need IdentityPool object")
	}

	nullifiers := map[int]*nullifier{
		0: &nullifier{
			hash:    big.NewInt(0).SetBytes(crypto.Keccak256([]byte("empty"))),
			content: "empty",
			votes: vote{
				records:  []*big.Int{},
				opinion:  []bool{},
				finished: false,
			},
		}}
	index := 0

	return &Proposal{
		nullifiers: nullifiers,
		index:      index,
		members:    identity,
	}, nil
}

// Propose : propose the hash of a question
func (p *Proposal) Propose(q string) int {
	if 0 == len(q) {
		utils.LogWarning("input qeustion in empty")
		return -1
	}

	// bigHashQus := big.NewInt(0).SetBytes(crypto.Keccak256([]byte(q)))
	bigHashQus := utils.GetBigIntFromHexString(q)
	p.nullifiers[p.index] = &nullifier{
		// TODO: Div(8) is a workaround because a bits conversion issue in circom
		hash:    bigHashQus.Div(bigHashQus, big.NewInt(8)),
		content: q,
		votes: vote{
			opinion:  []bool{},
			finished: false,
		},
	}
	p.index++

	return p.index - 1
}

// VoteWithProof : vote with zk proof
func (p *Proposal) VoteWithProof(idx int, proofs string, vkString string) error {
	if 0 < idx || 0 == len(proofs) || 0 == len(vkString) {
		utils.LogWarningf("invalid input:\n %d,\n %s,\n %s", idx, proofs, vkString)
		return fmt.Errorf("invalid input")
	}

	snarkVote, err := Parse(proofs)
	if nil != err {
		return err
	}
	if !p.isValidVote(idx, snarkVote, vkString) {
		return fmt.Errorf("not a valid vote")
	}

	bigNullHash, _ := big.NewInt(0).SetString(snarkVote.NullifierHash, 10)
	p.nullifiers[idx].votes.records = append(p.nullifiers[idx].votes.records, bigNullHash)

	if snarkVote.PublicSignal[2] == HASH_YES {
		p.nullifiers[idx].votes.opinion = append(p.nullifiers[idx].votes.opinion, true)
	} else {
		p.nullifiers[idx].votes.opinion = append(p.nullifiers[idx].votes.opinion, false)
	}

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
	p.nullifiers[idx].votes.finished = true
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
	ops := nul.votes.opinion

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
func (p *Proposal) isFinished(idx int) bool {
	return p.nullifiers[idx].votes.finished
}

func (p *Proposal) isValidVote(idx int, proofs *Ballot, vkString string) bool {
	if !p.checkIndex(idx) {
		return false
	}
	if p.isFinished(idx) {
		utils.LogWarningf("question has been closed (idex %d)", idx)
		return false
	}

	root := proofs.PublicSignal[0]
	nullifierHash := proofs.PublicSignal[1]
	singalHash := proofs.PublicSignal[2]
	externalNullifier := proofs.PublicSignal[3]

	bigExternalNull, _ := big.NewInt(0).SetString(externalNullifier, 10)
	if 0 != p.nullifiers[idx].hash.Cmp(bigExternalNull) {
		utils.LogWarningf("question doesn't match (%v)/(%v)\n", p.nullifiers[idx].hash, bigExternalNull)
		return false
	}
	if p.isVoted(idx, nullifierHash) {
		utils.LogWarningf("Voted already, %v\n", nullifierHash)
		return false
	}
	if !isValidOpinion(singalHash) {
		utils.LogWarningf("Not a valid vote hash, %v\n", singalHash)
		return false
	}
	bigRoot, _ := big.NewInt(0).SetString(root, 10)
	if !p.members.IsMember(NewIdPathElement(NewTreeContent(bigRoot))) {
		utils.LogWarningf("Not member, %v\n", root)
		return false
	}

	return Verify(vkString, proofs.Proof, proofs.PublicSignal)
}

func (p *Proposal) isVoted(idx int, nullifierHash string) bool {
	bigNullHash, _ := big.NewInt(0).SetString(nullifierHash, 10)
	for _, r := range p.nullifiers[idx].votes.records {
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
