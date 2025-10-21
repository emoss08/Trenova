package pagination

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
)

type TenantOptions struct {
	BuID   pulid.ID `json:"buId"`
	OrgID  pulid.ID `json:"orgId"`
	UserID pulid.ID `json:"userId"`
}

type FilterOperator string

const (
	OpEqual              = FilterOperator("eq")
	OpNotEqual           = FilterOperator("ne")
	OpGreaterThan        = FilterOperator("gt")
	OpGreaterThanOrEqual = FilterOperator("gte")
	OpLessThan           = FilterOperator("lt")
	OpLessThanOrEqual    = FilterOperator("lte")
	OpContains           = FilterOperator("contains")
	OpStartsWith         = FilterOperator("startswith")
	OpEndsWith           = FilterOperator("endswith")
	OpLike               = FilterOperator("like")
	OpILike              = FilterOperator("ilike")
	OpIn                 = FilterOperator("in")
	OpNotIn              = FilterOperator("notin")
	OpIsNull             = FilterOperator("isnull")
	OpIsNotNull          = FilterOperator("isnotnull")
	OpDateRange          = FilterOperator("daterange")
)

type SortDirection string

const (
	SortAsc  = SortDirection("asc")
	SortDesc = SortDirection("desc")
)

type FieldFilter struct {
	Field    string         `json:"field"    form:"field"`
	Operator FilterOperator `json:"operator" form:"operator" binding:"oneof=eq ne gt gte lt lte contains startswith endswith like ilike in notin isnull isnotnull daterange"`
	Value    any            `json:"value"    form:"value"`
}

type SortField struct {
	Field     string        `json:"field"     form:"field"`
	Direction SortDirection `json:"direction" form:"direction" binding:"oneof=asc desc"`
}

type QueryOptions struct {
	TenantOpts   TenantOptions `json:"tenantOpts"`
	Query        string        `json:"query"      form:"query"`
	Limit        int           `json:"limit"      form:"limit"  default:"20"`
	Offset       int           `json:"offset"     form:"offset" default:"0"`
	FieldFilters []FieldFilter `json:"filters"`
	Sort         []SortField   `json:"sort"`
}

type SelectQueryOptions struct {
	OrgID  pulid.ID `json:"orgId"`
	BuID   pulid.ID `json:"buId"`
	Limit  int      `json:"limit"  default:"20"`
	Offset int      `json:"offset" default:"0"`
	Query  string   `json:"query"               form:"query"`
}

func (q *QueryOptions) ValidateFilters(allowedFields map[string]bool) error {
	for _, filter := range q.FieldFilters {
		if !allowedFields[filter.Field] {
			return fmt.Errorf("filtering on field '%s' is not allowed", filter.Field)
		}

		if !isValidOperator(filter.Operator) {
			return fmt.Errorf("invalid filter operator '%s'", filter.Operator)
		}
	}
	return nil
}

func (q *QueryOptions) ValidateSort(allowedFields map[string]bool) error {
	for _, sort := range q.Sort {
		if !allowedFields[sort.Field] {
			return fmt.Errorf("sorting on field '%s' is not allowed", sort.Field)
		}

		if sort.Direction != SortAsc && sort.Direction != SortDesc {
			return fmt.Errorf("invalid sort direction '%s'", sort.Direction)
		}
	}
	return nil
}

func (q *QueryOptions) HasFilters() bool {
	return len(q.FieldFilters) > 0
}

func (q *QueryOptions) HasSort() bool {
	return len(q.Sort) > 0
}

func (q *QueryOptions) GetSortClause(fieldMap map[string]string, tableAlias string) string {
	if !q.HasSort() {
		return ""
	}

	var clauses []string
	for _, sort := range q.Sort {
		dbField := fieldMap[sort.Field]
		if dbField == "" {
			dbField = sort.Field
		}

		direction := strings.ToUpper(string(sort.Direction))
		if tableAlias != "" {
			clauses = append(clauses, fmt.Sprintf("%s.%s %s", tableAlias, dbField, direction))
		} else {
			clauses = append(clauses, fmt.Sprintf("%s %s", dbField, direction))
		}
	}

	return strings.Join(clauses, ", ")
}

func isValidOperator(op FilterOperator) bool {
	switch op {
	case OpEqual, OpNotEqual, OpGreaterThan, OpGreaterThanOrEqual,
		OpLessThan, OpLessThanOrEqual, OpContains, OpStartsWith,
		OpEndsWith, OpLike, OpILike, OpIn, OpNotIn, OpIsNull,
		OpIsNotNull, OpDateRange:
		return true
	default:
		return false
	}
}

type Response[T any] struct {
	Results T      `json:"results"`
	Count   int    `json:"count"`
	Next    string `json:"next"`
	Prev    string `json:"previous"`
}

type ListResult[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
}

type PageableHandler[T any] func(ctx *gin.Context, opts *QueryOptions) (*ListResult[T], error)

type JoinType string

const (
	JoinTypeInner = JoinType("INNER")
	JoinTypeLeft  = JoinType("LEFT")
	JoinTypeRight = JoinType("RIGHT")
	JoinTypeNone  = JoinType("")
)

type JoinDefinition struct {
	Table     string   `json:"table"`
	Alias     string   `json:"alias"`
	Condition string   `json:"condition"`
	JoinType  JoinType `json:"joinType"`
}

type NestedFieldDefinition struct {
	DatabaseField string           `json:"databaseField"`
	RequiredJoins []JoinDefinition `json:"requiredJoins"`
	IsEnum        bool             `json:"isEnum"`
}

type FieldConfiguration struct {
	FilterableFields map[string]bool                  `json:"filterableFields"`
	SortableFields   map[string]bool                  `json:"sortableFields"`
	FieldMap         map[string]string                `json:"fieldMap"`
	EnumMap          map[string]bool                  `json:"enumMap"`
	NestedFields     map[string]NestedFieldDefinition `json:"nestedFields"`
}

type Info struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}
