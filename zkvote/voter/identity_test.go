package voter

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

const idCommitment string = "13340458618778421339172160768420045586192754964779906616295828288105624722649"

func TestRegister(t *testing.T) {
	id, err := NewIdentityWithTreeLevel(10)
	assert.Nil(t, err, "new identity instance error")

	idc, _ := big.NewInt(0).SetString(idCommitment, 10)
	idx, err := id.Register(idc)
	assert.Nil(t, err, "register error")
	assert.Equal(t, 0, idx)

	expectedRoot, _ := big.NewInt(0).SetString("3045810468960591087191210641638281907572011303565471692703782899879208219595", 10)
	assert.Equal(t, expectedRoot, id.tree.GetRoot())
}

func TestRegister_10IDs(t *testing.T) {
	id, err := NewIdentityWithTreeLevel(10)
	assert.Nil(t, err, "new identity instance error")

	for i := 0; i < 10; i++ {
		idc, _ := big.NewInt(0).SetString(fmt.Sprintf("%d", i+1), 10)
		idx, err := id.Register(idc)
		assert.Nil(t, err, "register error")
		assert.Equal(t, i, idx)
	}
}

func TestRegister_Double(t *testing.T) {
	id, err := NewIdentityWithTreeLevel(10)
	assert.Nil(t, err, "new identity instance error")

	idc, _ := big.NewInt(0).SetString(idCommitment, 10)
	idx, err := id.Register(idc)
	assert.Nil(t, err, "register error")
	assert.Equal(t, 0, idx)

	idx, err = id.Register(idc)
	assert.NotNil(t, err, "should not register successfully")
}

func TestUpdate(t *testing.T) {
	id, err := NewIdentityWithTreeLevel(10)
	assert.Nil(t, err, "new identity instance error")

	idc, _ := big.NewInt(0).SetString(idCommitment, 10)
	idx, err := id.Register(idc)
	assert.Nil(t, err, "register error")
	assert.Equal(t, 0, idx)

	err = id.Update(uint(idx), idc, big.NewInt(100))
	assert.Nil(t, err, "update error")
}

func TestUpdate_IncorrectIdx(t *testing.T) {
	id, err := NewIdentityWithTreeLevel(10)
	assert.Nil(t, err, "new identity instance error")

	idc, _ := big.NewInt(0).SetString(idCommitment, 10)
	idx, err := id.Register(idc)
	assert.Nil(t, err, "register error")
	assert.Equal(t, 0, idx)

	err = id.Update(1, idc, big.NewInt(100))
	assert.NotNil(t, err, "update error")
}

func TestUpdate_IncorrectContent(t *testing.T) {
	id, err := NewIdentityWithTreeLevel(10)
	assert.Nil(t, err, "new identity instance error")

	idc, _ := big.NewInt(0).SetString(idCommitment, 10)
	idx, err := id.Register(idc)
	assert.Nil(t, err, "register error")
	assert.Equal(t, 0, idx)

	err = id.Update(uint(idx), big.NewInt(100), big.NewInt(100))
	assert.NotNil(t, err, "update error")
}

func TestIsMember(t *testing.T) {
	id, err := NewIdentityWithTreeLevel(10)
	assert.Nil(t, err, "new identity instance error")

	idc, _ := big.NewInt(0).SetString(idCommitment, 10)
	idx, err := id.Register(idc)
	assert.Nil(t, err, "register error")
	assert.Equal(t, 0, idx)

	assert.True(t, id.IsMember(id.tree.GetRoot()))
}

func TestIsMember2(t *testing.T) {
	id, err := NewIdentityWithTreeLevel(10)
	assert.Nil(t, err, "new identity instance error")

	idc, _ := big.NewInt(0).SetString(idCommitment, 10)
	idx, err := id.Register(idc)
	assert.Nil(t, err, "register error")
	assert.Equal(t, 0, idx)
	root1 := id.tree.GetRoot()

	idx, err = id.Register(big.NewInt(10))
	assert.Nil(t, err, "register error")
	assert.Equal(t, 1, idx)

	assert.True(t, id.IsMember(root1))
	assert.True(t, id.IsMember(id.tree.GetRoot()))
}
