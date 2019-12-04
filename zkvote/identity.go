package zkvote

import (
	"crypto/sha256"
)

// Identity ...
type Identity struct {
	commitment string
}

// // IdentityIndex ...
// type IdentityIndex struct {
// 	hash [32]byte
// }

func (i *Identity) hash() [32]byte {
	return sha256.Sum256([]byte(i.commitment))
}
