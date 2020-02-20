package node

import (
	"context"
	"fmt"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	crypto "github.com/libp2p/go-libp2p-crypto"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	dhtopts "github.com/libp2p/go-libp2p-kad-dht/opts"
	"github.com/unitychain/zkvote-node/zkvote/common/store"
	"github.com/unitychain/zkvote-node/zkvote/common/utils"
	zkp "github.com/unitychain/zkvote-node/zkvote/node/zkp_vote"
)

const DB_NODE_ID = "nodeID"

type Node struct {
	zkpVote *zkp.ZkpVote
	store   *store.Store
}

func NewNode(ctx context.Context, ds datastore.Batching, bucketSize int) *Node {
	cmgr := connmgr.NewConnManager(1500, 2000, time.Minute)

	// Ignoring most errors for brevity
	// See echo example for more details and better implementation
	prvKey, err := loadPrivateKey(ds)
	if err != nil {
		panic(err)
	}

	opts := []libp2p.Option{libp2p.ConnectionManager(cmgr), libp2p.Identity(prvKey)}
	host, err := libp2p.New(context.Background(), opts...)
	if err != nil {
		panic(err)
	}

	d1, err := dht.New(context.Background(), host, dhtopts.BucketSize(bucketSize), dhtopts.Datastore(ds), dhtopts.Validator(store.NodeValidator{}))
	if err != nil {
		panic(err)
	}

	store, err := store.NewStore(d1, ds)
	if err != nil {
		utils.LogFatalf("New store error, %v", err.Error())
	}

	zkp, err := zkp.NewZkpVote(store)
	if err != nil {
		utils.LogFatalf("New ZKP Vote error, %v", err.Error())
	}

	return &Node{
		zkpVote: zkp,
		store:   store,
	}
}

func loadPrivateKey(ds datastore.Batching) (crypto.PrivKey, error) {
	var prvKey crypto.PrivKey
	var err error

	tmpStore, _ := store.NewStore(nil, ds)
	strKey, _ := tmpStore.GetLocal(DB_NODE_ID)
	if 0 == len(strKey) {
		utils.LogInfof("Generate a new private key!")
		prvKey, _, _ = crypto.GenerateKeyPair(crypto.ECDSA, 0)
		// priv, _, _ := crypto.GenerateKeyPair(crypto.Ed25519, 0)
		b, _ := prvKey.Raw()
		tmpStore.PutLocal(DB_NODE_ID, utils.Remove0x(utils.GetHexStringFromBytes(b)))
	} else {
		prvKey, err = crypto.UnmarshalECDSAPrivateKey(utils.GetBytesFromHexString(strKey))
		if err != nil {
			return nil, fmt.Errorf("unmarshal private key error, %v", err)
		}
	}
	b, _ := prvKey.GetPublic().Bytes()
	utils.LogInfof("peer pub key: %v", utils.GetHexStringFromBytes(b))

	return prvKey, err
}
