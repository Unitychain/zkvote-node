package crypto

import (
	"math/big"

	"github.com/iden3/go-iden3-crypto/mimc7"
)

const BlockSize = 64
const Size = 32
const Denominator = 8

type mimc7Wrapper struct {
	data []byte
	len  uint64
}

func MiMC7New() HashWrapper {
	d := new(mimc7Wrapper)
	d.Reset()
	return d
}

func (m *mimc7Wrapper) Write(p []byte) (nn int, err error) {
	// m.len = uint64(len(p))
	// // if m.len < 64 {
	// // 	m.len = 64
	// // }
	// m.data = make([]byte, m.len)
	// copy(m.data, p)

	// // left := big.NewInt(0).SetBytes(m.data[:32])
	// // right := big.NewInt(0).SetBytes(m.data[32:])
	// // utils.LogDebugf("* %v/%v", left, right)
	return
}

func (m *mimc7Wrapper) Sum(b []byte) []byte {
	return nil
	// left := big.NewInt(0).SetBytes(m.data[:int(m.len/2)])
	// right := big.NewInt(0).SetBytes(m.data[int(m.len/2):])
	// if m.len <= 32 {
	// 	left = big.NewInt(0).SetBytes(m.data)
	// 	right = big.NewInt(0)
	// } else if m.len <= 64 {
	// 	left = big.NewInt(0).SetBytes(m.data[:32])
	// 	right = big.NewInt(0).SetBytes(m.data[32:])
	// }

	// // TODO: Div(8) is a workaround mimc7 in golang will check finite field Q
	// left = left.Div(left, big.NewInt(Denominator))
	// right = right.Div(right, big.NewInt(Denominator))

	// h, err := mimc7.Hash([]*big.Int{left, right}, nil)
	// m.Reset()
	// if err != nil {
	// 	utils.LogErrorf("mimc hash error, %v", err.Error())
	// 	return nil
	// }
	// // utils.LogDebugf("* %v/%v", left, right)
	// // utils.LogDebugf("  %v", h)
	// return h.Bytes()
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

func (m *mimc7Wrapper) Hash(arr []*big.Int) (*big.Int, error) {
	return m.hash(arr[:len(arr)-1], arr[len(arr)-1])
}

func (m *mimc7Wrapper) hash(arr []*big.Int, key *big.Int) (*big.Int, error) {
	for i, a := range arr {
		// TODO: Div(8) is a workaround, mimc7 in golang will check finite field, Q.
		arr[i] = new(big.Int).Div(a, big.NewInt(Denominator))
	}
	if nil == key {
		key = big.NewInt(0)
	}
	return mimc7.Hash(arr, key)
}
