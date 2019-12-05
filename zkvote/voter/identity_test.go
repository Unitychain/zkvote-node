package voter

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegister(t *testing.T) {
	id, err := NewIdentityWithTreeLevel(10)
	assert.Nil(t, err, "new identity instance error")

	idc, _ := big.NewInt(0).SetString("8754576736334110930938370425891433351958026259715678541049362293217590250071", 10)
	idx, err := id.Register(idc)
	assert.Nil(t, err, "register error")
	assert.Equal(t, 0, idx)

	expectedRoot, _ := big.NewInt(0).SetString("8689527539353720499147190441125863458163151721613317120863488267899702105029", 10)
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

	idc, _ := big.NewInt(0).SetString("8754576736334110930938370425891433351958026259715678541049362293217590250071", 10)
	idx, err := id.Register(idc)
	assert.Nil(t, err, "register error")
	assert.Equal(t, 0, idx)

	idx, err = id.Register(idc)
	assert.NotNil(t, err, "should not register successfully")
}

func TestUpdate(t *testing.T) {
	id, err := NewIdentityWithTreeLevel(10)
	assert.Nil(t, err, "new identity instance error")

	idc, _ := big.NewInt(0).SetString("8754576736334110930938370425891433351958026259715678541049362293217590250071", 10)
	idx, err := id.Register(idc)
	assert.Nil(t, err, "register error")
	assert.Equal(t, 0, idx)

	err = id.Update(uint(idx), idc, big.NewInt(100))
	assert.Nil(t, err, "update error")
}

func TestUpdate_IncorrectIdx(t *testing.T) {
	id, err := NewIdentityWithTreeLevel(10)
	assert.Nil(t, err, "new identity instance error")

	idc, _ := big.NewInt(0).SetString("8754576736334110930938370425891433351958026259715678541049362293217590250071", 10)
	idx, err := id.Register(idc)
	assert.Nil(t, err, "register error")
	assert.Equal(t, 0, idx)

	err = id.Update(1, idc, big.NewInt(100))
	assert.NotNil(t, err, "update error")
}

func TestUpdate_IncorrectContent(t *testing.T) {
	id, err := NewIdentityWithTreeLevel(10)
	assert.Nil(t, err, "new identity instance error")

	idc, _ := big.NewInt(0).SetString("8754576736334110930938370425891433351958026259715678541049362293217590250071", 10)
	idx, err := id.Register(idc)
	assert.Nil(t, err, "register error")
	assert.Equal(t, 0, idx)

	err = id.Update(uint(idx), big.NewInt(100), big.NewInt(100))
	assert.NotNil(t, err, "update error")
}

func TestIsMember(t *testing.T) {
	id, err := NewIdentityWithTreeLevel(10)
	assert.Nil(t, err, "new identity instance error")

	idc, _ := big.NewInt(0).SetString("8754576736334110930938370425891433351958026259715678541049362293217590250071", 10)
	idx, err := id.Register(idc)
	assert.Nil(t, err, "register error")
	assert.Equal(t, 0, idx)

	assert.True(t, id.IsMember(idc))
}
