package voter

import (
	"fmt"
	"math/big"

	"../snark"
	crypto "github.com/ethereum/go-ethereum/crypto"
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

type Proposal struct {
	nullifiers map[int]*nullifier
	index      int
	members    *Identity
}

const HASH_YES = "43379584054787486383572605962602545002668015983485933488536749112829893476306"
const HASH_NO = "89477152217924674838424037953991966239322087453347756267410168184682657981552"

// NewProposal ...
func NewProposal(identity *Identity) (*Proposal, error) {

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

	// TODO: load data from DHT/PubSub

	return &Proposal{
		nullifiers: nullifiers,
		index:      index,
		members:    identity,
	}, nil
}

func (p *Proposal) Propose(q string) int {

	bigHashQus := big.NewInt(0).SetBytes(crypto.Keccak256([]byte(q)))
	p.nullifiers[p.index] = &nullifier{
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

func (p *Proposal) isValidVote(idx int, proofs *snark.Vote, vkString string) bool {
	if !p.checkIndex(idx) {
		return false
	}
	if nil == p.nullifiers[idx] {
		fmt.Printf("question doesn't exist (idex %d)", idx)
		return false
	}

	root := proofs.PublicSignal[0]
	nullifierHash := proofs.PublicSignal[1]
	singalHash := proofs.PublicSignal[2]
	externalNullifier := proofs.PublicSignal[3]

	bigExternalNull, _ := big.NewInt(0).SetString(externalNullifier, 10)
	if 0 != p.nullifiers[idx].hash.Cmp(bigExternalNull) {
		fmt.Printf("question doesn't match (%v)/(%v)\n", p.nullifiers[idx].hash, bigExternalNull)
		return false
	}
	if p.isVoted(idx, nullifierHash) {
		fmt.Printf("Voted already, %v\n", nullifierHash)
		return false
	}
	if !isValidOpinion(singalHash) {
		fmt.Printf("Not a valid vote hash, %v\n", singalHash)
		return false
	}
	bigRoot, _ := big.NewInt(0).SetString(root, 10)
	if !p.members.IsMember(bigRoot) {
		fmt.Printf("Not member, %v\n", root)
		return false
	}

	return snark.Verify(vkString, proofs.Proof, proofs.PublicSignal)
}

func (p *Proposal) Vote(idx int, proofs string, vkString string) bool {

	snarkVote := snark.Parse(proofs)
	if !p.isValidVote(idx, snarkVote, vkString) {
		fmt.Println("Not a valid vote!")
		return false
	}

	bigNullHash, _ := big.NewInt(0).SetString(snarkVote.NullifierHash, 10)
	p.nullifiers[idx].votes.records = append(p.nullifiers[idx].votes.records, bigNullHash)

	if snarkVote.PublicSignal[3] == HASH_YES {
		p.nullifiers[idx].votes.opinion = append(p.nullifiers[idx].votes.opinion, true)
	} else {
		p.nullifiers[idx].votes.opinion = append(p.nullifiers[idx].votes.opinion, false)
	}

	return true
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

func (p *Proposal) Remove(idx int) {
	if !p.checkIndex(idx) {
		return
	}
	delete(p.nullifiers, idx)
}

func (p *Proposal) Close(idx int) {
	if !p.checkIndex(idx) {
		return
	}
	p.nullifiers[idx].votes.finished = true
}

func (p *Proposal) HasProposal(q string) int {

	for i, e := range p.nullifiers {
		if e.content == q {
			return i
		}
	}
	return -1
}

func (p *Proposal) HasProposalByHash(hash *big.Int) int {

	for i, e := range p.nullifiers {
		if 0 == e.hash.Cmp(hash) {
			return i
		}
	}
	return -1
}

func (p *Proposal) GetProposal(idx int) *nullifier {
	if p.checkIndex(idx) {
		return nil
	}
	return p.nullifiers[idx]
}

func (p *Proposal) GetProposalByHash(hash *big.Int) *nullifier {
	idx := p.HasProposalByHash(hash)
	return p.GetProposal(idx)
}

func (p *Proposal) GetCurrentIdex() int {
	return p.index
}

func (p *Proposal) checkIndex(idx int) bool {
	if idx > p.GetCurrentIdex() {
		fmt.Printf("index (%d) is incorrect, max index is %d", idx, p.index)
		return false
	}
	return true
}
