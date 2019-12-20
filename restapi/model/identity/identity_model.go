package identity

// A GenericError is the default error message that is generated.
// For certain status codes there are more appropriate error structures.
//
// swagger:response genericError
type GenericError struct {
	// in: body
	Body struct {
		Code    int32  `json:"code"`
		Message string `json:"message"`
	} `json:"body"`
}

// GetSnarkDataRequest ...
type GetSnarkDataRequest struct{}

// GetSnarkDataResponse ...
type GetSnarkDataResponse struct {
	// in: body
	Results string `json:"results"`
}
