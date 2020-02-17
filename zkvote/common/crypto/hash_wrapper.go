package crypto

import (
	"hash"
	"math/big"
)

type HashWrapper interface {
	hash.Hash
	Hash(arr []*big.Int) (*big.Int, error)
}
