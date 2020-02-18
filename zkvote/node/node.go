package node

import (
	"context"

	"github.com/ipfs/go-datastore"
	"github.com/unitychain/zkvote-node/zkvote/common/store"
	"github.com/unitychain/zkvote-node/zkvote/common/utils"
	zkp "github.com/unitychain/zkvote-node/zkvote/node/zkp_vote"
)

type Node struct {
	zkpVote *zkp.ZkpVote
	store   *store.Store
}

func NewNode(ctx context.Context, ds datastore.Batching) *Node {

	store, err := store.NewStore(nil, ds)
	if err != nil {
		utils.LogFatalf("New store error, ", err.Error())
	}

	zkp, err := zkp.NewZkpVote(store)
	if err != nil {
		utils.LogFatalf("New ZKP Vote error, ", err.Error())
	}

	return &Node{
		zkpVote: zkp,
		store:   store,
	}
}
