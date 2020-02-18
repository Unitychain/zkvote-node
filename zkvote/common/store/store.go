package store

import (
	"context"

	"github.com/ipfs/go-datastore"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/whyrusleeping/base32"
)

// Store ...
type Store struct {
	dht *dht.IpfsDHT
	db  datastore.Batching
}

// NewStore ...
func NewStore(dht *dht.IpfsDHT, db datastore.Batching) (*Store, error) {
	return &Store{
		dht: dht,
		db:  db,
	}, nil
}

// func (s *Store) InsertSubject(HashHex)

// PutDHT ...
func (store *Store) PutDHT(k, v string) error {
	ctx := context.Background()
	err := store.dht.PutValue(ctx, k, []byte(v))
	if err != nil {
		return err
	}

	return nil
}

// GetDHT ...
func (store *Store) GetDHT(k string) ([]byte, error) {
	ctx := context.Background()

	vb, err := store.dht.GetValue(ctx, k)
	if err != nil {
		return nil, err
	}
	return vb, nil
}

// PutLocal ...
func (store *Store) PutLocal(k, v string) error {
	err := store.db.Put(mkDsKey(k), []byte(v))
	if err != nil {
		return err
	}

	return nil
}

// GetLocal ...
func (store *Store) GetLocal(k string) (string, error) {
	vb, err := store.db.Get(mkDsKey(k))
	if err != nil {
		return "", err
	}

	return string(vb), nil
}

func mkDsKey(s string) datastore.Key {
	return datastore.NewKey(base32.RawStdEncoding.EncodeToString([]byte(s)))
}
