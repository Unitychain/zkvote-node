package identity

import (
	"math/big"

	"github.com/unitychain/zkvote-node/zkvote/operator/service/utils"
)

// Identity ...
type Identity string

// NewIdentity ...
func NewIdentity(commitment string) *Identity {
	// TODO: Change the encoding tp hex if needed
	if err := utils.CheckHex(commitment); err != nil {
		return nil
	}
	id := Identity(utils.Remove0x(commitment))
	return &id
}

// NewIdentityFromBytes ...
func NewIdentityFromBytes(bytes []byte) *Identity {
	hex := utils.GetHexStringFromBytes(bytes)
	id := Identity(hex)
	return &id
}

// Hash ...
type Hash []byte

// Byte ...
func (id *Identity) Byte() []byte { return utils.GetBytesFromHexString(id.String()) }

// String ...
func (id *Identity) String() string { return string(*id) }

// Hex ...
func (id *Identity) Hex() string {
	return utils.GetHexStringFromBigInt(big.NewInt(0).SetBytes(id.Byte()))
}

// Equal .
func (id *Identity) Equal(othID *Identity) bool {
	self := big.NewInt(0).SetBytes(id.Byte())
	oth := big.NewInt(0).SetBytes(othID.Byte())
	return 0 == self.Cmp(oth)
}

// PathElement ...
func (id Identity) PathElement() *IdPathElement {
	// TODO: do check
	bigValue := big.NewInt(0).SetBytes(id.Byte())
	return NewIdPathElement(NewTreeContent(bigValue))
}

// Set ...
type Set map[Identity]string

// NewSet ...
func NewSet() Set {
	result := Set(make(map[Identity]string))
	return result
}

//
// IdPathElement
//

type IdPathElement struct {
	e *TreeContent
}

func NewIdPathElement(t *TreeContent) *IdPathElement {
	return &IdPathElement{t}
}
func (i *IdPathElement) String() string {
	if nil == i.e || 0 == i.e.BigInt().Cmp(big.NewInt(0)) {
		return "0"
	}
	return i.e.String()
}
func (i *IdPathElement) Hex() string {
	if nil == i.e || 0 == i.e.BigInt().Cmp(big.NewInt(0)) {
		return "0x0"
	}
	return i.e.Hex()
}
func (i *IdPathElement) Equal(oth IdPathElement) bool {
	b, _ := i.e.Equals(oth.e)
	return b
}
func (i *IdPathElement) BigInt() *big.Int     { return i.e.BigInt() }
func (i *IdPathElement) Content() TreeContent { return *i.e }
