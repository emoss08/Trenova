package cursorpagination

import "github.com/uptrace/bun"

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// Cursor represents a position in the dataset using composite keys
type Cursor struct {
	Values map[string]any `json:"values"` // Map of field names to their values
}

// Query represents pagination parameters
type Query struct {
	Cursor   *string `json:"cursor"`   // Encoded cursor for the current position
	PageSize int     `json:"pageSize"` // Number of items per page
}

// BaseModel defines required methods for models using cursor pagination
type BaseModel interface {
	GetTableName() string
	GetTableAlias() string
	GetPrimaryKey() PrimaryKey
	GetID() string
}

// PrimaryKey defines the structure of a composite primary key
type PrimaryKey struct {
	Fields    []string // Field names in the composite key
	Values    []any    // Values for each field
	Composite bool     // Whether this is a composite key
}

// BuildPaginatedQueryOptions contains options for building paginated queries
type BuildPaginatedQueryOptions struct {
	Query          *bun.SelectQuery
	PaginationOpts Query
	Model          BaseModel
}

// Meta contains metadata about the pagination state
type Meta struct {
	TotalCount int  `json:"totalCount"` // Total number of records
	HasMore    bool `json:"hasMore"`    // Whether there are more records
}

// Cursors contains pagination cursors for navigation
type Cursors struct {
	Next     *string `json:"next,omitempty"`     // Cursor for next page
	Previous *string `json:"previous,omitempty"` // Cursor for previous page
}

// Response represents the paginated response
type Response[T BaseModel] struct {
	Data    []*T    `json:"data"`    // The paginated data
	Meta    Meta    `json:"meta"`    // Metadata about the response
	Cursors Cursors `json:"cursors"` // Navigation cursors
}
