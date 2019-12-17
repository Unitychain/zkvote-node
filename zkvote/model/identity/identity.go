package identity

// Identity ...
type Identity string

// NewIdentity ...
func NewIdentity(commitment string) *Identity {
	// TODO: Check if commitment is a hex string
	// TODO: Change the encoding tp hex if needed
	id := Identity(commitment)
	return &id
}

// Hash ...
type Hash []byte

// Byte ...
func (id Identity) Byte() []byte { return []byte(string(id)) }

// String ...
func (id Identity) String() string { return string(id) }

// Set ...
type Set map[Identity]string

// NewSet ...
func NewSet() Set {
	result := Set(make(map[Identity]string))
	return result
}
