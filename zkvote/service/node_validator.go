package service

// NodeValidator ...
type NodeValidator struct{}

// Validate ...
func (nv NodeValidator) Validate(ket string, value []byte) error {
	return nil
}

// Select ...
func (nv NodeValidator) Select(key string, values [][]byte) (int, error) {
	return 0, nil
}
