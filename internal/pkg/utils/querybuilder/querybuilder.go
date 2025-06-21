package querybuilder

import (
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/uptrace/bun"
)

// QueryBuilder provides utilities for building queries with filtering and sorting
type QueryBuilder struct {
	query       *bun.SelectQuery
	tableAlias  string
	fieldConfig *ports.FieldConfiguration
}

// New creates a new QueryBuilder
func New(
	query *bun.SelectQuery,
	tableAlias string,
	fieldConfig *ports.FieldConfiguration,
) *QueryBuilder {
	return &QueryBuilder{
		query:       query,
		tableAlias:  tableAlias,
		fieldConfig: fieldConfig,
	}
}

// ApplyFilters applies the filter conditions to the query
func (qb *QueryBuilder) ApplyFilters(filters []ports.FieldFilter) *QueryBuilder {
	for _, filter := range filters {
		if !qb.fieldConfig.FilterableFields[filter.Field] {
			continue // Skip unauthorized fields
		}

		dbField := qb.getDBField(filter.Field)
		isEnum := qb.isEnumField(filter.Field)
		qb.applyFilter(dbField, filter.Operator, filter.Value, isEnum)
	}
	return qb
}

// ApplySort applies the sort conditions to the query
func (qb *QueryBuilder) ApplySort(sorts []ports.SortField) *QueryBuilder {
	for _, sort := range sorts {
		if !qb.fieldConfig.SortableFields[sort.Field] {
			continue // Skip unauthorized fields
		}

		dbField := qb.getDBField(sort.Field)
		direction := strings.ToUpper(string(sort.Direction))

		if qb.tableAlias != "" {
			qb.query = qb.query.Order(fmt.Sprintf("%s.%s %s", qb.tableAlias, dbField, direction))
		} else {
			qb.query = qb.query.Order(fmt.Sprintf("%s %s", dbField, direction))
		}
	}
	return qb
}

// GetQuery returns the modified query
func (qb *QueryBuilder) GetQuery() *bun.SelectQuery {
	return qb.query
}

// getDBField gets the database field name from the API field name
func (qb *QueryBuilder) getDBField(apiField string) string {
	if dbField, exists := qb.fieldConfig.FieldMap[apiField]; exists {
		return dbField
	}

	return apiField
}

// isEnumField checks if a field is an enum field
func (qb *QueryBuilder) isEnumField(apiField string) bool {
	_, exists := qb.fieldConfig.EnumMap[apiField]
	return exists
}

// applyFilter applies a single filter condition
func (qb *QueryBuilder) applyFilter(
	dbField string,
	operator ports.FilterOperator,
	value any,
	isEnum bool,
) {
	fieldRef := qb.getFieldReference(dbField)

	// Cast enum fields to text for string operations
	if isEnum && qb.isStringOperation(operator) {
		fieldRef = fmt.Sprintf("%s::text", fieldRef)
	}

	// * Apply the where statement to the query
	qb.applyStatement(operator, fieldRef, value)
}

func (qb *QueryBuilder) applyStatement(operator ports.FilterOperator, fieldRef string, value any) {
	switch operator {
	case ports.OpEqual:
		if qb.isEmptyEnumValue(value) {
			return
		}
		qb.query = qb.query.Where(fmt.Sprintf("%s = ?", fieldRef), value)
	case ports.OpNotEqual:
		if qb.isEmptyEnumValue(value) {
			return
		}
		qb.query = qb.query.Where(fmt.Sprintf("%s != ?", fieldRef), value)
	case ports.OpGreaterThan:
		qb.query = qb.query.Where(fmt.Sprintf("%s > ?", fieldRef), value)
	case ports.OpGreaterThanOrEqual:
		qb.query = qb.query.Where(fmt.Sprintf("%s >= ?", fieldRef), value)
	case ports.OpLessThan:
		qb.query = qb.query.Where(fmt.Sprintf("%s < ?", fieldRef), value)
	case ports.OpLessThanOrEqual:
		qb.query = qb.query.Where(fmt.Sprintf("%s <= ?", fieldRef), value)
	case ports.OpContains:
		qb.query = qb.query.Where(
			fmt.Sprintf("%s ILIKE ?", fieldRef),
			fmt.Sprintf("%%%v%%", value),
		)
	case ports.OpStartsWith:
		qb.query = qb.query.Where(fmt.Sprintf("%s ILIKE ?", fieldRef), fmt.Sprintf("%v%%", value))
	case ports.OpEndsWith:
		qb.query = qb.query.Where(fmt.Sprintf("%s ILIKE ?", fieldRef), fmt.Sprintf("%%%v", value))
	case ports.OpLike:
		qb.query = qb.query.Where(fmt.Sprintf("%s LIKE ?", fieldRef), value)
	case ports.OpILike:
		qb.query = qb.query.Where(fmt.Sprintf("%s ILIKE ?", fieldRef), value)
	case ports.OpIn:
		qb.query = qb.query.Where(fmt.Sprintf("%s IN (?)", fieldRef), bun.In(value))
	case ports.OpNotIn:
		qb.query = qb.query.Where(fmt.Sprintf("%s NOT IN (?)", fieldRef), bun.In(value))
	case ports.OpIsNull:
		qb.query = qb.query.Where(fmt.Sprintf("%s IS NULL", fieldRef))
	case ports.OpIsNotNull:
		qb.query = qb.query.Where(fmt.Sprintf("%s IS NOT NULL", fieldRef))
	case ports.OpDateRange:
		qb.applyDateRangeFilter(fieldRef, value)
	}
}

