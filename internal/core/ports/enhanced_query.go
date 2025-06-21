package ports

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// FilterOperator defines the available filter operators
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

// SortDirection defines the available sort directions
type SortDirection string

const (
	SortAsc  = SortDirection("asc")
	SortDesc = SortDirection("desc")
)

// FieldFilter represents a filter condition for a specific field
type FieldFilter struct {
	Field    string         `json:"field"    query:"field"`
	Operator FilterOperator `json:"operator" query:"operator"`
	Value    any            `json:"value"    query:"value"`
}

// SortField represents a sort condition for a specific field
type SortField struct {
	Field     string        `json:"field"     query:"field"`
	Direction SortDirection `json:"direction" query:"direction"`
}

// QueryOptions extends LimitOffsetQueryOptions with filtering and sorting
type QueryOptions struct {
	LimitOffsetQueryOptions `              json:"limitOffsetQueryOptions"`
	Filters                 []FieldFilter `json:"filters"                 query:"filters"`
	Sort                    []SortField   `json:"sort"                    query:"sort"`
}

// ToLimitOffsetQueryOptions converts QueryOptions to LimitOffsetQueryOptions
// for backward compatibility
func (q *QueryOptions) ToLimitOffsetQueryOptions() *LimitOffsetQueryOptions {
	return &LimitOffsetQueryOptions{
		TenantOpts: q.TenantOpts,
		Limit:      q.Limit,
		Offset:     q.Offset,
		Query:      q.Query,
		ID:         q.ID,
	}
}

// ValidateFilters validates the filter conditions
func (q *QueryOptions) ValidateFilters(allowedFields map[string]bool) error {
	for _, filter := range q.Filters {
		if !allowedFields[filter.Field] {
			return fmt.Errorf("filtering on field '%s' is not allowed", filter.Field)
		}

		if !isValidOperator(filter.Operator) {
			return fmt.Errorf("invalid filter operator '%s'", filter.Operator)
		}
	}
	return nil
}

// ValidateSort validates the sort conditions
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

// HasFilters returns true if there are any filters
func (q *QueryOptions) HasFilters() bool {
	return len(q.Filters) > 0
}

// HasSort returns true if there are any sort conditions
func (q *QueryOptions) HasSort() bool {
	return len(q.Sort) > 0
}

// GetSortClause returns a SQL ORDER BY clause from the sort conditions
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

// isValidOperator checks if the operator is valid
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

// EnhancedPageableHandler is a function that handles an enhanced pageable request
type EnhancedPageableHandler[T any] func(ctx *fiber.Ctx, opts *QueryOptions) (*ListResult[T], error)

// FieldConfiguration defines allowed fields for filtering and sorting
type FieldConfiguration struct {
	// * Allowed fields for filtering (API field name -> allowed)
	FilterableFields map[string]bool
	// * Allowed fields for sorting (API field name -> allowed)
	SortableFields map[string]bool
	// * Field mapping from API names to database column names
	FieldMap map[string]string
	// * Field mapping from API names to enum values
	EnumMap map[string]bool
}

// NewFieldConfiguration creates a new field configuration
func NewFieldConfiguration() *FieldConfiguration {
	return &FieldConfiguration{
		FilterableFields: make(map[string]bool),
		SortableFields:   make(map[string]bool),
		FieldMap:         make(map[string]string),
		EnumMap:          make(map[string]bool),
	}
}

// AddFilterableField adds a field as filterable
func (fc *FieldConfiguration) AddFilterableField(apiField, dbField string) *FieldConfiguration {
	fc.FilterableFields[apiField] = true
	if dbField != "" {
		fc.FieldMap[apiField] = dbField
	}
	return fc
}

// AddSortableField adds a field as sortable
func (fc *FieldConfiguration) AddSortableField(apiField, dbField string) *FieldConfiguration {
	fc.SortableFields[apiField] = true
	if dbField != "" {
		fc.FieldMap[apiField] = dbField
	}
	return fc
}

// AddField adds a field as both filterable and sortable
func (fc *FieldConfiguration) AddField(apiField, dbField string) *FieldConfiguration {
	return fc.AddFilterableField(apiField, dbField).AddSortableField(apiField, dbField)
}
