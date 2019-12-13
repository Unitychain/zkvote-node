package subject

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

// IndexRequest ...
type IndexRequest struct{}

// ProposeRequest ...
type ProposeRequest struct {
	*ProposeParams
}

// ProposeParams ...
type ProposeParams struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// IndexResponse ...
type IndexResponse struct {
	// in: body
	Results []map[string]string `json:"results"`
}

// ProposeResponse ...
type ProposeResponse struct {
	// in: body
	Results string `json:"results"`
}
