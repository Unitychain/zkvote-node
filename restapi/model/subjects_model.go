package model

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

// QuerySubjectsParams ...
type QuerySubjectsParams struct {
	// Title
	Title string `json:"title"`
}

// QuerySubjectsResponse ...
type QuerySubjectsResponse struct {
	// in: body
	Results map[string]string `json:"results"`
}