// isStringOperation checks if the operator requires string operations
func (qb *QueryBuilder) isStringOperation(operator ports.FilterOperator) bool {
	switch operator { //nolint:exhaustive // We only support the operators we need
	case ports.OpContains, ports.OpStartsWith, ports.OpEndsWith, ports.OpLike, ports.OpILike:
		return true
	default:
		return false
	}
}

// isEmptyEnumValue checks if the value is an empty string for enum fields
func (qb *QueryBuilder) isEmptyEnumValue(value any) bool {
	if str, ok := value.(string); ok && str == "" {
		return true
	}
	return false
}

// applyDateRangeFilter applies a date range filter
func (qb *QueryBuilder) applyDateRangeFilter(fieldRef string, value any) {
	// Expected format: {"start": "2024-01-01", "end": "2024-12-31"}
	if dateRange, ok := value.(map[string]any); ok {
		if startStr, hasStart := dateRange["start"]; hasStart {
			if start, err := time.Parse("2006-01-02", fmt.Sprintf("%v", startStr)); err == nil {
				qb.query = qb.query.Where(fmt.Sprintf("%s >= ?", fieldRef), start)
			}
		}
		if endStr, hasEnd := dateRange["end"]; hasEnd {
			if end, err := time.Parse("2006-01-02", fmt.Sprintf("%v", endStr)); err == nil {
				// Add 1 day to make it inclusive of the end date
				endInclusive := end.AddDate(0, 0, 1)
				qb.query = qb.query.Where(fmt.Sprintf("%s < ?", fieldRef), endInclusive)
			}
		}
	}
}

// getFieldReference returns the properly formatted field reference for SQL
func (qb *QueryBuilder) getFieldReference(dbField string) string {
	if qb.tableAlias != "" {
		return fmt.Sprintf("%s.%s", qb.tableAlias, dbField)
	}
	return dbField
}

// ApplyDefaultFilters applies common filters like tenant filtering
func (qb *QueryBuilder) ApplyDefaultFilters(
	tenantOpts ports.TenantOptions,
) *QueryBuilder {
	qb.query = qb.query.Where(
		fmt.Sprintf("%s.organization_id = ?", qb.tableAlias),
		tenantOpts.OrgID,
	)
	qb.query = qb.query.Where(
		fmt.Sprintf("%s.business_unit_id = ?", qb.tableAlias),
		tenantOpts.BuID,
	)
	return qb
}

// ApplyPagination applies limit and offset to the query
func (qb *QueryBuilder) ApplyPagination(limit, offset int) *QueryBuilder {
	if limit > 0 {
		qb.query = qb.query.Limit(limit)
	}
	if offset > 0 {
		qb.query = qb.query.Offset(offset)
	}
	return qb
}

// ApplyTextSearch applies full-text search if query is provided
func (qb *QueryBuilder) ApplyTextSearch(
	searchQuery string,
	searchableFields []string,
) *QueryBuilder {
	if searchQuery == "" || len(searchableFields) == 0 {
		return qb
	}

	// Build ILIKE conditions for each searchable field
	var conditions []string
	var args []any

	for _, field := range searchableFields {
		dbField := qb.getDBField(field)
		fieldRef := qb.getFieldReference(dbField)

		// Cast enum fields to text for ILIKE operations
		if qb.isEnumField(field) {
			fieldRef = fmt.Sprintf("%s::text", fieldRef)
		}

		conditions = append(conditions, fmt.Sprintf("%s ILIKE ?", fieldRef))
		args = append(args, fmt.Sprintf("%%%s%%", searchQuery))
	}

	if len(conditions) > 0 {
		whereClause := fmt.Sprintf("(%s)", strings.Join(conditions, " OR "))
		qb.query = qb.query.Where(whereClause, args...)
	}

	return qb
}
