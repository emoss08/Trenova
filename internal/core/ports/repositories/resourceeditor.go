package repositories

import "context"

// SchemaInformation holds all schema details for the requested schema.
type SchemaInformation struct {
	SchemaName string         `json:"schemaName"`
	Tables     []TableDetails `json:"tables"`
}

// TableDetails holds information about a specific table.
type TableDetails struct {
	TableName   string              `json:"tableName"`
	Columns     []ColumnDetails     `json:"columns"`
	Indexes     []IndexDetails      `json:"indexes"`
	Constraints []ConstraintDetails `json:"constraints"`
}

// ColumnDetails holds information about a table column.
type ColumnDetails struct {
	ColumnName             string  `json:"columnName"`
	OrdinalPosition        int     `json:"ordinalPosition"`
	ColumnDefault          *string `json:"columnDefault,omitempty"`
	IsNullable             string  `json:"isNullable"` // "YES" or "NO"
	DataType               string  `json:"dataType"`
	CharacterMaximumLength *int64  `json:"characterMaximumLength,omitempty"`
	NumericPrecision       *int64  `json:"numericPrecision,omitempty"`
	NumericScale           *int64  `json:"numericScale,omitempty"`
	Comment                *string `json:"comment,omitempty"`
}

// IndexDetails holds information about a table index.
type IndexDetails struct {
	IndexName       string   `json:"indexName"`
	IndexDefinition string   `json:"indexDefinition"` // SQL definition
	IsUnique        bool     `json:"isUnique"`
	IsPrimary       bool     `json:"isPrimary"`
	IndexType       string   `json:"indexType"` // e.g., btree, hash
	Columns         []string `json:"columns"`
}

// ConstraintDetails holds information about a table constraint.
type ConstraintDetails struct {
	ConstraintName     string   `json:"constraintName"`
	ConstraintType     string   `json:"constraintType"` // PRIMARY KEY, FOREIGN KEY, UNIQUE, CHECK
	ColumnNames        []string `json:"columnNames,omitempty"`
	ForeignTableName   *string  `json:"foreignTableName,omitempty"`
	ForeignColumnNames []string `json:"foreignColumnNames,omitempty"`
	CheckClause        *string  `json:"checkClause,omitempty"`
	Deferrable         bool     `json:"deferrable"`
	InitiallyDeferred  bool     `json:"initiallyDeferred"`
}

// AutocompleteRequest defines the structure for autocomplete requests.
type AutocompleteRequest struct {
	SchemaName   string `json:"schemaName"`
	TableName    string `json:"tableName,omitempty"`    // Optional, if context is specific to a table
	CurrentQuery string `json:"currentQuery,omitempty"` // The full query text so far
	Prefix       string `json:"prefix,omitempty"`       // The word/prefix the user is currently typing
	// We might add cursor position or other context later if needed
}

// AutocompleteSuggestion defines a single suggestion item.
type AutocompleteSuggestion struct {
	Value   string `json:"value"`   // The actual text to be inserted
	Caption string `json:"caption"` // Text displayed in the suggestion list (can be same as value)
	Meta    string `json:"meta"`    // Type of suggestion (e.g., "table", "column", "keyword", "schema")
	Score   int    `json:"score"`   // Score to rank suggestions (higher is better)
}

// AutocompleteResponse is the list of suggestions.
type AutocompleteResponse struct {
	Suggestions []AutocompleteSuggestion `json:"suggestions"`
}

// --- SQL Query Execution Types ---

// ExecuteQueryRequest defines the request for executing a SQL query.
type ExecuteQueryRequest struct {
	SchemaName string `json:"schemaName"` // To ensure context, though the query itself might specify a schema
	Query      string `json:"query"`
	Page       int    `json:"page,omitempty"`
	PageSize   int    `json:"pageSize,omitempty"`
}

// QueryResult holds the structured result of a successful SELECT query.
type QueryResult struct {
	Columns     []string        `json:"columns"`           // List of column names in order
	Rows        [][]interface{} `json:"rows"`              // Slice of rows, each row is a slice of interfaces
	Message     string          `json:"message,omitempty"` // For non-SELECT queries or messages like "0 rows affected"
	Error       string          `json:"error,omitempty"`   // If query execution itself had an error
	TotalRows   *int64          `json:"totalRows,omitempty"`
	TotalPages  *int            `json:"totalPages,omitempty"`
	CurrentPage *int            `json:"currentPage,omitempty"`
	PageSize    *int            `json:"pageSize,omitempty"`
}

// ExecuteQueryResponse is the response from executing a SQL query.
// It might contain results or just a message/error.
type ExecuteQueryResponse struct {
	Result QueryResult `json:"result"`
}

type ResourceEditorRepository interface {
	GetTableSchema(ctx context.Context, schemaName string) (*SchemaInformation, error)
	GetAutocompleteSuggestions(ctx context.Context, req AutocompleteRequest) (*AutocompleteResponse, error)
	ExecuteSQLQuery(ctx context.Context, req ExecuteQueryRequest) (*ExecuteQueryResponse, error)
}
