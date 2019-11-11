package main

// KNValidator ...
type KNValidator struct{}

// Validate ...
func (knv KNValidator) Validate(ket string, value []byte) error {
	return nil
}

// Select ...
func (knv KNValidator) Select(key string, values [][]byte) (int, error) {
	return 0, nil
}
