package search

// Request represents the search API request parameters.
type Request struct {
	Query     string   `query:"q"`                                        // Search query string
	Types     []string `query:"types" validate:"omitempty"`               // Optional entity types filter
	Limit     int      `query:"limit" validate:"omitempty,min=1,max=100"` // Result count limit (1-100)
	Offset    int      `query:"offset" validate:"omitempty,min=0"`        // Pagination offset
	SortBy    string   `query:"sortBy" validate:"omitempty"`              // Sort field and direction (e.g., "createdAt:desc")
	Filter    string   `query:"filter" validate:"omitempty"`              // Filter expression
	Facets    []string `query:"facets" validate:"omitempty"`              // Fields to facet on
	Highlight bool     `query:"highlight" validate:"omitempty"`           // Whether to highlight matches
}

// Response represents the search API response.
type Response struct {
	Results     any    `json:"results"`            // Search results
	Total       int    `json:"total"`              // Total result count
	ProcessedIn string `json:"processedIn"`        // Processing time in human-readable format
	Query       string `json:"query"`              // Original query
	Offset      int    `json:"offset"`             // Current offset
	Limit       int    `json:"limit"`              // Applied limit
	Facets      any    `json:"facets,omitempty"`   // Faceted results if requested
	Metadata    any    `json:"metadata,omitempty"` // Additional metadata
}
