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

// JoinRequest ...
type JoinRequest struct {
	*JoinParams
}

// VoteRequest ...
type VoteRequest struct {
	*VoteParams
}

// GetIdentityPathRequest ...
type GetIdentityPathRequest struct {
	*GetIdentityPathParams
}

// ProposeParams ...
type ProposeParams struct {
	Title              string `json:"title"`
	Description        string `json:"description"`
	IdentityCommitment string `json:"identityCommitment"`
}

// JoinParams ...
type JoinParams struct {
	SubjectHash        string `json:"subjectHash"`
	IdentityCommitment string `json:"identityCommitment"`
}

// VoteParams ...
type VoteParams struct {
	SubjectHash string `json:"subjectHash"`
	Proof       string `json:"proof"`
}

// GetIdentityPathParams ...
type GetIdentityPathParams struct {
	SubjectHash        string `json:"subjectHash"`
	IdentityCommitment string `json:"identityCommitment"`
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

// JoinResponse ...
type JoinResponse struct {
	// in: body
	Results string `json:"results"`
}

// VoteResponse ...
type VoteResponse struct {
	// in: body
	Results string `json:"results"`
}

// GetIdentityPathResponse ...
type GetIdentityPathResponse struct {
	// in: body
	Results struct {
		Path  []string `json:"path"`
		Index []int    `json:"index"`
		Root  string   `json:"root"`
	} `json:"results"`
}
