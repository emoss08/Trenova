/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package querybuilder

import (
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/infra"
	"github.com/uptrace/bun"
)

// Constants for search configuration
const (
	defaultSimilarityThreshold = 0.3
	wildcardPattern            = "%"
)

// QueryBuilder provides utilities for building queries with filtering and sorting
type QueryBuilder struct {
	query             *bun.SelectQuery
	tableAlias        string
	fieldConfig       *ports.FieldConfiguration
	searchConfig      *infra.PostgresSearchConfig
	usePostgresSearch bool
	appliedSorts      []ports.SortField // Track user-specified sorts
	appliedJoins      map[string]bool   // Track applied joins to avoid duplicates
}

// New creates a new QueryBuilder
func New(
	query *bun.SelectQuery,
	tableAlias string,
	fieldConfig *ports.FieldConfiguration,
) *QueryBuilder {
	return &QueryBuilder{
		query:             query,
		tableAlias:        tableAlias,
		fieldConfig:       fieldConfig,
		usePostgresSearch: false,
		appliedSorts:      make([]ports.SortField, 0),
		appliedJoins:      make(map[string]bool),
	}
}

// NewWithPostgresSearch creates a new QueryBuilder with postgres search configuration
func NewWithPostgresSearch[T infra.PostgresSearchable](
	query *bun.SelectQuery,
	tableAlias string,
	fieldConfig *ports.FieldConfiguration,
	entity T,
) *QueryBuilder {
	config := entity.GetPostgresSearchConfig()
	return &QueryBuilder{
		query:             query,
		tableAlias:        tableAlias,
		fieldConfig:       fieldConfig,
		searchConfig:      &config,
		usePostgresSearch: true,
		appliedSorts:      make([]ports.SortField, 0),
		appliedJoins:      make(map[string]bool),
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

		// If postgres search is enabled, store sorts for later integration
		if qb.usePostgresSearch {
			qb.appliedSorts = append(qb.appliedSorts, sort)
			continue
		}

		// Otherwise apply sorts immediately (traditional behavior)
		dbField := qb.getDBField(sort.Field)
		direction := strings.ToUpper(string(sort.Direction))

		// Check if the field is already qualified (contains a dot)
		switch {
		case strings.Contains(dbField, "."):
			qb.query = qb.query.Order(fmt.Sprintf("%s %s", dbField, direction))
		case qb.tableAlias != "":
			qb.query = qb.query.Order(fmt.Sprintf("%s.%s %s", qb.tableAlias, dbField, direction))
		default:
			qb.query = qb.query.Order(fmt.Sprintf("%s %s", dbField, direction))
		}
	}
	return qb
}

// GetQuery returns the modified query
func (qb *QueryBuilder) GetQuery() *bun.SelectQuery {
	// If postgres search is enabled but no search was performed,
	// we still need to apply any pending sorts
	if qb.usePostgresSearch && len(qb.appliedSorts) > 0 {
		qb.applyPendingSorts()
	}
	return qb.query
}

// applyPendingSorts applies any stored sorts (used when postgres search is enabled but no search query was provided)
func (qb *QueryBuilder) applyPendingSorts() {
	for _, sort := range qb.appliedSorts {
		dbField := qb.getDBField(sort.Field)
		direction := strings.ToUpper(string(sort.Direction))

		// Check if the field is already qualified (contains a dot)
		switch {
		case strings.Contains(dbField, "."):
			qb.query = qb.query.Order(fmt.Sprintf("%s %s", dbField, direction))
		case qb.tableAlias != "":
			qb.query = qb.query.Order(fmt.Sprintf("%s.%s %s", qb.tableAlias, dbField, direction))
		default:
			qb.query = qb.query.Order(fmt.Sprintf("%s %s", dbField, direction))
		}
	}
	// Clear applied sorts to avoid double application
	qb.appliedSorts = make([]ports.SortField, 0)
}

