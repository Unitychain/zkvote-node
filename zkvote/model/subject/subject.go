package subject

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/unitychain/zkvote-node/zkvote/model/identity"
)

// Subject ...
type Subject struct {
	Title       string             `json:"title"`
	Description string             `json:"Desc"`
	Proposer    *identity.Identity `json:"proposer"`
	hash        HashHex
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
	s := Subject{Title: title, Description: description, Proposer: identity}
	s.hash = s.Hash().Hex()
	return &s
}

// NewMap ...
func NewMap() Map {
	return Map(make(map[HashHex]*Subject))
}

// HashHex .
func (s *Subject) HashHex() *HashHex {
	if 0 == len(s.hash) {
		s.hash = s.Hash().Hex()
	}
	return &s.hash
}

// Hash ...
func (s *Subject) Hash() *Hash {
	h := sha256.Sum256([]byte(s.Title + s.Description + s.Proposer.String()))
	result := Hash(h[:])
	return &result
}

// JSON ...
func (s *Subject) JSON() map[string]string {
	return map[string]string{
		"hash":        s.HashHex().String(),
		"title":       s.Title,
		"description": s.Description,
		"proposer":    s.Proposer.String(),
	}
}

// GetTitle ...
func (s *Subject) GetTitle() string {
	return s.Title
}

// GetDescription ...
func (s *Subject) GetDescription() string {
	return s.Description
}

// GetProposer ...
func (s *Subject) GetProposer() *identity.Identity {
	return s.Proposer
}
