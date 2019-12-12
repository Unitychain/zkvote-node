package store

import (
	"github.com/unitychain/zkvote-node/zkvote/pubsubhandler/subject"
	"github.com/unitychain/zkvote-node/zkvote/pubsubhandler/voter"
)

// Cache ...
type Cache struct {
	collectedSubjects subject.Map
	createdSubjects   subject.Map
	identityIndex     voter.Index
}

// NewCache ...
func NewCache() (*Cache, error) {
	return &Cache{
		collectedSubjects: subject.NewMap(),
		createdSubjects:   subject.NewMap(),
		identityIndex:     voter.NewIndex(),
	}, nil
}

// InsertColletedSubject .
func (c *Cache) InsertColletedSubject(k subject.HashHex, v *subject.Subject) {
	c.collectedSubjects[k] = v
}

//GetCollectedSubjects .
func (c *Cache) GetCollectedSubjects() subject.Map {
	return c.collectedSubjects
}

// GetACollectedSubject ...
func (c *Cache) GetACollectedSubject(k subject.HashHex) *subject.Subject {
	return c.collectedSubjects[k]
}

// InsertCreatedSubject .
func (c *Cache) InsertCreatedSubject(k subject.HashHex, v *subject.Subject) {
	c.createdSubjects[k] = v
}

//GetCreatedSubjects .
func (c *Cache) GetCreatedSubjects() subject.Map {
	return c.createdSubjects
}

// GetACreatedSubject ...
func (c *Cache) GetACreatedSubject(k subject.HashHex) *subject.Subject {
	return c.createdSubjects[k]
}

// InsertIDIndex .
func (c *Cache) InsertIDIndex(k subject.HashHex, v voter.HashSet) {
	c.identityIndex[k] = v
}

//GetIDIndexes .
func (c *Cache) GetIDIndexes() voter.Index {
	return c.identityIndex
}

// GetAIDIndex ...
func (c *Cache) GetAIDIndex(k subject.HashHex) voter.HashSet {
	return c.identityIndex[k]
}
