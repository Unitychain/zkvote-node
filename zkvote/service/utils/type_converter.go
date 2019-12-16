package utils

import (
	h "encoding/hex"
	"math/big"
	"strings"
)

// Remove0x ...
func Remove0x(key string) string {
	if strings.HasPrefix(key, "0x") {
		return key[2:len(key)]
	}
	return key
}

// Prepend0x ...
func Prepend0x(key string) string {
	if strings.HasPrefix(key, "0x") {
		return key
	}
	return "0x" + key
}

func GetBigIntFromHexString(hex string) *big.Int {
	b, _ := big.NewInt(0).SetString(Remove0x(hex), 16)
	return b
}

func GetBytesFromHexString(hex string) []byte {
	b, _ := h.DecodeString(hex)
	return b
}
