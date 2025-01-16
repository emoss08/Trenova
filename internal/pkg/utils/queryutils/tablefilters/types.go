package tablefilters

// QueryParams represents the main query parameter structure
type QueryParams struct {
	Filters      []*Filter    `json:"filters"`
	JoinOperator JoinOperator `json:"joinOperator"`
}

// Filter represents a single filter condition
type Filter struct {
	Field    string     `json:"field"`
	Operator Operator   `json:"operator"`
	Value    any        `json:"value"`   // Can be string, []string, number, bool, etc.
	ValueTo  any        `json:"valueTo"` // For between operations
	Type     ColumnType `json:"type"`
}

type QueryOptions struct {
	TableAlias       string
	EnumFields       []EnumField
	RelationMappings []RelationMapping
}

type EnumField struct {
	TableAlias string
	Fields     []string
}

// RelationType defines how tables are related
type RelationType string

const (
	RelationTypeOneToOne  RelationType = "one_to_one"
	RelationTypeOneToMany RelationType = "one_to_many"
	RelationTypeManyToOne RelationType = "many_to_one"
)

// RelationMapping defines how tables are related and mapped
type RelationMapping struct {
	From       string       // Source field/table name (e.g., "profile")
	To         string       // Target table alias (e.g., "worker_profiles")
	Type       RelationType // Type of relationship
	LocalKey   string       // Local key field (e.g., "id")
	ForeignKey string       // Foreign key field (e.g., "worker_id")
}

type Operator string

// Text Operators
const (
	OperatorILike      Operator = "iLike"
	OperatorNotILike   Operator = "notILike"
	OperatorEq         Operator = "eq"
	OperatorNe         Operator = "ne"
	OperatorIsEmpty    Operator = "isEmpty"
	OperatorIsNotEmpty Operator = "isNotEmpty"

	// Numeric/Date Operators
	OperatorLt        Operator = "lt"
	OperatorLte       Operator = "lte"
	OperatorGt        Operator = "gt"
	OperatorGte       Operator = "gte"
	OperatorIsBetween Operator = "isBetween"

	// Select/Multi-select Operators
	OperatorIn    Operator = "in"
	OperatorNotIn Operator = "notIn"

	// Special Operators
	OperatorIsRelativeToToday Operator = "isRelativeToToday"
)

type JoinOperator string

const (
	JoinOperatorAnd JoinOperator = "and"
	JoinOperatorOr  JoinOperator = "or"
)

type SortOrder string

const (
	SortOrderAsc  SortOrder = "asc"
	SortOrderDesc SortOrder = "desc"
)

type ColumnType string

const (
	ColumnTypeText        ColumnType = "text"
	ColumnTypeNumber      ColumnType = "number"
	ColumnTypeDate        ColumnType = "date"
	ColumnTypeBoolean     ColumnType = "boolean"
	ColumnTypeSelect      ColumnType = "select"
	ColumnTypeMultiSelect ColumnType = "multi-select"
)
