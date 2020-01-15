package voter

import (
	"fmt"
	"math/big"
	"testing"

	. "github.com/unitychain/zkvote-node/zkvote/model/identity"

	"github.com/stretchr/testify/assert"
)

const idCommitment string = "17610192990552485611214212559447309706539616482639833145108503521837267798810"

func TestRegister(t *testing.T) {
	id, err := NewIdentityPoolWithTreeLevel(10)
	assert.Nil(t, err, "new identity instance error")

	idc, _ := big.NewInt(0).SetString(idCommitment, 10)
	idx, err := id.InsertIdc(NewIdPathElement(NewTreeContent(idc)))
	assert.Nil(t, err, "register error")
	assert.Equal(t, 0, idx)

	expectedRoot, _ := big.NewInt(0).SetString("1603056697422863699573935817849018482475219731925672640724433076363786113", 10)
	assert.Equal(t, expectedRoot, id.tree.GetRoot().BigInt())
}

func TestRegister_10IDs(t *testing.T) {
	id, err := NewIdentityPoolWithTreeLevel(10)
	assert.Nil(t, err, "new identity instance error")

	for i := 0; i < 10; i++ {
		idc, _ := big.NewInt(0).SetString(fmt.Sprintf("%d", 100*i+1), 10)
		idx, err := id.InsertIdc(NewIdPathElement(NewTreeContent(idc)))
		assert.Nil(t, err, "register error")
		assert.Equal(t, i, idx)
	}
}

func TestRegister_Double(t *testing.T) {
	id, err := NewIdentityPoolWithTreeLevel(10)
	assert.Nil(t, err, "new identity instance error")

	idc, _ := big.NewInt(0).SetString(idCommitment, 10)
	idx, err := id.InsertIdc(NewIdPathElement(NewTreeContent(idc)))
	assert.Nil(t, err, "register error")
	assert.Equal(t, 0, idx)

	idx, err = id.InsertIdc(NewIdPathElement(NewTreeContent(idc)))
	assert.NotNil(t, err, "should not register successfully")
}

func TestUpdate(t *testing.T) {
	id, err := NewIdentityPoolWithTreeLevel(10)
	assert.Nil(t, err, "new identity instance error")

	idc, _ := big.NewInt(0).SetString(idCommitment, 10)
	idx, err := id.InsertIdc(NewIdPathElement(NewTreeContent(idc)))
	assert.Nil(t, err, "register error")
	assert.Equal(t, 0, idx)

	err = id.Update(uint(idx), NewIdPathElement(NewTreeContent(idc)), NewIdPathElement(NewTreeContent(big.NewInt(100))))
	assert.Nil(t, err, "update error")
}

func TestUpdate_IncorrectIdx(t *testing.T) {
	id, err := NewIdentityPoolWithTreeLevel(10)
	assert.Nil(t, err, "new identity instance error")

	idc, _ := big.NewInt(0).SetString(idCommitment, 10)
	idx, err := id.InsertIdc(NewIdPathElement(NewTreeContent(idc)))
	assert.Nil(t, err, "register error")
	assert.Equal(t, 0, idx)

	err = id.Update(1, NewIdPathElement(NewTreeContent(idc)), NewIdPathElement(NewTreeContent(big.NewInt(100))))
	assert.NotNil(t, err, "update error")
}

func TestUpdate_IncorrectContent(t *testing.T) {
	id, err := NewIdentityPoolWithTreeLevel(10)
	assert.Nil(t, err, "new identity instance error")

	idc, _ := big.NewInt(0).SetString(idCommitment, 10)
	idx, err := id.InsertIdc(NewIdPathElement(NewTreeContent(idc)))
	assert.Nil(t, err, "register error")
	assert.Equal(t, 0, idx)

	err = id.Update(uint(idx), NewIdPathElement(NewTreeContent(big.NewInt(100))), NewIdPathElement(NewTreeContent(big.NewInt(100))))
	assert.NotNil(t, err, "update error")
}

