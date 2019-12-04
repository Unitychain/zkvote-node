package zkvote

import (
	"crypto/sha256"
)

// Subject ...
type Subject struct {
	title       string
	description string
}

func (s *Subject) hash() [32]byte {
	return sha256.Sum256([]byte(s.title + s.description))
}
