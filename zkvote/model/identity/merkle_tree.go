package identity

import (
	"fmt"
	"math"
	"math/big"

	merkletree "github.com/cbergoon/merkletree"
	hashWrapper "github.com/unitychain/zkvote-node/zkvote/service/crypto"
	"github.com/unitychain/zkvote-node/zkvote/service/utils"
)

//
//  TreeContent
//

// TreeContent ...
type TreeContent struct {
	x *big.Int
}

func NewTreeContent(value *big.Int) *TreeContent {
	return &TreeContent{value}
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
	// return t.x.String() == other.(TreeContent).x.String(), nil
	return 0 == t.x.Cmp(other.(TreeContent).x), nil
}

func (t TreeContent) String() string {
	if nil == t.x || 0 == t.x.Cmp(big.NewInt(0)) {
		return "0"
	}
	return t.x.String()
}

func (t TreeContent) Hex() string {
	if nil == t.x || 0 == t.x.Cmp(big.NewInt(0)) {
		return "0x0"
	}
	return utils.GetHexStringFromBigInt(t.x)
}

func (t TreeContent) Bytes() []byte {
	return t.x.Bytes()
}

func (t TreeContent) BigInt() *big.Int {
	return t.x
}

//
//  MerkleTree
//

// MerkleTree ...
type MerkleTree struct {
	levels    uint8
	nextIndex uint

	root       *TreeContent
	content    []merkletree.Content
	mapContent map[merkletree.Content]uint

	hashStrategy hashWrapper.HashWrapper
}

// NewMerkleTree ...
func NewMerkleTree(levels uint8) (*MerkleTree, error) {

	// create an empty tree with zeros
	var content []merkletree.Content
	numIndexes := int(math.Pow(2, float64(levels)))
	for i := 0; i < numIndexes; i++ {
		content = append(content, TreeContent{big.NewInt(0)})
	}
	tree := &MerkleTree{
		levels:       levels,
		nextIndex:    0,
		content:      content,
		hashStrategy: hashWrapper.MiMC7New(),
		mapContent:   make(map[merkletree.Content]uint),
	}

	root, err := tree.calculateRoot()
	if err != nil {
		return nil, err
	}
	tree.root = root

	utils.LogInfof("total elements %d, init root: %v", numIndexes, root)
	return tree, nil
}

// Insert : insert into to the merkle tree
func (m *MerkleTree) Insert(value *TreeContent) (int, error) {
	if value == nil {
		return -1, fmt.Errorf("invalid input value")
	}
	if m.IsExisted(value) {
		return -1, fmt.Errorf("value existed, %v", value)
	}

	currentIndex := m.nextIndex
	m.addContent(currentIndex, value)

	root, err := m.calculateRoot()
	if err != nil {
		return -1, err
	}
	m.root = root
	m.nextIndex++
	utils.LogInfof("new merkle root: %v", root)

	return int(currentIndex), nil
}

// Update : update a leaf of this merkle tree
func (m *MerkleTree) Update(index uint, oldValue, newValue *TreeContent) error {

	if b, _ := oldValue.Equals(*newValue); b {
		return fmt.Errorf("old and new value are the same")
	}
	if !m.IsExisted(oldValue) {
		return fmt.Errorf("old value not existed, %v", oldValue)
	}
	if eq, _ := m.content[index].Equals(*oldValue); !eq {
		// utils.LogErrorf("value of the index is not matched old value.")
		return fmt.Errorf("value of the index is not matched old value")
	}

	m.addContent(index, newValue)
	root, err := m.calculateRoot()
	if err != nil {
		return err
	}
	m.root = root
	utils.LogInfof("new root: %v", root)

	return nil
}

// GetRoot : get current merkle root
func (m *MerkleTree) GetRoot() *TreeContent {
	return m.root
}

// GetPath : get merkle path of a leaf
func (m *MerkleTree) GetPath(value *TreeContent) []byte {

	idx := m.GetIndexByValue(value)
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
func (m *MerkleTree) GetIntermediateValues(value *TreeContent) ([]*TreeContent, []int, *TreeContent) {

	var idx int
	if nil == value {
		idx = 0
	} else {
		idx = m.GetIndexByValue(value)
		if -1 == idx {
			utils.LogWarningf("Can NOT find index of value, %v", value)
			return nil, nil, nil
		}
	}

	currentIdx := idx
	imv := make([]*TreeContent, m.levels)
	imi := make([]int, m.levels)
	tree := make([][]*TreeContent, m.levels)
	for i := 0; i < int(m.levels); i++ {

		numElemofLevel := int(math.Pow(2, float64(int(m.levels)-i)))
		valuesOfLevel := make([]*TreeContent, numElemofLevel)
		imi[i] = currentIdx % 2
		for j := 0; j < numElemofLevel; j++ {
			if 0 == i {
				h, _ := m.content[j].CalculateHash()
				valuesOfLevel[j] = &TreeContent{big.NewInt(0).SetBytes(h)}

			} else {
				h, err := m.hashStrategy.Hash([]*big.Int{tree[i-1][2*j].x, tree[i-1][2*j+1].x, big.NewInt(0)})
				if err != nil {
					utils.LogFatalf("ERROR: calculate mimc7 error, %v", err.Error())
					return nil, nil, nil
				}
				valuesOfLevel[j] = &TreeContent{h}
			}
		}

		if 0 == currentIdx%2 {
			imv[i] = valuesOfLevel[currentIdx+1]
		} else {
			imv[i] = valuesOfLevel[currentIdx-1]
		}

		tree[i] = valuesOfLevel
		currentIdx = int(currentIdx / 2)
	}
	root, err := m.hashStrategy.Hash([]*big.Int{tree[m.levels-1][0].x, tree[m.levels-1][1].x, big.NewInt(0)})
	if err != nil {
		utils.LogFatalf("ERROR: calculate root through mimc7 error, %v", err.Error())
		return nil, nil, nil
	}
	return imv, imi, &TreeContent{root}
}

// GetAllContent .
func (m *MerkleTree) GetAllContent() []*TreeContent {
	lenWithContent := len(m.mapContent)
	ids := make([]*TreeContent, lenWithContent)
	for i := 0; i < lenWithContent; i++ {
		ids[i] = &TreeContent{m.content[i].(*TreeContent).x}
	}
	return ids
}

// IsExisted ...
func (m *MerkleTree) IsExisted(value *TreeContent) bool {
	if 0 <= m.GetIndexByValue(value) {
		return true
	}
	return false
}

// GetIndexByValue .
func (m *MerkleTree) GetIndexByValue(value *TreeContent) int {
	for i, c := range m.content {
		if eq, _ := c.Equals(*value); eq {
			utils.LogDebugf("Got index, %d", i)
			return i
		}
	}
	return -1
}

// Len .
func (m *MerkleTree) Len() int {
	return int(m.nextIndex)
}

//
// Internal functions
//

func (m *MerkleTree) addContent(idx uint, value *TreeContent) {
	m.content[idx] = value
	m.mapContent[value] = idx
}

func (m *MerkleTree) calculateRoot() (*TreeContent, error) {

	_, _, root := m.GetIntermediateValues(nil)
	return root, nil
}
