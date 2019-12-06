package subject

import (
	"crypto/sha256"
	"encoding/hex"
)

// Subject ...
type Subject struct {
	title       string
	description string
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

// NewSubject ...
func NewSubject(title string, description string) *Subject {
	return &Subject{title: title, description: description}
}

// NewMap ...
func NewMap() Map {
	var result Map
	result = Map(make(map[HashHex]*Subject))
	return result
}

// Hash ...
func (s *Subject) Hash() *Hash {
	h := sha256.Sum256([]byte(s.title + s.description))
	result := Hash(h[:])

	return &result
}

// GetTitle ...
func (s *Subject) GetTitle() string {
	return s.title
}

// GetDescription ...
func (s *Subject) GetDescription() string {
	return s.description
}
