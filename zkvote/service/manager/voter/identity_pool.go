package voter

import (
	. "github.com/unitychain/zkvote-node/zkvote/model/identity"
)

// IdentityPool ...
type IdentityPool struct {
	rootHistory []*TreeContent
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
	rootHistory := []*TreeContent{tree.GetRoot()}

	return &IdentityPool{
		rootHistory: rootHistory,
		tree:        tree,
	}, nil
}

// InsertIdc : register id
func (i *IdentityPool) InsertIdc(idCommitment *IdPathElement) (int, error) {
	c := idCommitment.Content()
	idx, err := i.tree.Insert(&c)
	if err != nil {
		return -1, err
	}
	i.appendRoot(i.tree.GetRoot())

	return idx, nil
}

// Update : update id
func (i *IdentityPool) Update(index uint, oldIDCommitment, newIDCommitment *IdPathElement) error {
	old, new := oldIDCommitment.Content(), newIDCommitment.Content()
	err := i.tree.Update(index, &old, &new)
	if err != nil {
		return err
	}
	i.appendRoot(i.tree.GetRoot())

	return nil
}

// IsMember : check if the merkle root is in the root list or not
func (i *IdentityPool) IsMember(root *IdPathElement) bool {
	for _, r := range i.rootHistory {
		if b, _ := r.Equals(root.Content()); b {
			return true
		}
	}
	return false
}

// HasRegistered .
func (i *IdentityPool) HasRegistered(idc *IdPathElement) bool {
	c := idc.Content()
	return i.tree.IsExisted(&c)
}

// GetAllIds .
func (i *IdentityPool) GetAllIds() []*IdPathElement {
	treeContents := i.tree.GetAllContent()
	elements := make([]*IdPathElement, len(treeContents))
	for i, c := range treeContents {
		elements[i] = NewIdPathElement(c)
	}
	return elements
}

// GetIndex .
func (i *IdentityPool) GetIndex(value *IdPathElement) int {
	c := value.Content()
	return i.tree.GetIndexByValue(&c)
}

// GetIdentityTreePath .
func (i *IdentityPool) GetIdentityTreePath(value *IdPathElement) ([]*IdPathElement, []int, *IdPathElement) {
	c := value.Content()
	inters, interIdxs, root := i.tree.GetIntermediateValues(&c)
	elements := make([]*IdPathElement, len(inters))
	for i, c := range inters {
		elements[i] = NewIdPathElement(c)
	}
	return elements, interIdxs, NewIdPathElement(root)
}

//
// Internal functions
//
func (i *IdentityPool) appendRoot(r *TreeContent) {
	i.rootHistory = append(i.rootHistory, i.tree.GetRoot())
}
