package identity

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/unitychain/zkvote-node/zkvote/model/subject"
)

// Identity ...
type Identity struct {
	commitment string
}

// NewIdentity ...
func NewIdentity(commitment string) *Identity {
	return &Identity{commitment: commitment}
}

// Hash ...
type Hash []byte

// Byte ...
func (h Hash) Byte() []byte { return []byte(h) }

// Hex ...
func (h Hash) Hex() HashHex {
	return HashHex(hex.EncodeToString(h.Byte()))
}

// Index ...
type Index map[subject.HashHex]HashSet

// NewIndex ...
func NewIndex() Index {
	return Index(make(map[subject.HashHex]HashSet))
}

// HashSet ...
type HashSet map[HashHex]string

// NewHashSet ...
func NewHashSet() HashSet {
	result := HashSet(make(map[HashHex]string))
	return result
}

// HashHex ...
type HashHex string

// String ...
func (h HashHex) String() string { return string(h) }

// Hash ...
func (i *Identity) Hash() *Hash {
	h := sha256.Sum256([]byte(i.commitment))
	result := Hash(h[:])
	return &result
}