func TestIsMember(t *testing.T) {
	id, err := NewIdentityPoolWithTreeLevel(10)
	assert.Nil(t, err, "new identity instance error")

	idc, _ := big.NewInt(0).SetString(idCommitment, 10)
	idx, err := id.InsertIdc(NewIdPathElement(NewTreeContent(idc)))
	assert.Nil(t, err, "register error")
	assert.Equal(t, 0, idx)

	assert.True(t, id.IsMember(NewIdPathElement(id.tree.GetRoot())))
}

func TestIsMember2(t *testing.T) {
	id, err := NewIdentityPoolWithTreeLevel(10)
	assert.Nil(t, err, "new identity instance error")

	idc, _ := big.NewInt(0).SetString(idCommitment, 10)
	idx, err := id.InsertIdc(NewIdPathElement(NewTreeContent(idc)))
	assert.Nil(t, err, "register error")
	assert.Equal(t, 0, idx)
	root1 := id.tree.GetRoot()

	idx, err = id.InsertIdc(NewIdPathElement(NewTreeContent(big.NewInt(10))))
	assert.Nil(t, err, "register error")
	assert.Equal(t, 1, idx)

	assert.True(t, id.IsMember(NewIdPathElement(root1)))
	assert.True(t, id.IsMember(NewIdPathElement(id.tree.GetRoot())))
}

func TestOverwrite(t *testing.T) {
	id, err := NewIdentityPoolWithTreeLevel(10)
	assert.Nil(t, err, "new identity instance error")

	for i := 0; i < 3; i++ {
		idc, _ := big.NewInt(0).SetString(fmt.Sprintf("%d", 100*i+1), 10)
		idx, err := id.InsertIdc(NewIdPathElement(NewTreeContent(idc)))
		assert.Nil(t, err, "register error")
		assert.Equal(t, i, idx)
	}

	commitmentSet := make([]*IdPathElement, 10)
	for i := 0; i < 10; i++ {
		idc, _ := big.NewInt(0).SetString(fmt.Sprintf("%d", 100*i+1), 10)
		commitmentSet[i] = NewIdPathElement(NewTreeContent(idc))
	}

	num, err := id.Overwrite(commitmentSet)
	assert.Nil(t, err, "overwrite error")
	assert.Equal(t, 10, num)
}
func TestOverwrite2(t *testing.T) {
	id, err := NewIdentityPoolWithTreeLevel(10)
	assert.Nil(t, err, "new identity instance error")

	for i := 0; i < 10; i++ {
		idc, _ := big.NewInt(0).SetString(fmt.Sprintf("%d", 100*i+1), 10)
		idx, err := id.InsertIdc(NewIdPathElement(NewTreeContent(idc)))
		assert.Nil(t, err, "register error")
		assert.Equal(t, i, idx)
	}

	commitmentSet := make([]*IdPathElement, 3)
	for i := 0; i < 3; i++ {
		idc, _ := big.NewInt(0).SetString(fmt.Sprintf("%d", 100*i+1), 10)
		commitmentSet[i] = NewIdPathElement(NewTreeContent(idc))
	}

	num, err := id.Overwrite(commitmentSet)
	assert.Nil(t, err, "overwrite error")
	assert.Equal(t, 3, num)
}

func TestOverwrite_ForceError(t *testing.T) {
	id, err := NewIdentityPoolWithTreeLevel(10)
	assert.Nil(t, err, "new identity instance error")

	for i := 0; i < 3; i++ {
		idc, _ := big.NewInt(0).SetString(fmt.Sprintf("%d", 100*i+1), 10)
		idx, err := id.InsertIdc(NewIdPathElement(NewTreeContent(idc)))
		assert.Nil(t, err, "register error")
		assert.Equal(t, i, idx)
	}

	commitmentSet := make([]*IdPathElement, 10)
	for i := 0; i < 10; i++ {
		commitmentSet[i] = NewIdPathElement(NewTreeContent(big.NewInt(0)))
	}

	num, err := id.Overwrite(commitmentSet)
	assert.NotNil(t, err, "overwrite should have errors")
	assert.Equal(t, 3, num)
}
