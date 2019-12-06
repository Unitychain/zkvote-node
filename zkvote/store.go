package zkvote

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/ipfs/go-datastore"
	"github.com/unitychain/zkvote-node/zkvote/identity"
	"github.com/unitychain/zkvote-node/zkvote/subject"
	"github.com/whyrusleeping/base32"
)

// Store ...
type Store struct {
	*Node
	collectedSubjects subject.Map
	createdSubjects   subject.Map
	identityIndex     identity.Index
}

// NewStore ...
func NewStore(node *Node) (*Store, error) {
	store := &Store{
		Node:              node,
		collectedSubjects: subject.NewMap(),
		createdSubjects:   subject.NewMap(),
		identityIndex:     identity.NewIndex(),
	}

	return store, nil
}

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
