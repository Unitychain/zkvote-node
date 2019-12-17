package voter

import (
	"math/big"

	. "github.com/unitychain/zkvote-node/zkvote/model/identity"
)

// IdentityPool ...
type IdentityPool struct {
	rootHistory []*big.Int
	tree        *MerkleTree
}

const TREE_LEVEL uint8 = 10

// NewIdentityPool ...
func NewIdentityPool() (*IdentityPool, error) {
	return NewIdentityPoolWithTreeLevel(TREE_LEVEL)
}

// NewIdentityPoolWithTreeLevel ...
func NewIdentityPoolWithTreeLevel(treeLevel uint8) (*IdentityPool, error) {
	tree, err := NewMerkleTree(treeLevel)
	if err != nil {
		return nil, err
	}
	rootHistory := []*big.Int{tree.GetRoot()}

	// TODO: load from DHT/PubSub

	return &IdentityPool{
		rootHistory: rootHistory,
		tree:        tree,
	}, nil
}

// Insert : register id
func (i *IdentityPool) Insert(idCommitment *big.Int) (int, error) {
	idx, err := i.tree.Insert(idCommitment)
	if err != nil {
		// utils.LogErrorf("register error, %v", err.Error())
		return -1, err
	}
	i.appendRoot(i.tree.GetRoot())

	return idx, nil
}

// Update : update id
func (i *IdentityPool) Update(index uint, oldIDCommitment, newIDCommitment *big.Int) error {
	err := i.tree.Update(index, oldIDCommitment, newIDCommitment)
	if err != nil {
		// utils.LogErrorf("update id error, %v", err.Error())
		return err
	}
	i.appendRoot(i.tree.GetRoot())

	return nil
}

// IsMember : check if the merkle root is in the root list or not
func (i *IdentityPool) IsMember(root *big.Int) bool {
	for _, r := range i.rootHistory {
		if 0 == r.Cmp(root) {
			return true
		}
	}
	return false
}

// HasRegistered .
func (i *IdentityPool) HasRegistered(idc *big.Int) bool {
	return i.tree.IsExisted(idc)
}

// GetAllIds .
func (i *IdentityPool) GetAllIds() []*big.Int {
	return i.tree.GetAllContent()
}

//
// Internal functions
//
func (i *IdentityPool) appendRoot(r *big.Int) {
	i.rootHistory = append(i.rootHistory, i.tree.GetRoot())
}
