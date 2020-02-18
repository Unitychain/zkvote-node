package zkp_vote

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/arnaucube/go-snark/externalVerif"
	"github.com/unitychain/zkvote-node/zkvote/common/store"
	"github.com/unitychain/zkvote-node/zkvote/common/utils"
	"github.com/unitychain/zkvote-node/zkvote/snark"

	s "github.com/unitychain/zkvote-node/zkvote/model/subject"
)

const ZKPVOTE_DHT_KEY = "zkp-votes"

type RollupProof struct {
	Root         string                     `json:"root"`
	Ballots      []int                      `json:"ballots"`
	Proof        *externalVerif.CircomProof `json:"proof"`
	PublicSignal []string                   `json:"public_signal"`
}

type ZkpVote struct {
	votes *Votes
}

func NewZkpVote(s *store.Store) (*ZkpVote, error) {

	v, err := loadDataFromDHT(s)
	if err != nil {
		return nil, err
	}

	return &ZkpVote{
		votes: v,
	}, nil
}

func loadDataFromDHT(s *store.Store) (*Votes, error) {
	utils.LogDebugf("ZKPVote: load DHT")

	value, err := s.GetDHT(ZKPVOTE_DHT_KEY)
	if err != nil {
		utils.LogErrorf("Get DHT data error, %v", err)
		return nil, err
	}

	if len(value) == 0 {
		return NewVotes()
	}
	return NewVotesWithSerializedString(string(value))
}

func (z *ZkpVote) Rollup(subHashHex s.HashHex, prvRoot *big.Int, rProof *RollupProof, vkString string) error {
	if !z.votes.IsRootMatched(prvRoot) {
		utils.LogErrorf("Not match current root %v/%v", z.votes.Root, prvRoot)
		return fmt.Errorf("Not match current root %v/%v", z.votes.Root, prvRoot)
	}
	if b, e := z.isValidProof(subHashHex, rProof, vkString); !b {
		utils.LogErrorf("Not a valid proof, ", e)
		return e
	}

	ballot_Yes, _ := strconv.Atoi(rProof.PublicSignal[1])
	ballot_No, _ := strconv.Atoi(rProof.PublicSignal[2])
	return z.votes.Update(subHashHex, []int{ballot_Yes, ballot_No})
}

func (z *ZkpVote) isValidProof(subj s.HashHex, rProof *RollupProof, vkString string) (bool, error) {
	if 0 == len(vkString) {
		utils.LogWarningf("invalid input: %s", vkString)
		return false, fmt.Errorf("vk string is empty")
	}

	newRoot := rProof.PublicSignal[0]
	ballot_Yes, _ := strconv.Atoi(rProof.PublicSignal[1])
	ballot_No, _ := strconv.Atoi(rProof.PublicSignal[2])

	bigNewRoot, _ := big.NewInt(0).SetString(newRoot, 10)
	bigRoot, _ := big.NewInt(0).SetString(rProof.Root, 10)
	if 0 != bigRoot.Cmp(bigNewRoot) {
		utils.LogWarningf("root is inconsistent (%v)/(%v)", rProof.Root, bigNewRoot)
		return false, fmt.Errorf(fmt.Sprintf("root is inconsistent (%v)/(%v)", rProof.Root, bigNewRoot))
	}

	if b, e := z.votes.IsValidBallotNumber(subj, []int{ballot_Yes, ballot_No}); !b {
		utils.LogWarningf("Not a valid ballot, %v", e)
		return false, e
	}

	return snark.Verify(vkString, rProof.Proof, rProof.PublicSignal), nil
}
