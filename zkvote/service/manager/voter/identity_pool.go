package voter

import (
	. "github.com/unitychain/zkvote-node/zkvote/model/identity"
	"github.com/unitychain/zkvote-node/zkvote/service/utils"
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

	utils.LogDebugf("Root: %v", i.tree.GetRoot())
	return idx, nil
}

// Overwrite .
// return total len and error
func (i *IdentityPool) Overwrite(commitmentSet []*IdPathElement) (int, error) {

	// backup tree and history
	bckTree := i.tree
	bckRootHistory := i.rootHistory

	// Iniitalize a new merkle tree
	tree, err := NewMerkleTree(TREE_LEVEL)
	if err != nil {
		return i.tree.Len(), err
	}
	rootHistory := []*TreeContent{tree.GetRoot()}
	i.tree = tree
	i.rootHistory = rootHistory

	// insert values to the new merkle tree
	var idx int = 0
	for _, e := range commitmentSet {
		idx, err = i.InsertIdc(e)
		if err != nil {
			break
		}
	}

	// error handling, recovery
	if err != nil {
		i.tree = bckTree
		i.rootHistory = bckRootHistory
		return i.tree.Len(), err
	}

	return idx + 1, nil
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
