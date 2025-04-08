package infra

import (
	"context"
	"time"

	"github.com/meilisearch/meilisearch-go"
)

type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusProcessing TaskStatus = "processing"
	TaskStatusSucceeded  TaskStatus = "succeeded"
	TaskStatusFailed     TaskStatus = "failed"
)

// SearchOptions defines parameters for a search operation.
type SearchOptions struct {
	Query       string   `json:"query"`                 // The search query text
	Limit       int      `json:"limit,omitempty"`       // Maximum number of results to return
	Offset      int      `json:"offset,omitempty"`      // Starting offset for pagination
	Types       []string `json:"types,omitempty"`       // Filter by entity types (e.g., "shipment", "customer", "driver")
	OrgID       string   `json:"orgId"`                 // Filter by organization ID for multi-tenancy
	BuID        string   `json:"buId"`                  // Filter by business unit ID for multi-tenancy
	SortBy      []string `json:"sortBy,omitempty"`      // Sorting options (e.g., "createdAt:desc")
	Filters     []string `json:"filters,omitempty"`     // Additional filters in Meilisearch filter syntax
	Facets      []string `json:"facets,omitempty"`      // Fields to generate facets for
	Highlight   []string `json:"highlight,omitempty"`   // Fields to highlight matches in
	MatchFields []string `json:"matchFields,omitempty"` // Limit search to specific fields
}

// SearchDocument represents a document in the search index.
type SearchDocument struct {
	// Core fields
	ID             string `json:"id" mapstructure:"id"`
	Type           string `json:"type" mapstructure:"type"`
	BusinessUnitID string `json:"businessUnitId" mapstructure:"businessUnitId"`
	OrganizationID string `json:"organizationId" mapstructure:"organizationId"`
	CreatedAt      int64  `json:"createdAt" mapstructure:"createdAt"`
	UpdatedAt      int64  `json:"updatedAt" mapstructure:"updatedAt"`

	// Searchable content
	Title          string `json:"title" mapstructure:"title"`
	Description    string `json:"description" mapstructure:"description"`
	SearchableText string `json:"searchableText" mapstructure:"searchableText"`

	// Additional data
	Metadata map[string]any `json:"metadata" mapstructure:"metadata"`

	// Highlight information (populated during search when highlighting is enabled)
	Highlights map[string][]string `json:"highlights,omitempty" mapstructure:"_highlights"`
}

// SearchTaskError contains details about a failed search operation.
type SearchTaskError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
	Link    string `json:"link"`
}

// SearchTask represents the status and details of an asynchronous search operation.
type SearchTask struct {
	Status     TaskStatus          `json:"status"`
	TaskUID    int64               `json:"taskUid"`
	IndexUID   string              `json:"indexUid"`
	Type       string              `json:"type"`
	Error      SearchTaskError     `json:"error"`
	Duration   string              `json:"duration,omitempty"`
	EnqueuedAt time.Time           `json:"enqueuedAt"`
	StartedAt  time.Time           `json:"startedAt,omitempty"`
	FinishedAt time.Time           `json:"finishedAt,omitempty"`
	Details    meilisearch.Details `json:"details,omitempty"`
	CanceledBy int64               `json:"canceledBy,omitempty"`
}

// SearchTaskInfo provides information about a queued search task.
type SearchTaskInfo struct {
	UID        int64  `json:"uid"`
	IndexUID   string `json:"indexUid"`
	Type       string `json:"type"`
	EnqueuedAt string `json:"enqueuedAt"`
	TaskUID    int64  `json:"taskUid"`
	Status     string `json:"status"`
	Duration   string `json:"duration,omitempty"`
}

// SearchFacetResult represents a facet result in the search response.
type SearchFacetResult struct {
	Value     string `json:"value"`
	Count     int    `json:"count"`
	Highlight string `json:"highlight,omitempty"`
}

// SearchClient defines the interface for interacting with the search backend.
type SearchClient interface {
	// Core operations

	// IndexDocuments adds documents to the search index
	IndexDocuments(indexName string, docs []*SearchDocument) (*SearchTaskInfo, error)

	// Search performs a search operation and returns matching documents
	Search(ctx context.Context, opts *SearchOptions) ([]*SearchDocument, error)

	// GetIndexName returns the name of the search index
	GetIndexName() string

	// WaitForTask waits for an asynchronous task to complete
	WaitForTask(taskUID int64, timeout time.Duration) (*SearchTask, error)

	// Index management

	// InitializeIndexes sets up the required search indexes
	InitializeIndexes() error

	// Document operations

	// DeleteDocument removes a document from the search index
	DeleteDocument(id string) (*SearchTaskInfo, error)

	// DeleteDocuments removes multiple documents from the search index
	DeleteDocuments(ids []string) (*SearchTaskInfo, error)

	// Monitoring

	// GetStats returns statistics about the search index
	GetStats() (map[string]any, error)

	// Advanced search operations

	// SuggestCompletions returns query suggestions for autocomplete
	SuggestCompletions(ctx context.Context, prefix string, limit int, types []string) ([]string, error)
}

// SearchableEntity defines an interface for entities that can be indexed for search.
type SearchableEntity interface {
	// GetID returns the unique identifier of the entity
	GetID() string

	// GetSearchType returns the type of the entity (e.g., "shipment", "customer")
	GetSearchType() string

	// ToDocument converts the entity to a search document
	ToDocument() SearchDocument
}
