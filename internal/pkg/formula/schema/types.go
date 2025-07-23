// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package schema

// * DataSource defines where and how to fetch data for a schema
type DataSource struct {
	Table   string                `json:"table"`             // Database table name
	Entity  string                `json:"entity"`            // Go entity type (for reflection)
	Preload []string              `json:"preload,omitempty"` // Relations to preload
	Joins   map[string]JoinConfig `json:"joins,omitempty"`   // Join configurations
	Filters []FilterConfig        `json:"filters,omitempty"` // Default filters
	OrderBy string                `json:"orderBy,omitempty"` // Default ordering
}

// * JoinConfig defines how to join related tables
type JoinConfig struct {
	Table     string `json:"table"`
	On        string `json:"on"`
	Type      string `json:"type,omitempty"`      // LEFT, INNER, etc. Default: LEFT
	Condition string `json:"condition,omitempty"` // Additional WHERE conditions
}

// * FilterConfig defines default filters for queries
type FilterConfig struct {
	Field    string `json:"field"`
	Operator string `json:"operator"` // eq, ne, gt, lt, like, in, etc.
	Value    any    `json:"value"`
}

// * FieldSource defines how to extract a field value from the source
type FieldSource struct {
	Field     string   `json:"field,omitempty"`     // Database column name
	Path      string   `json:"path,omitempty"`      // Go struct field path
	Computed  bool     `json:"computed,omitempty"`  // Is this a computed field?
	Function  string   `json:"function,omitempty"`  // Compute function name
	Requires  []string `json:"requires,omitempty"`  // Required fields for computation
	Preload   []string `json:"preload,omitempty"`   // Additional preloads for this field
	Relation  string   `json:"relation,omitempty"`  // Relation name if this is a relation
	Table     string   `json:"table,omitempty"`     // Related table name
	Nullable  bool     `json:"nullable,omitempty"`  // Can this field be null?
	Transform string   `json:"transform,omitempty"` // Transformation function (e.g., decimalToFloat64)
	Type      string   `json:"type,omitempty"`      // Field type hint (decimal, enum, etc.)
}

// * FormulaContextExtension contains formula-specific metadata
type FormulaContextExtension struct {
	Category    string   `json:"category"`
	Entities    []string `json:"entities"`       // Which entities this schema applies to
	Permissions []string `json:"permissions"`    // Required permissions
	Tags        []string `json:"tags,omitempty"` // Searchable tags
}

// * Transform functions that can be applied to field values
const (
	TransformDecimalToFloat64 = "decimalToFloat64"
	TransformInt64ToFloat64   = "int64ToFloat64"
	TransformInt16ToFloat64   = "int16ToFloat64"
	TransformStringToUpper    = "stringToUpper"
	TransformStringToLower    = "stringToLower"
	TransformUnixToISO8601    = "unixToISO8601"
)

// * Standard operators for filters
const (
	OperatorEqual        = "eq"
	OperatorNotEqual     = "ne"
	OperatorGreaterThan  = "gt"
	OperatorLessThan     = "lt"
	OperatorGreaterEqual = "gte"
	OperatorLessEqual    = "lte"
	OperatorLike         = "like"
	OperatorIn           = "in"
	OperatorNotIn        = "not_in"
	OperatorIsNull       = "is_null"
	OperatorIsNotNull    = "is_not_null"
)