// getDBField gets the database field name from the API field name
func (qb *QueryBuilder) getDBField(apiField string) string {
	// Check if it's a nested field first
	if nestedDef, exists := qb.fieldConfig.NestedFields[apiField]; exists {
		// Apply required joins
		qb.applyNestedFieldJoins(nestedDef.RequiredJoins)
		return nestedDef.DatabaseField
	}

	// Check regular field mapping
	if dbField, exists := qb.fieldConfig.FieldMap[apiField]; exists {
		return dbField
	}

	return apiField
}

// applyNestedFieldJoins applies the required joins for a nested field
func (qb *QueryBuilder) applyNestedFieldJoins(joins []ports.JoinDefinition) {
	for _, join := range joins {
		// Create a unique key for this join
		joinKey := fmt.Sprintf("%s_%s", join.Table, join.Alias)

		// Skip if we've already applied this join
		if qb.appliedJoins[joinKey] {
			continue
		}

		// Apply the join based on type
		switch join.JoinType {
		case ports.JoinTypeLeft:
			qb.query = qb.query.Join(
				fmt.Sprintf("LEFT JOIN %s AS %s ON %s", join.Table, join.Alias, join.Condition),
			)
		case ports.JoinTypeRight:
			qb.query = qb.query.Join(
				fmt.Sprintf("RIGHT JOIN %s AS %s ON %s", join.Table, join.Alias, join.Condition),
			)
		case ports.JoinTypeInner, ports.JoinTypeNone:
			qb.query = qb.query.Join(
				fmt.Sprintf("INNER JOIN %s AS %s ON %s", join.Table, join.Alias, join.Condition),
			)
		default:
			qb.query = qb.query.Join(
				fmt.Sprintf(
					"%s JOIN %s AS %s ON %s",
					join.JoinType,
					join.Table,
					join.Alias,
					join.Condition,
				),
			)
		}

		// Mark this join as applied
		qb.appliedJoins[joinKey] = true
	}
}

// isEnumField checks if a field is an enum field
func (qb *QueryBuilder) isEnumField(apiField string) bool {
	// Check if it's a nested field first
	if nestedDef, exists := qb.fieldConfig.NestedFields[apiField]; exists {
		return nestedDef.IsEnum
	}

	// Check regular enum mapping
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
	case ports.OpContains,
		ports.OpEqual,
		ports.OpStartsWith,
		ports.OpEndsWith,
		ports.OpLike,
		ports.OpILike:
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
				// * Always convert to Unix timestamp since all date fields are bigints
				qb.query = qb.query.Where(fmt.Sprintf("%s >= ?", fieldRef), start.Unix())
			}
		}
		if endStr, hasEnd := dateRange["end"]; hasEnd {
			if end, err := time.Parse("2006-01-02", fmt.Sprintf("%v", endStr)); err == nil {
				// Add 1 day to make it inclusive of the end date
				endInclusive := end.AddDate(0, 0, 1)
				// * Always convert to Unix timestamp since all date fields are bigints
				qb.query = qb.query.Where(fmt.Sprintf("%s < ?", fieldRef), endInclusive.Unix())
			}
		}
	}
}

// getFieldReference returns the properly formatted field reference for SQL
func (qb *QueryBuilder) getFieldReference(dbField string) string {
	// If the field already contains a dot, it's already qualified (nested field)
	if strings.Contains(dbField, ".") {
		return dbField
	}

	// Otherwise, qualify it with the table alias
	if qb.tableAlias != "" {
		return fmt.Sprintf("%s.%s", qb.tableAlias, dbField)
	}
	return dbField
}

