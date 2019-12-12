package store

import (
	"bufio"
	"context"
	"fmt"
	"os"

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
func (store *Store) PutDHT() error {
	ctx := context.Background()

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Key: ")
	scanner.Scan()
	k := scanner.Text()
	fmt.Println("Input key: ", k)

	fmt.Print("Value: ")
	scanner.Scan()
	v := scanner.Text()
	vb := []byte(v)
	fmt.Println("Input value: ", v)

	err := store.dht.PutValue(ctx, k, vb)
	if err != nil {
		fmt.Println(err)
	}

	return nil
}

// PutLocal ...
func (store *Store) PutLocal() error {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Key: ")
	scanner.Scan()
	k := scanner.Text()
	fmt.Println("Input key: ", k)

	fmt.Print("Value: ")
	scanner.Scan()
	v := scanner.Text()
	vb := []byte(v)
	fmt.Println("Input value: ", v)

	err := store.db.Put(mkDsKey(k), vb)
	if err != nil {
		fmt.Println(err)
	}

	return nil
}

func mkDsKey(s string) datastore.Key {
	return datastore.NewKey(base32.RawStdEncoding.EncodeToString([]byte(s)))
}

// GetDHT ...
func (store *Store) GetDHT() error {
	ctx := context.Background()

	fmt.Print("Key: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	k := scanner.Text()
	fmt.Println("Input key: ", k)

	vb, err := store.dht.GetValue(ctx, k)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Value: ", string(vb))

	return nil
}

// GetLocal ...
func (store *Store) GetLocal() error {
	fmt.Print("Key: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	k := scanner.Text()
	fmt.Println("Input key: ", k)

	vb, err := store.db.Get(mkDsKey(k))
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Value: ", string(vb))

	return nil
}