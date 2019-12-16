package store

import (
	"github.com/unitychain/zkvote-node/zkvote/model/subject"
)

// Cache ...
type Cache struct {
	collectedSubjects subject.Map
	createdSubjects   subject.Map
}

// NewCache ...
func NewCache() (*Cache, error) {
	return &Cache{
		collectedSubjects: subject.NewMap(),
		createdSubjects:   subject.NewMap(),
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
