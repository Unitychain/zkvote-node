package identity

import (
	"crypto/sha256"
	"encoding/hex"
)

// Identity ...
type Identity string

// NewIdentity ...
func NewIdentity(commitment string) *Identity {
	// TODO: Check if commitment is a hex string
	id := Identity(commitment)
	return &id
}

// Hash ...
type Hash []byte

// Byte ...
func (h Hash) Byte() []byte { return []byte(h) }

// Hex ...
func (h Hash) Hex() HashHex {
	return HashHex(hex.EncodeToString(h.Byte()))
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
	h := sha256.Sum256([]byte(string(*i)))
	result := Hash(h[:])
	return &result
}
