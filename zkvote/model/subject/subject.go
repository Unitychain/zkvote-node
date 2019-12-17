package subject

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/unitychain/zkvote-node/zkvote/model/identity"
)

// Subject ...
type Subject struct {
	title       string
	description string
	proposer    *identity.Identity
}

// Hash ...
type Hash []byte

// Byte ...
func (h Hash) Byte() []byte { return []byte(h) }

// Hex ...
func (h Hash) Hex() HashHex {
	return HashHex(hex.EncodeToString(h.Byte()))
}

// Map ...
type Map map[HashHex]*Subject

// HashHex ...
type HashHex string

// String ...
func (h HashHex) String() string { return string(h) }

// Hash ...
func (h HashHex) Hash() Hash {
	result, _ := hex.DecodeString(string(h))
	return Hash(result)
}

// NewSubject ...
func NewSubject(title string, description string, identity *identity.Identity) *Subject {
	return &Subject{title: title, description: description, proposer: identity}
}

// NewMap ...
func NewMap() Map {
	return Map(make(map[HashHex]*Subject))
}

// Hash ...
func (s *Subject) Hash() *Hash {
	h := sha256.Sum256([]byte(s.title + s.description + s.proposer.String()))
	result := Hash(h[:])
	return &result
}

// JSON ...
func (s *Subject) JSON() map[string]string {
	return map[string]string{
		"hash":        s.Hash().Hex().String(),
		"title":       s.title,
		"description": s.description,
		"proposer":    s.proposer.String(),
	}
}

// GetTitle ...
func (s *Subject) GetTitle() string {
	return s.title
}

// GetDescription ...
func (s *Subject) GetDescription() string {
	return s.description
}

// GetProposer ...
func (s *Subject) GetProposer() identity.Identity {
	return *s.proposer
}
