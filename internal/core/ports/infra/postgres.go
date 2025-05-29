package infra

type PostgresSearchType string

const (
	PostgresSearchTypeText   = PostgresSearchType("text") // Regular text search
	PostgresSearchTypeNumber = PostgresSearchType(
		"number",
	) // Number search (uses pattern matching)
	PostgresSearchTypeEnum      = PostgresSearchType("enum")  // Enum search (exact match)
	PostgresSearchTypeArray     = PostgresSearchType("array") // Array search (exact match)
	PostgresSearchTypeComposite = PostgresSearchType(
		"composite",
	) // Composite fields (like pro_number)
)

// SearchableField represents a field that can be searched
type PostgresSearchableField struct {
	Name       string             // Database column name
	Weight     string             // Weight for ranking (A, B, C, D)
	Type       PostgresSearchType // Type of search to perform
	Dictionary string             // PostgreSQL dictionary to use (default: 'english')
}

// Config holds the configuration for search functionality
type PostgresSearchConfig struct {
	TableAlias      string                    // Database table alias
	Fields          []PostgresSearchableField // Fields to search
	MinLength       int                       // Minimum search query length
	MaxTerms        int                       // Maximum number of search terms
	UsePartialMatch bool                      // Whether to use pattern matching for partial matches
	CustomRank      string                    // Optional custom ranking expression
}

type PostgresSearchable interface {
	GetPostgresSearchConfig() PostgresSearchConfig
}
