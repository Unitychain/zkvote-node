package store

import (
	"github.com/unitychain/zkvote-node/zkvote/model/ballot"
	"github.com/unitychain/zkvote-node/zkvote/model/identity"
	"github.com/unitychain/zkvote-node/zkvote/model/subject"
)

// Cache ...
type Cache struct {
	collectedSubjects subject.Map
	createdSubjects   subject.Map
	ballotMap         map[subject.HashHex]ballot.Map
	idMap             map[subject.HashHex]identity.Set
}

// NewCache ...
func NewCache() (*Cache, error) {
	return &Cache{
		collectedSubjects: subject.NewMap(),
		createdSubjects:   subject.NewMap(),
		ballotMap:         make(map[subject.HashHex]ballot.Map),
		idMap:             make(map[subject.HashHex]identity.Set),
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

// GetBallotSet .
func (c *Cache) GetBallotSet(subHashHex subject.HashHex) ballot.Map {
	return c.ballotMap[subHashHex]
}

// InsertBallotSet .
func (c *Cache) InsertBallotSet(subHashHex subject.HashHex, ballotMap ballot.Map) {
	_, ok := c.ballotMap[subHashHex]
	if !ok {
		c.ballotMap[subHashHex] = ballot.NewMap()
	}
	c.ballotMap[subHashHex] = ballotMap
}

// InsertBallot .
func (c *Cache) InsertBallot(subHashHex subject.HashHex, ba *ballot.Ballot) {
	_, ok := c.ballotMap[subHashHex]
	if !ok {
		c.ballotMap[subHashHex] = ballot.NewMap()
	}
	c.ballotMap[subHashHex][ba.NullifierHashHex()] = ba
}

func (c *Cache) InsertIdentitySet(subHashHex subject.HashHex, idSet identity.Set) {
	_, ok := c.idMap[subHashHex]
	if !ok {
		c.idMap[subHashHex] = identity.NewSet()
	}
	c.idMap[subHashHex] = idSet
}

func (c *Cache) InsertIdentity(subHashHex subject.HashHex, id identity.Identity) {
	_, ok := c.idMap[subHashHex]
	if !ok {
		c.idMap[subHashHex] = identity.NewSet()
	}
	c.idMap[subHashHex][id] = id.String()
}

func (c *Cache) GetIdentitySet(subHashHex subject.HashHex) identity.Set {
	return c.idMap[subHashHex]
}
