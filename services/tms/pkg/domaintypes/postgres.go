package domaintypes

// FieldType defines how a field should be handled in queries
type FieldType string

const (
	FieldTypeText      FieldType = "text"      // Use ILIKE for text search
	FieldTypeNumber    FieldType = "number"    // Use exact match or range
	FieldTypeBoolean   FieldType = "boolean"   // Use exact match
	FieldTypeEnum      FieldType = "enum"      // Use exact match with ::text cast
	FieldTypeDate      FieldType = "date"      // Use date comparisons
	FieldTypeComposite FieldType = "composite" // Use composite type for search
)

type SearchWeight string

const (
	SearchWeightA     SearchWeight = "A"
	SearchWeightB     SearchWeight = "B"
	SearchWeightC     SearchWeight = "C"
	SearchWeightD     SearchWeight = "D"
	SearchWeightBlank SearchWeight = ""
)

var SearchWeights = []SearchWeight{SearchWeightA, SearchWeightB, SearchWeightC, SearchWeightD}

func (w SearchWeight) GetScore() int {
	switch w {
	case SearchWeightA:
		return 4
	case SearchWeightB:
		return 3
	case SearchWeightC:
		return 2
	case SearchWeightD:
		return 1
	case SearchWeightBlank:
		return 0
	}
	return 1
}

func (w SearchWeight) String() string {
	return string(w)
}

// SearchableField defines a field that can be searched
type SearchableField struct {
	Name   string       // Database field name (snake_case)
	Type   FieldType    // How to handle this field in searches
	Weight SearchWeight // PostgreSQL text search weight (A, B, C, D) for ranking (optional, for text fields)
}

// PostgresSearchConfig contains metadata about entity fields for query building
type PostgresSearchConfig struct {
	TableAlias         string                    // Table alias for queries
	SearchableFields   []SearchableField         // Fields and how to search them
	Relationships      []*RelationshipDefinition // Queryable relationships for this entity
	UseSearchVector    bool                      // Whether to use search_vector column for text search
	SearchVectorColumn string                    // Name of the search vector column (default: "search_vector")
}

type RelationshipType string

const (
	RelationshipTypeBelongsTo  RelationshipType = "belongs_to"
	RelationshipTypeHasOne     RelationshipType = "has_one"
	RelationshipTypeHasMany    RelationshipType = "has_many"
	RelationshipTypeManyToMany RelationshipType = "many_to_many"
	RelationshipTypeCustom     RelationshipType = "custom" // For complex multi-hop joins
)

// JoinType defines the type of SQL join to use
type JoinType string

const (
	JoinTypeLeft  JoinType = "LEFT"
	JoinTypeRight JoinType = "RIGHT"
	JoinTypeInner JoinType = "INNER"
)

// JoinStep represents a single join in a multi-hop join path
type JoinStep struct {
	Table     string   // Table name to join
	Alias     string   // Alias for the joined table
	Condition string   // Join condition (e.g., "sp.id = sm.shipment_id AND sm.type = 'Pickup'")
	JoinType  JoinType // Type of join (LEFT, RIGHT, INNER)
}

type RelationshipDefinition struct {
	Field        string           // Field name in the struct (e.g., "LocationCategory")
	Type         RelationshipType // Type of relationship
	TargetEntity any              // Target entity type (for field extraction)
	TargetTable  string           // Target table name (e.g., "location_categories")
	ForeignKey   string           // Foreign key field (e.g., "location_category_id")
	ReferenceKey string           // Reference key in target (e.g., "id")
	Alias        string           // Table alias to use in joins (e.g., "lc")
	Queryable    bool             // Whether this relationship can be used in queries

	// Many-to-many specific fields
	JoinTable          string // Join table name (e.g., "user_roles")
	JoinTableAlias     string // Join table alias (e.g., "ur")
	JoinTableSourceKey string // Source key in join table (e.g., "user_id")
	JoinTableTargetKey string // Target key in join table (e.g., "role_id")

	// Custom join path fields (for complex multi-hop relationships)
	CustomJoinPath []JoinStep // Explicit join sequence for complex relationships
	TargetField    string     // The field to reference on the final table (e.g., "name")
	IsEnum         bool       // Whether the target field is an enum type
}

type PostgresSearchable interface {
	GetPostgresSearchConfig() PostgresSearchConfig
	GetTableName() string
}
