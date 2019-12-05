package voter

import (
	"fmt"
	"math"
	"math/big"

	merkletree "github.com/cbergoon/merkletree"
	mimc7 "github.com/iden3/go-iden3-crypto/mimc7"
	mimcwrapper "github.com/unitychain/zkvote-node/zkvote/crypto"
	"github.com/unitychain/zkvote-node/zkvote/utils"
)

//
//  TreeContent
//

// TreeContent ...
type TreeContent struct {
	x *big.Int
}

//CalculateHash hashes the values of a TreeContent
func (t TreeContent) CalculateHash() ([]byte, error) {
	return t.x.Bytes(), nil
	// if 0 == t.x.Cmp(big.NewInt(0)) {
	// 	return []byte{0}, nil
	// }
	// // return mimc7.MIMC7Hash(t.x, big.NewInt(0)).Bytes(), nil

	// return crypto.Keccak256(t.x.Bytes()), nil
}

//Equals tests for equality of two Contents
func (t TreeContent) Equals(other merkletree.Content) (bool, error) {
	return 0 == t.x.Cmp(other.(TreeContent).x), nil
}

//
//  MerkleTree
//

// MerkleTree ...
type MerkleTree struct {
	levels    uint8
	nextIndex uint

	root    *big.Int
	content []merkletree.Content
}

// NewMerkleTree ...
func NewMerkleTree(levels uint8) (*MerkleTree, error) {

	// create an empty tree with zeros
	var content []merkletree.Content
	numIndexes := int(math.Pow(2, float64(levels)))
	for i := 0; i < numIndexes; i++ {
		content = append(content, TreeContent{big.NewInt(0)})
	}

	root, err := calculateRoot(content)
	if err != nil {
		return nil, err
	}

	utils.LogInfof("total elements %d, init root: %v\n", numIndexes, root)
	return &MerkleTree{
		levels:    levels,
		root:      root,
		nextIndex: 0,
		content:   content,
	}, nil
}

// Insert : insert into to the merkle tree
func (m *MerkleTree) Insert(value *big.Int) (int, error) {

	if m.IsExisted(value) {
		return -1, fmt.Errorf("value existed, %v", value)
	}

	currentIndex := m.nextIndex
	m.content[currentIndex] = TreeContent{value}

	root, err := calculateRoot(m.content)
	if err != nil {
		return -1, err
	}
	m.root = root
	m.nextIndex++
	utils.LogInfof("new merkle root: %v", root)

	return int(currentIndex), nil
}

// Update : update a leaf of this merkle tree
func (m *MerkleTree) Update(index uint, oldValue, newValue *big.Int) error {

	if 0 == oldValue.Cmp(newValue) {
		return fmt.Errorf("old and new value are the same")
	}
	if !m.IsExisted(oldValue) {
		return fmt.Errorf("old value not existed, %v", oldValue)
	}
	if eq, _ := m.content[index].Equals(TreeContent{oldValue}); !eq {
		// utils.LogErrorf("value of the index is not matched old value.")
		return fmt.Errorf("value of the index is not matched old value")
	}

	m.content[index] = TreeContent{newValue}
	root, err := calculateRoot(m.content)
	if err != nil {
		return err
	}
	m.root = root
	utils.LogInfof("new root: %v", root)

	return nil
}

// GetRoot : get current merkle root
func (m *MerkleTree) GetRoot() *big.Int {
	return m.root
}

// GetPath : get merkle path of a leaf
func (m *MerkleTree) GetPath(value *big.Int) []byte {

	idx := m.getIndexByValue(value)
	if idx == -1 {
		utils.LogWarningf("Can NOT find index of value, %v", value)
		return nil
	}
	paths := make([]byte, m.levels)

	for i := 0; i < int(m.levels); i++ {
		if 0 == idx%2 {
			paths[i] = 0
		} else {
			paths[i] = 1
		}
		idx /= 2
	}

	return paths
}

// GetIntermediateValues : get all intermediate values of a leaf
func (m *MerkleTree) GetIntermediateValues(value *big.Int) []big.Int {

	idx := m.getIndexByValue(value)
	if idx == -1 {
		utils.LogWarningf("Can NOT find index of value, %v", value)
		return nil
	}
	currentIdx := idx
	imv := make([]big.Int, m.levels)

	tree := make([][]big.Int, m.levels)
	for i := 0; i < int(m.levels); i++ {

		numElemofLevel := int(math.Pow(2, float64(int(m.levels)-i)))
		valuesOfLevel := make([]big.Int, numElemofLevel)
		for j := 0; j < numElemofLevel; j++ {
			if 0 == i {
				if idx == j {
					valuesOfLevel[j] = *value
				} else {
					valuesOfLevel[j] = *m.content[j].(TreeContent).x
				}
			} else {
				h, e := mimc7.Hash([]*big.Int{&tree[i-1][2*j], &tree[i-1][2*j+1]}, nil)
				if e != nil {
					utils.LogWarningf("ERROR: calculate mimc7 error, %v", e.Error())
					return nil
				}
				valuesOfLevel[j] = *h
				// fmt.Printf("%x-%x --> %x\n", &tree[i-1][2*j], &tree[i-1][2*j+1], &valuesOfLevel[j])
			}
		}

		if 0 == currentIdx%2 {
			imv[i] = valuesOfLevel[currentIdx+1]
		} else {
			imv[i] = valuesOfLevel[currentIdx-1]
		}
		// utils.LogDebugf("%v\n", &imv[i])
		tree[i] = valuesOfLevel
		currentIdx = int(currentIdx / 2)
	}
	return imv
}

// IsExisted ...
func (m *MerkleTree) IsExisted(value *big.Int) bool {
	if 0 <= m.getIndexByValue(value) {
		return true
	}
	return false
}

//
// Internal functions
//
func (m *MerkleTree) getIndexByValue(value *big.Int) int {
	for i, c := range m.content {
		if eq, _ := c.Equals(TreeContent{value}); eq {
			utils.LogDebugf("Got index, %d", i)
			return i
		}
	}
	return -1
}

func calculateRoot(content []merkletree.Content) (*big.Int, error) {

	var hashStrategy = mimcwrapper.MiMC7New
	tree, err := merkletree.NewTreeWithHashStrategy(content, hashStrategy)
	if err != nil {
		utils.LogErrorf("init merkle tree error, %v", err.Error())
		return nil, err
	}

	root := big.NewInt(0).SetBytes(tree.MerkleRoot())
	return root, nil
}
