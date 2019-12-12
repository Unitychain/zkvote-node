package zkvote

import (
	"context"
	"sync"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/unitychain/zkvote-node/zkvote/store"
)

type Context struct {
	Mutex *sync.RWMutex
	Host  host.Host
	Store *store.Store
	Cache *store.Cache
	Ctx   *context.Context
}

func NewContext(mutex *sync.RWMutex, host host.Host, store *store.Store, cache *store.Cache, ctx *context.Context) *Context {
	return &Context{
		Mutex: mutex,
		Host:  host,
		Store: store,
		Cache: cache,
		Ctx:   ctx,
	}
}
