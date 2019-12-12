package store

import (
	"github.com/unitychain/zkvote-node/zkvote/pubsubhandler/identity"
	"github.com/unitychain/zkvote-node/zkvote/pubsubhandler/subject"
)

// Cache ...
type Cache struct {
	collectedSubjects subject.Map
	createdSubjects   subject.Map
	identityIndex     identity.Index
}

// NewCache ...
func NewCache() (*Cache, error) {
	return &Cache{
		collectedSubjects: subject.NewMap(),
		createdSubjects:   subject.NewMap(),
		identityIndex:     identity.NewIndex(),
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
func (c *Cache) InsertIDIndex(k string, v identity.HashSet) {
	c.identityIndex[k] = v
}

//GetIDIndexes .
func (c *Cache) GetIDIndexes() identity.Index {
	return c.identityIndex
}

// GetAIDIndex ...
func (c *Cache) GetAIDIndex(k string) identity.HashSet {
	return c.identityIndex[k]
}
