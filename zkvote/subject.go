package zkvote

import (
	"crypto/sha256"
	"encoding/hex"
)

// Subject ...
type Subject struct {
	title       string
	description string
}

// SubjectHash ...
type SubjectHash struct {
	hash []byte
}

// SubjectMap ...
type SubjectMap struct {
	Map map[SubjectHashHex]*Subject
}

// SubjectHashHex ...
type SubjectHashHex struct {
	hex string
}

func (s *Subject) hash() *SubjectHash {
	h := sha256.Sum256([]byte(s.title + s.description))
	return &SubjectHash{hash: h[:]}
}

func (s *SubjectHash) hex() SubjectHashHex {
	return SubjectHashHex{hex: hex.EncodeToString(s.hash)}
}
