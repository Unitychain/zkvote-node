package crypto

import (
	"hash"
	"math/big"

	"github.com/iden3/go-iden3-crypto/mimc7"
	"github.com/unitychain/zkvote-node/zkvote/utils"
)

const BlockSize = 64
const Size = 32

type mimc7Wrapper struct {
	data []byte
	len  uint64
}

func MiMC7New() hash.Hash {
	d := new(mimc7Wrapper)
	d.Reset()
	return d
}

func (m *mimc7Wrapper) Write(p []byte) (nn int, err error) {
	m.len = uint64(len(p))
	if m.len < 64 {
		m.len = 64
	}
	m.data = make([]byte, m.len)
	copy(m.data, p)

	// left := big.NewInt(0).SetBytes(m.data[:32])
	// right := big.NewInt(0).SetBytes(m.data[32:])
	// fmt.Printf("- %v/%v\n", left, right)
	return
}

func (m *mimc7Wrapper) Sum(b []byte) []byte {
	left := big.NewInt(0).SetBytes(m.data[:int(m.len/2)])
	right := big.NewInt(0).SetBytes(m.data[int(m.len/2):])
	h, err := mimc7.Hash([]*big.Int{left, right}, nil)
	m.Reset()
	if err != nil {
		utils.LogErrorf("mimc hash error, %v", err.Error())
		return nil
	}
	// fmt.Printf("  %v\n", h)
	return h.Bytes()
}

// Reset resets the Hash to its initial state.
func (m *mimc7Wrapper) Reset() {
	m.data = nil
	m.len = 0
}

// Size returns the number of bytes Sum will return.
func (m *mimc7Wrapper) Size() int { return Size }

// BlockSize returns the hash's underlying block size.
// The Write method must be able to accept any amount
// of data, but it may operate more efficiently if all writes
// are a multiple of the block size.
func (m *mimc7Wrapper) BlockSize() int { return BlockSize }
