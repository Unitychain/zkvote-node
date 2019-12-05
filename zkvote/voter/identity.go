package voter

import (
	"fmt"
	"math/big"
)

// Identity ...
type Identity struct {
	rootHistory []*big.Int
	tree        *MerkleTree
}

const TREE_LEVEL uint8 = 10

// NewIdentity ...
func NewIdentity() (*Identity, error) {

	tree, err := NewMerkleTree(TREE_LEVEL)
	if err != nil {
		return nil, err
	}
	rootHistory := []*big.Int{tree.GetRoot()}

	// TODO: load from DHT/PubSub

	return &Identity{
		rootHistory: rootHistory,
		tree:        tree,
	}, nil
}

// Register ...
func (i *Identity) Register(idCommitment *big.Int) (int, error) {
	idx, err := i.tree.Insert(idCommitment)
	if err != nil {
		fmt.Printf("register error, ", err.Error())
		return -1, err
	}
	i.appendRoot(i.tree.GetRoot())

	return idx, nil
}

// Update ...
func (i *Identity) Update(index uint, oldIDCommitment, newIDCommitment *big.Int) error {
	err := i.tree.Update(index, oldIDCommitment, newIDCommitment)
	if err != nil {
		fmt.Printf("update id error, ", err.Error())
		return err
	}
	i.appendRoot(i.tree.GetRoot())

	return nil
}

func (i *Identity) IsMember(root *big.Int) bool {
	for _, r := range i.rootHistory {
		if 0 == r.Cmp(root) {
			return true
		}
	}
	return false
}

func (i *Identity) appendRoot(r *big.Int) {
	i.rootHistory = append(i.rootHistory, i.tree.GetRoot())
}
