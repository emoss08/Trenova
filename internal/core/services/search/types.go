package search

import (
	"time"

	"github.com/emoss08/trenova/internal/core/ports/infra"
	"github.com/rotisserie/eris"
)

// Common errors that can be returned by the search service.
var (
	ErrServiceStopped      = eris.New("search service is stopped")
	ErrIndexingInProgress  = eris.New("indexing is already in progress")
	ErrDocumentNotFound    = eris.New("document not found")
	ErrBatchProcessingFail = eris.New("batch processing failed")
	ErrTaskTimeout         = eris.New("task timed out")
	ErrInvalidRequest      = eris.New("invalid search request")
)

// SearchRequest defines parameters for a search request.
type SearchRequest struct {
	Query     string   `json:"query" validate:"required"` // The search query text
	Types     []string `json:"types,omitempty"`           // Filter by entity types
	Limit     int      `json:"limit,omitempty"`           // Maximum number of results to return
	Offset    int      `json:"offset,omitempty"`          // Starting offset for pagination
	OrgID     string   `json:"orgId" validate:"required"` // Organization ID for multi-tenancy
	BuID      string   `json:"buId" validate:"required"`  // Business Unit ID for multi-tenancy
	Facets    []string `json:"facets,omitempty"`          // Fields to generate facets for
	Filter    string   `json:"filter,omitempty"`          // Filter expression
	SortBy    []string `json:"sortBy,omitempty"`          // Sorting options
	Highlight bool     `json:"highlight,omitempty"`       // Whether to highlight matches
}

// SearchResponse contains search results and metadata.
type SearchResponse struct {
	Results     []*infra.SearchDocument `json:"results"`          // The search results
	Total       int                     `json:"total"`            // Total number of matching documents
	ProcessedIn time.Duration           `json:"processedIn"`      // Time taken to process the search
	Query       string                  `json:"query"`            // The original query
	Facets      map[string]interface{}  `json:"facets,omitempty"` // Facet results if requested
}

// batchOpt represents a batch operation for document indexing.
type batchOpt struct {
	documents []*infra.SearchDocument
	callback  func(error)
}
