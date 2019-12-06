package zkvote

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/unitychain/zkvote-node/zkvote/subject"
)

// Identity ...
type Identity struct {
	commitment []byte
}

// IdentityHash ...
type IdentityHash struct {
	hash []byte
}

// IdentityIndex ...
type IdentityIndex struct {
	Index map[subject.HashHex]*IdentityHashSet
}

// IdentityHashSet ...
type IdentityHashSet struct {
	set map[IdentityHashHex]string
}

// IdentityHashHex ...
type IdentityHashHex struct {
	hex string
}

func (i *Identity) hash() *IdentityHash {
	h := sha256.Sum256([]byte(i.commitment))
	return &IdentityHash{hash: h[:]}
}

func (i *IdentityHash) hex() IdentityHashHex {
	return IdentityHashHex{hex: hex.EncodeToString(i.hash)}
}
