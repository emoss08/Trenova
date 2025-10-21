package schema

type DataSource struct {
	Table   string                `json:"table"`             // Database table name
	Entity  string                `json:"entity"`            // Go entity type (for reflection)
	Preload []string              `json:"preload,omitempty"` // Relations to preload
	Joins   map[string]JoinConfig `json:"joins,omitempty"`   // Join configurations
	Filters []FilterConfig        `json:"filters,omitempty"` // Default filters
	OrderBy string                `json:"orderBy,omitempty"` // Default ordering
}

type JoinConfig struct {
	Table     string `json:"table"`
	On        string `json:"on"`
	Type      string `json:"type,omitempty"`      // LEFT, INNER, etc. Default: LEFT
	Condition string `json:"condition,omitempty"` // Additional WHERE conditions
}

type FilterConfig struct {
	Field    string `json:"field"`
	Operator string `json:"operator"` // eq, ne, gt, lt, like, in, etc.
	Value    any    `json:"value"`
}

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

type FormulaContextExtension struct {
	Category    string   `json:"category"`
	Entities    []string `json:"entities"`       // Which entities this schema applies to
	Permissions []string `json:"permissions"`    // Required permissions
	Tags        []string `json:"tags,omitempty"` // Searchable tags
}

const (
	TransformDecimalToFloat64 = "decimalToFloat64"
	TransformInt64ToFloat64   = "int64ToFloat64"
	TransformInt16ToFloat64   = "int16ToFloat64"
	TransformStringToUpper    = "stringToUpper"
	TransformStringToLower    = "stringToLower"
	TransformUnixToISO8601    = "unixToISO8601"
)

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