// ApplyTenantFilters applies common filters like tenant filtering
func (qb *QueryBuilder) ApplyTenantFilters(tenantOpts ports.TenantOptions) *QueryBuilder {
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

// ApplyTextSearch applies full-text search using postgres search capabilities or fallback to ILIKE
func (qb *QueryBuilder) ApplyTextSearch(
	searchQuery string,
	searchableFields []string,
) *QueryBuilder {
	if searchQuery == "" {
		return qb
	}

	// Use postgres search if configured, otherwise fallback to ILIKE
	if qb.usePostgresSearch && qb.searchConfig != nil {
		return qb.applyPostgresSearch(searchQuery)
	}

	return qb.applyFallbackSearch(searchQuery, searchableFields)
}

// applyPostgresSearch applies the sophisticated postgres search functionality
func (qb *QueryBuilder) applyPostgresSearch(query string) *QueryBuilder {
	config := qb.searchConfig

	// Check minimum query length
	if len(strings.TrimSpace(query)) < config.MinLength {
		return qb
	}

	// Limit number of terms
	terms := strings.Fields(query)
	if len(terms) > config.MaxTerms {
		terms = terms[:config.MaxTerms]
	}

	// Build tsquery exactly like postgressearch.go
	var tsQueryBuilder strings.Builder
	tsQueryBuilder.Grow(len(query) + len(terms)*3)

	for i, term := range terms {
		if i > 0 {
			tsQueryBuilder.WriteString(" | ")
		}
		tsQueryBuilder.WriteString(term)
	}
	tsqueryStr := tsQueryBuilder.String()
	tsqueryWithWildcard := tsqueryStr + ":*"

	tableAliasWithDot := qb.getTableAliasWithDot()

	// Follow the exact same pattern as postgressearch.go:
	// 1. First explicitly select table columns
	qb.query = qb.query.ColumnExpr(tableAliasWithDot + "*")

	// 2. Then add rank column
	rankExpr := fmt.Sprintf(
		`ts_rank(%ssearch_vector, to_tsquery('simple', ?)) AS rank`,
		tableAliasWithDot,
	)
	qb.query = qb.query.ColumnExpr(rankExpr, tsqueryWithWildcard)

	// 3. Build and apply search conditions
	whereParts, whereArgs := qb.buildSearchConditions(config, tableAliasWithDot, query, tsqueryStr)

	// Apply search conditions exactly like postgressearch.go
	if len(whereParts) > 0 {
		var searchCondBuilder strings.Builder
		searchCondBuilder.WriteString("(")
		for i, part := range whereParts {
			if i > 0 {
				searchCondBuilder.WriteString(" OR ")
			}
			searchCondBuilder.WriteString(part)
		}
		searchCondBuilder.WriteString(")")
		qb.query = qb.query.Where(searchCondBuilder.String(), whereArgs...)
	}

	// 4. Apply ordering exactly like postgressearch.go
	orderParts, orderArgs := qb.buildOrderingConditions(config, tableAliasWithDot, query)

	for i, orderPart := range orderParts {
		if i < len(orderArgs) {
			qb.query = qb.query.OrderExpr(orderPart, orderArgs[i])
		} else {
			qb.query = qb.query.OrderExpr(orderPart)
		}
	}

	// Clear applied sorts since they've been integrated into search ordering
	qb.appliedSorts = make([]ports.SortField, 0)

	return qb
}

// buildSearchConditions builds the WHERE conditions for postgres search
func (qb *QueryBuilder) buildSearchConditions(
	config *infra.PostgresSearchConfig,
	tableAliasWithDot, query, tsqueryStr string,
) (whereParts []string, whereArgs []any) {
	whereParts = make([]string, 0, len(config.Fields)+1)
	whereArgs = make([]any, 0, len(config.Fields)*2+1)

	// Primary text search vector condition
	whereParts = append(whereParts,
		fmt.Sprintf("%ssearch_vector @@ to_tsquery('simple', ?)", tableAliasWithDot))
	whereArgs = append(whereArgs, tsqueryStr+":*")

	// Additional field-specific conditions for partial matching
	if config.UsePartialMatch {
		queryWithWildcards := wildcardPattern + query + wildcardPattern

		for _, field := range config.Fields {
			switch field.Type {
			case infra.PostgresSearchTypeArray:
				whereParts = append(whereParts,
					fmt.Sprintf("%s%s @> ?", tableAliasWithDot, field.Name))
				whereArgs = append(whereArgs, queryWithWildcards)
			case infra.PostgresSearchTypeComposite, infra.PostgresSearchTypeNumber:
				whereParts = append(whereParts,
					fmt.Sprintf("%s%s ILIKE ?", tableAliasWithDot, field.Name))
				whereArgs = append(whereArgs, queryWithWildcards)
			case infra.PostgresSearchTypeText:
				whereParts = append(whereParts,
					fmt.Sprintf("(%s%s ILIKE ? OR similarity(%s%s, ?) > %g)",
						tableAliasWithDot, field.Name,
						tableAliasWithDot, field.Name, defaultSimilarityThreshold))
				whereArgs = append(whereArgs, queryWithWildcards, query)
			case infra.PostgresSearchTypeEnum:
				whereParts = append(whereParts,
					fmt.Sprintf("%s%s::text = ?", tableAliasWithDot, field.Name))
				whereArgs = append(whereArgs, query)
			}
		}
	}

	return whereParts, whereArgs
}

// buildOrderingConditions builds the ORDER BY conditions for postgres search
func (qb *QueryBuilder) buildOrderingConditions(
	config *infra.PostgresSearchConfig,
	tableAliasWithDot, query string,
) (orderParts []string, orderArgs []any) {
	// Calculate total capacity: search ordering + user sorts
	searchOrderCount := len(config.Fields)*2 + 1 // exact + prefix + rank
	userSortCount := len(qb.appliedSorts)
	totalCapacity := searchOrderCount + userSortCount

	orderParts = make([]string, 0, totalCapacity)
	orderArgs = make([]any, 0, len(config.Fields)*2) // Only search ordering needs args

	// 1. Search-specific ordering (highest priority)
	// Exact match priority
	for _, field := range config.Fields {
		if field.Type == infra.PostgresSearchTypeComposite ||
			field.Type == infra.PostgresSearchTypeNumber {
			orderParts = append(orderParts,
				fmt.Sprintf("CASE WHEN %s%s = ? THEN 1 ELSE 0 END DESC",
					tableAliasWithDot, field.Name))
			orderArgs = append(orderArgs, query)
		}
	}

	// Prefix match priority
	queryWithSuffix := query + wildcardPattern
	for _, field := range config.Fields {
		if field.Type == infra.PostgresSearchTypeComposite ||
			field.Type == infra.PostgresSearchTypeNumber {
			orderParts = append(orderParts,
				fmt.Sprintf("CASE WHEN %s%s ILIKE ? THEN 1 ELSE 0 END DESC",
					tableAliasWithDot, field.Name))
			orderArgs = append(orderArgs, queryWithSuffix)
		}
	}

	// Relevance ranking
	orderParts = append(orderParts, "rank DESC NULLS LAST")

	// 2. User-specified sorts (secondary priority for tie-breaking)
	for _, sort := range qb.appliedSorts {
		dbField := qb.getDBField(sort.Field)
		direction := strings.ToUpper(string(sort.Direction))

		fieldRef := qb.getFieldReference(dbField)
		orderParts = append(orderParts, fmt.Sprintf("%s %s", fieldRef, direction))
	}

	return orderParts, orderArgs
}

// applyFallbackSearch applies the original ILIKE-based search as fallback
func (qb *QueryBuilder) applyFallbackSearch(
	searchQuery string,
	searchableFields []string,
) *QueryBuilder {
	if len(searchableFields) == 0 {
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

// getTableAliasWithDot returns the table alias with dot suffix
func (qb *QueryBuilder) getTableAliasWithDot() string {
	if qb.tableAlias != "" {
		return qb.tableAlias + "."
	}
	return ""
}
