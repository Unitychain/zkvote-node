package identity

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

const idCommitment string = "17610192990552485611214212559447309706539616482639833145108503521837267798810"

func TestInitValue(t *testing.T) {
	tree, err := NewMerkleTree(10)
	assert.Nil(t, err, "new merkle tree instance error")

	expectedRoot, _ := big.NewInt(0).SetString("3045810468960591087191210641638281907572011303565471692703782899879208219595", 10)
	assert.Equal(t, expectedRoot, tree.GetRoot().BigInt())
}

func TestInsert(t *testing.T) {
	tree, err := NewMerkleTree(10)
	assert.Nil(t, err, "new merkle tree instance error")

	idc, _ := big.NewInt(0).SetString(idCommitment, 10)
	idx, err := tree.Insert(&TreeContent{idc})
	assert.Nil(t, err, "insert error")
	assert.Equal(t, 0, idx)

	expectedRoot, _ := big.NewInt(0).SetString("1603056697422863699573935817849018482475219731925672640724433076363786113", 10)
	assert.Equal(t, expectedRoot, tree.GetRoot().BigInt())
}

func TestInsert_10IDs(t *testing.T) {
	tree, err := NewMerkleTree(10)
	assert.Nil(t, err, "new merkle tree instance error")

	for i := 0; i < 10; i++ {
		idc, _ := big.NewInt(0).SetString(fmt.Sprintf("%d", i+1), 10)
		idx, err := tree.Insert(&TreeContent{idc})
		tree.GetIntermediateValues(&TreeContent{idc})

		assert.Nil(t, err, "insert error")
		assert.Equal(t, i, idx)
	}
	assert.Equal(t, "1445582704315932794177642993412188437980560505275108425483448780308059834769", tree.GetRoot().String())
	// root := tree.GetRoot().String()
	// fmt.Println("root, ", root)
}

func TestInsert_Double(t *testing.T) {
	tree, err := NewMerkleTree(10)
	assert.Nil(t, err, "new identity instance error")

	idc, _ := big.NewInt(0).SetString(idCommitment, 10)
	idx, err := tree.Insert(&TreeContent{idc})
	assert.Nil(t, err, "Insert error")
	assert.Equal(t, 0, idx)

	idx, err = tree.Insert(&TreeContent{idc})
	assert.NotNil(t, err, "should not Insert successfully")
}

func TestTreeUpdate(t *testing.T) {
	tree, err := NewMerkleTree(10)
	assert.Nil(t, err, "new identity instance error")

	idc, _ := big.NewInt(0).SetString(idCommitment, 10)
	idx, err := tree.Insert(&TreeContent{idc})
	assert.Nil(t, err, "Insert error")
	assert.Equal(t, 0, idx)

	err = tree.Update(uint(idx), &TreeContent{idc}, &TreeContent{big.NewInt(100)})
	assert.Nil(t, err, "update error")
	assert.Equal(t, "5860034871856545585778554733050920915757269722014984975581566802595274325429", tree.GetRoot().String())

}

func TestTreeUpdate_IncorrectIdx(t *testing.T) {
	tree, err := NewMerkleTree(10)
	assert.Nil(t, err, "new identity instance error")

	idc, _ := big.NewInt(0).SetString(idCommitment, 10)
	idx, err := tree.Insert(&TreeContent{idc})
	assert.Nil(t, err, "Insert error")
	assert.Equal(t, 0, idx)

	err = tree.Update(1, &TreeContent{idc}, &TreeContent{big.NewInt(100)})
	assert.NotNil(t, err, "update error")
}

func TestTreeUpdate_IncorrectContent(t *testing.T) {
	tree, err := NewMerkleTree(10)
	assert.Nil(t, err, "new identity instance error")

	idc, _ := big.NewInt(0).SetString(idCommitment, 10)
	idx, err := tree.Insert(&TreeContent{idc})
	assert.Nil(t, err, "Insert error")
	assert.Equal(t, 0, idx)

	err = tree.Update(uint(idx), &TreeContent{big.NewInt(100)}, &TreeContent{big.NewInt(100)})
	assert.NotNil(t, err, "update error")
}
