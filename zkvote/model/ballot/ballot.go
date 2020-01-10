package ballot

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/arnaucube/go-snark/externalVerif"
	"github.com/unitychain/zkvote-node/zkvote/service/utils"
)

// Ballot ...
type Ballot struct {
	Root          string                     `json:"root"`
	NullifierHash string                     `json:"nullifier_hash"`
	Proof         *externalVerif.CircomProof `json:"proof"`
	PublicSignal  []string                   `json:"public_signal"` //root, nullifiers_hash, signal_hash, external_nullifier
}

// Hash ...
type Hash []byte

// HashHex ...
type HashHex string

// NewBallot ...
func NewBallot(proof string) (*Ballot, error) {
	if 0 == len(proof) {
		utils.LogWarningf("invalid input:\n %s", proof)
		return nil, fmt.Errorf("invalid input")
	}

	var b Ballot
	err := json.Unmarshal([]byte(proof), &b)
	if err != nil {
		utils.LogErrorf("parse proof: unmarshal error %v", err.Error())
		return nil, err
	}
	return &b, nil
}

// Byte ...
func (b *Ballot) Byte() ([]byte, error) {
	return json.Marshal(b)
}

// Byte ...
func (h Hash) Byte() []byte { return []byte(h) }

// Hash ...
func (b *Ballot) Hash() *Hash {
	bByte, _ := b.Byte()
	h := sha256.Sum256(bByte)
	result := Hash(h[:])
	return &result
}

// Hex ...
func (h Hash) Hex() HashHex {
	return HashHex(hex.EncodeToString(h.Byte()))
}

// Map ...
type Map map[HashHex]*Ballot

// NewMap ...
func NewMap() Map {
	return Map(make(map[HashHex]*Ballot))
}