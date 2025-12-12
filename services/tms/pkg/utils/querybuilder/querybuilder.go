package querybuilder

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/uptrace/bun"
)

type Option func(*QueryBuilder) *QueryBuilder

func WithFieldConfig(fieldConfig *pagination.FieldConfiguration) Option {
	return func(qb *QueryBuilder) *QueryBuilder {
		qb.fieldConfig = fieldConfig
		return qb
	}
}

func WithEntity(entity any) Option {
	return func(qb *QueryBuilder) *QueryBuilder {
		qb.entity = entity
		return qb
	}
}

type QueryBuilder struct {
	query            *bun.SelectQuery
	tableAlias       string
	fieldConfig      *pagination.FieldConfiguration
	searchConfig     *domaintypes.PostgresSearchConfig
	appliedSorts     []pagination.SortField
	appliedJoins     map[string]bool
	traversalEnabled bool
	entity           any
}

func New(
	query *bun.SelectQuery,
	tableAlias string,
	opts ...Option,
) *QueryBuilder {
	qb := &QueryBuilder{
		query:            query,
		tableAlias:       tableAlias,
		appliedSorts:     make([]pagination.SortField, 0),
		appliedJoins:     make(map[string]bool),
		traversalEnabled: false,
	}

	for _, opt := range opts {
		qb = opt(qb)
	}

	return qb
}

func NewWithPostgresSearch[T domaintypes.PostgresSearchable](
	query *bun.SelectQuery,
	tableAlias string,
	fieldConfig *pagination.FieldConfiguration,
	entity T,
) *QueryBuilder {
	config := entity.GetPostgresSearchConfig()
	return &QueryBuilder{
		query:            query,
		tableAlias:       tableAlias,
		fieldConfig:      fieldConfig,
		searchConfig:     &config,
		appliedSorts:     make([]pagination.SortField, 0),
		appliedJoins:     make(map[string]bool),
		traversalEnabled: false,
		entity:           entity,
	}
}

func (qb *QueryBuilder) ApplyFilters(filters []pagination.FieldFilter) *QueryBuilder {
	for _, filter := range filters {
		field := filter.Field

		if qb.traversalEnabled && strings.Contains(field, ".") {
			qb.applyTraversalFilter(field, filter)
		} else {
			if !qb.fieldConfig.FilterableFields[field] {
				continue
			}
			dbField := qb.getDBField(field)
			isEnum := qb.isEnumField(field)
			qb.applyFilter(dbField, filter.Operator, filter.Value, isEnum)
		}
	}
	return qb
}

func (qb *QueryBuilder) ApplySort(sorts []pagination.SortField) *QueryBuilder {
	for _, sort := range sorts {
		if !qb.fieldConfig.SortableFields[sort.Field] {
			continue
		}

		// Handle traversal-enabled custom joins for sorting
		if qb.traversalEnabled && strings.Contains(sort.Field, ".") && qb.entity != nil &&
			qb.searchConfig != nil {
			qb.applyTraversalSort(sort)
		} else {
			qb.applySortField(sort)
		}
	}
	return qb
}

// applyTraversalSort handles sorting on custom relationship fields
func (qb *QueryBuilder) applyTraversalSort(sort pagination.SortField) {
	sqlField, joinDefs := qb.resolveNestedField(sort.Field)
	if len(joinDefs) == 0 {
		qb.applySortField(sort)
		return
	}

	// Apply joins if not already applied
	for _, joinDef := range joinDefs {
		joinKey := fmt.Sprintf("%s_%s", joinDef.Table, joinDef.Alias)
		if !qb.appliedJoins[joinKey] {
			qb.applyNestedFieldJoins([]pagination.JoinDefinition{joinDef})
			qb.appliedJoins[joinKey] = true
		}
	}

	// Apply sort on the resolved field
	if sqlField != "" {
		direction := strings.ToUpper(string(sort.Direction))
		qb.query = qb.query.Order(fmt.Sprintf("%s %s", sqlField, direction))
	}
}

func (qb *QueryBuilder) WithTraversalSupport(enabled bool) *QueryBuilder {
	qb.traversalEnabled = enabled
	return qb
}

func (qb *QueryBuilder) GetQuery() *bun.SelectQuery {
	return qb.query
}

func (qb *QueryBuilder) applySortField(sort pagination.SortField) {
	dbField := qb.getDBField(sort.Field)
	direction := strings.ToUpper(string(sort.Direction))

	switch {
	case strings.Contains(dbField, "."):
		qb.query = qb.query.Order(fmt.Sprintf("%s %s", dbField, direction))
	case qb.tableAlias != "":
		qb.query = qb.query.Order(fmt.Sprintf("%s.%s %s", qb.tableAlias, dbField, direction))
	default:
		qb.query = qb.query.Order(fmt.Sprintf("%s %s", dbField, direction))
	}
}

func (qb *QueryBuilder) getDBField(apiField string) string {
	if nestedDef, exists := qb.fieldConfig.NestedFields[apiField]; exists {
		qb.applyNestedFieldJoins(nestedDef.RequiredJoins)
		return nestedDef.DatabaseField
	}

	if dbField, exists := qb.fieldConfig.FieldMap[apiField]; exists {
		return dbField
	}

	return apiField
}

func (qb *QueryBuilder) applyNestedFieldJoins(joins []pagination.JoinDefinition) {
	for _, join := range joins {
		joinKey := fmt.Sprintf("%s_%s", join.Table, join.Alias)
		if qb.appliedJoins[joinKey] {
			continue
		}

		var joinType string
		switch join.JoinType {
		case pagination.JoinTypeLeft:
			joinType = "LEFT"
		case pagination.JoinTypeRight:
			joinType = "RIGHT"
		case pagination.JoinTypeInner, pagination.JoinTypeNone:
			joinType = "INNER"
		default:
			joinType = string(join.JoinType)
		}

		joinClause := fmt.Sprintf("%s JOIN %s AS %s ON %s",
			joinType, join.Table, join.Alias, join.Condition)
		qb.query = qb.query.Join(joinClause)
		qb.appliedJoins[joinKey] = true
	}
}

func (qb *QueryBuilder) isEnumField(apiField string) bool {
	if nestedDef, exists := qb.fieldConfig.NestedFields[apiField]; exists {
		return nestedDef.IsEnum
	}

	_, exists := qb.fieldConfig.EnumMap[apiField]
	return exists
}

func (qb *QueryBuilder) applyFilter(
	dbField string,
	operator pagination.FilterOperator,
	value any,
	isEnum bool,
) {
	fieldRef := qb.getFieldReference(dbField)

	if isEnum &&
		(qb.isStringOperation(operator) || operator == pagination.OpIn || operator == pagination.OpNotIn) {
		fieldRef = fmt.Sprintf("%s::text", fieldRef)
	}

	qb.applyStatement(operator, fieldRef, value)
}

func (qb *QueryBuilder) applyStatement(
	operator pagination.FilterOperator,
	fieldRef string,
	value any,
) {
	switch operator {
	case pagination.OpEqual:
		if qb.isEmptyEnumValue(value) {
			return
		}
		qb.query = qb.query.Where(fmt.Sprintf("%s = ?", fieldRef), value)
	case pagination.OpNotEqual:
		if qb.isEmptyEnumValue(value) {
			return
		}
		qb.query = qb.query.Where(fmt.Sprintf("%s != ?", fieldRef), value)
	case pagination.OpGreaterThan:
		qb.query = qb.query.Where(fmt.Sprintf("%s > ?", fieldRef), value)
	case pagination.OpGreaterThanOrEqual:
		qb.query = qb.query.Where(fmt.Sprintf("%s >= ?", fieldRef), value)
	case pagination.OpLessThan:
		qb.query = qb.query.Where(fmt.Sprintf("%s < ?", fieldRef), value)
	case pagination.OpLessThanOrEqual:
		qb.query = qb.query.Where(fmt.Sprintf("%s <= ?", fieldRef), value)
	case pagination.OpContains:
		qb.query = qb.query.Where(
			fmt.Sprintf("%s ILIKE ?", fieldRef),
			fmt.Sprintf("%%%v%%", value),
		)
	case pagination.OpStartsWith:
		qb.query = qb.query.Where(fmt.Sprintf("%s ILIKE ?", fieldRef), fmt.Sprintf("%v%%", value))
	case pagination.OpEndsWith:
		qb.query = qb.query.Where(fmt.Sprintf("%s ILIKE ?", fieldRef), fmt.Sprintf("%%%v", value))
	case pagination.OpLike:
		qb.query = qb.query.Where(fmt.Sprintf("%s LIKE ?", fieldRef), value)
	case pagination.OpILike:
		qb.query = qb.query.Where(fmt.Sprintf("%s ILIKE ?", fieldRef), value)
	case pagination.OpIn:
		qb.query = qb.query.Where(fmt.Sprintf("%s IN (?)", fieldRef), bun.In(value))
	case pagination.OpNotIn:
		qb.query = qb.query.Where(fmt.Sprintf("%s NOT IN (?)", fieldRef), bun.In(value))
	case pagination.OpIsNull:
		qb.query = qb.query.Where(fmt.Sprintf("%s IS NULL", fieldRef))
	case pagination.OpIsNotNull:
		qb.query = qb.query.Where(fmt.Sprintf("%s IS NOT NULL", fieldRef))
	case pagination.OpDateRange:
		qb.applyDateRangeFilter(fieldRef, value)
	}
}

func (qb *QueryBuilder) isStringOperation(operator pagination.FilterOperator) bool {
	switch operator { //nolint:exhaustive // We only support the operators we need
	case pagination.OpContains,
		pagination.OpEqual,
		pagination.OpStartsWith,
		pagination.OpEndsWith,
		pagination.OpLike,
		pagination.OpILike:
		return true
	default:
		return false
	}
}

func (qb *QueryBuilder) isEmptyEnumValue(value any) bool {
	if str, ok := value.(string); ok && str == "" {
		return true
	}
	return false
}

func (qb *QueryBuilder) applyDateRangeFilter(fieldRef string, value any) {
	dateRange, ok := value.(map[string]any)
	if !ok {
		return
	}

	qb.applyStartDateFilter(fieldRef, dateRange)
	qb.applyEndDateFilter(fieldRef, dateRange)
}

func (qb *QueryBuilder) applyStartDateFilter(fieldRef string, dateRange map[string]any) {
	startStr, hasStart := dateRange["start"]
	if !hasStart {
		return
	}

	start, err := time.Parse("2006-01-02", fmt.Sprintf("%v", startStr))
	if err == nil {
		qb.query = qb.query.Where(fmt.Sprintf("%s >= ?", fieldRef), start.Unix())
	}
}

func (qb *QueryBuilder) applyEndDateFilter(fieldRef string, dateRange map[string]any) {
	endStr, hasEnd := dateRange["end"]
	if !hasEnd {
		return
	}

	end, err := time.Parse("2006-01-02", fmt.Sprintf("%v", endStr))
	if err == nil {
		endInclusive := end.AddDate(0, 0, 1)
		qb.query = qb.query.Where(fmt.Sprintf("%s < ?", fieldRef), endInclusive.Unix())
	}
}

func (qb *QueryBuilder) applyTraversalFilter(fieldPath string, filter pagination.FieldFilter) {
	parts := strings.Split(fieldPath, ".")
	if len(parts) < 2 {
		return
	}

	if nestedDef, exists := qb.fieldConfig.NestedFields[fieldPath]; exists {
		qb.applyNestedFieldJoins(nestedDef.RequiredJoins)
		qb.applyFilter(nestedDef.DatabaseField, filter.Operator, filter.Value, nestedDef.IsEnum)
		return
	}

	if qb.entity != nil && qb.searchConfig != nil {
		qb.applyEntityTraversalFilter(fieldPath, filter)
	}
}

func (qb *QueryBuilder) applyEntityTraversalFilter(
	fieldPath string,
	filter pagination.FieldFilter,
) {
	sqlField, joinDefs := qb.resolveNestedField(fieldPath)
	if len(joinDefs) == 0 {
		return
	}

	for _, joinDef := range joinDefs {
		joinKey := fmt.Sprintf("%s_%s", joinDef.Table, joinDef.Alias)
		if !qb.appliedJoins[joinKey] {
			qb.applyNestedFieldJoins([]pagination.JoinDefinition{joinDef})
			qb.appliedJoins[joinKey] = true
		}
	}

	if sqlField != "" {
		// Check if this is an enum field in a custom relationship
		isEnum := qb.isCustomRelationshipEnum(fieldPath)
		qb.applyFilter(sqlField, filter.Operator, filter.Value, isEnum)
	}
}

// isCustomRelationshipEnum checks if a field path references an enum in a custom relationship
func (qb *QueryBuilder) isCustomRelationshipEnum(fieldPath string) bool {
	if qb.searchConfig == nil {
		return false
	}

	parts := strings.Split(fieldPath, ".")
	if len(parts) < 2 {
		return false
	}

	relName := parts[0]
	for i := range qb.searchConfig.Relationships {
		rel := qb.searchConfig.Relationships[i]
		if strings.EqualFold(rel.Field, relName) && rel.Type == domaintypes.RelationshipTypeCustom {
			return rel.IsEnum
		}
	}

	return false
}

func (qb *QueryBuilder) resolveNestedField(fieldPath string) (string, []pagination.JoinDefinition) {
	parts := strings.Split(fieldPath, ".")
	if len(parts) < 2 || qb.searchConfig == nil {
		return fieldPath, nil
	}

	// First, try to find a custom relationship that matches the first part (e.g., "originLocation")
	relName := parts[0]
	var rel *domaintypes.RelationshipDefinition
	for i := range qb.searchConfig.Relationships {
		if strings.EqualFold(qb.searchConfig.Relationships[i].Field, relName) {
			rel = qb.searchConfig.Relationships[i]
			break
		}
	}

	if rel == nil {
		return "", nil
	}

	var joinDefs []pagination.JoinDefinition

	switch rel.Type {
	case domaintypes.RelationshipTypeCustom:
		// Handle custom multi-hop joins
		for _, step := range rel.CustomJoinPath {
			joinDef := pagination.JoinDefinition{
				Table:     step.Table,
				Alias:     step.Alias,
				Condition: step.Condition,
			}

			// Convert JoinType
			switch step.JoinType {
			case domaintypes.JoinTypeLeft:
				joinDef.JoinType = pagination.JoinTypeLeft
			case domaintypes.JoinTypeRight:
				joinDef.JoinType = pagination.JoinTypeRight
			case domaintypes.JoinTypeInner:
				joinDef.JoinType = pagination.JoinTypeInner
			default:
				joinDef.JoinType = pagination.JoinTypeLeft
			}

			joinDefs = append(joinDefs, joinDef)
		}

		// Get the field name from the path (e.g., "name" from "originLocation.name")
		fieldName := parts[len(parts)-1]

		// If TargetField is specified, use it; otherwise convert from camelCase
		targetFieldName := rel.TargetField
		if targetFieldName == "" {
			targetFieldName = utils.ConvertCamelToSnake(fieldName)
		}

		// Use the alias from the relationship definition, or fall back to the final join step alias
		finalAlias := rel.Alias
		if finalAlias == "" && len(rel.CustomJoinPath) > 0 {
			finalAlias = rel.CustomJoinPath[len(rel.CustomJoinPath)-1].Alias
		}

		if finalAlias != "" {
			sqlField := fmt.Sprintf("%s.%s", finalAlias, targetFieldName)
			return sqlField, joinDefs
		}

		return "", nil

	case domaintypes.RelationshipTypeBelongsTo:
		joinDef := pagination.JoinDefinition{
			Table:    rel.TargetTable,
			Alias:    rel.Alias,
			JoinType: pagination.JoinTypeLeft,
			Condition: fmt.Sprintf("%s.%s = %s.%s",
				qb.tableAlias, rel.ForeignKey,
				rel.Alias, rel.ReferenceKey),
		}
		joinDefs = append(joinDefs, joinDef)

	case domaintypes.RelationshipTypeHasOne, domaintypes.RelationshipTypeHasMany:
		joinDef := pagination.JoinDefinition{
			Table:    rel.TargetTable,
			Alias:    rel.Alias,
			JoinType: pagination.JoinTypeLeft,
			Condition: fmt.Sprintf("%s.%s = %s.%s",
				rel.Alias, rel.ForeignKey,
				qb.tableAlias, rel.ReferenceKey),
		}
		joinDefs = append(joinDefs, joinDef)

	case domaintypes.RelationshipTypeManyToMany:
		joinTableDef := pagination.JoinDefinition{
			Table:    rel.JoinTable,
			Alias:    rel.JoinTableAlias,
			JoinType: pagination.JoinTypeLeft,
			Condition: fmt.Sprintf("%s.%s = %s.%s",
				qb.tableAlias, rel.ReferenceKey,
				rel.JoinTableAlias, rel.JoinTableSourceKey),
		}
		joinDefs = append(joinDefs, joinTableDef)

		targetTableDef := pagination.JoinDefinition{
			Table:    rel.TargetTable,
			Alias:    rel.Alias,
			JoinType: pagination.JoinTypeLeft,
			Condition: fmt.Sprintf("%s.%s = %s.%s",
				rel.JoinTableAlias, rel.JoinTableTargetKey,
				rel.Alias, rel.ReferenceKey),
		}
		joinDefs = append(joinDefs, targetTableDef)

	default:
		return "", nil
	}

	sqlFieldName := utils.ConvertCamelToSnake(parts[len(parts)-1])
	sqlField := fmt.Sprintf("%s.%s", rel.Alias, sqlFieldName)
	return sqlField, joinDefs
}

func (qb *QueryBuilder) getFieldReference(dbField string) string {
	if strings.Contains(dbField, ".") {
		return dbField
	}

	if qb.tableAlias != "" {
		return fmt.Sprintf("%s.%s", qb.tableAlias, dbField)
	}
	return dbField
}

func (qb *QueryBuilder) ApplyTenantFilters(tenantOpts pagination.TenantOptions) *QueryBuilder {
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

func (qb *QueryBuilder) ApplyTextSearch(
	searchQuery string,
	searchableFields []string,
) *QueryBuilder {
	if searchQuery == "" {
		return qb
	}

	if qb.searchConfig != nil && qb.searchConfig.UseSearchVector {
		return qb.applyPostgresTextSearch(searchQuery)
	}

	return qb.applySimpleSearch(searchQuery, searchableFields)
}

func (qb *QueryBuilder) applyPostgresTextSearch(searchQuery string) *QueryBuilder {
	if qb.searchConfig == nil || searchQuery == "" {
		return qb
	}

	config := qb.searchConfig
	tableAlias := qb.tableAlias
	if tableAlias != "" {
		tableAlias += "."
	}

	if config.UseSearchVector {
		qb.buildSearchVectorConditions(searchQuery, tableAlias)
	} else {
		qb.buildFieldSearchConditions(searchQuery, tableAlias)
	}

	qb.applySearchOrdering(searchQuery)

	return qb
}

func (qb *QueryBuilder) buildSearchVectorConditions(searchQuery, tableAlias string) {
	config := qb.searchConfig
	vectorCol := config.SearchVectorColumn
	if vectorCol == "" {
		vectorCol = "search_vector"
	}

	qb.query = qb.query.ColumnExpr(tableAlias + "*")

	usesAdvancedSyntax := qb.hasWebsearchOperators(searchQuery)

	if !usesAdvancedSyntax && len(searchQuery) >= 2 {
		qb.applyWebsearchWithPrefix(searchQuery, tableAlias, vectorCol)
	} else {
		qb.applyWebsearchOnly(searchQuery, tableAlias, vectorCol)
	}
}

func (qb *QueryBuilder) hasWebsearchOperators(query string) bool {
	if strings.Contains(query, "\"") {
		return true
	}

	if strings.Contains(query, " OR ") || strings.Contains(query, " or ") {
		return true
	}

	words := strings.FieldsSeq(query)
	for word := range words {
		if strings.HasPrefix(word, "-") && len(word) > 1 {
			return true
		}
	}

	return false
}

func (qb *QueryBuilder) applyWebsearchWithPrefix(searchQuery, tableAlias, vectorCol string) {
	words := strings.Fields(searchQuery)
	if len(words) == 0 {
		qb.applyWebsearchOnly(searchQuery, tableAlias, vectorCol)
		return
	}

	validPrefixParts := make([]string, 0, len(words))
	for _, word := range words {
		trimmed := strings.Trim(word, "&|!()\"'")
		if trimmed == "" || len(trimmed) < 2 {
			continue
		}

		if isValidSearchTerm(trimmed) {
			escapedWord := strings.ReplaceAll(trimmed, "'", "''")
			validPrefixParts = append(validPrefixParts, escapedWord+":*")
		}
	}

	if len(validPrefixParts) == 0 {
		qb.applyWebsearchOnly(searchQuery, tableAlias, vectorCol)
		return
	}

	prefixQuery := strings.Join(validPrefixParts, " & ")

	qb.query = qb.query.ColumnExpr(
		fmt.Sprintf(
			"ts_rank(%s%s, websearch_to_tsquery('english', ?) || to_tsquery('english', ?)) AS rank",
			tableAlias,
			vectorCol,
		),
		searchQuery,
		prefixQuery,
	)

	qb.query = qb.query.Where(
		fmt.Sprintf(
			"%s%s @@ (websearch_to_tsquery('english', ?) || to_tsquery('english', ?))",
			tableAlias,
			vectorCol,
		),
		searchQuery,
		prefixQuery,
	)
}

func isValidSearchTerm(term string) bool {
	for _, r := range term {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

func (qb *QueryBuilder) applyWebsearchOnly(searchQuery, tableAlias, vectorCol string) {
	qb.query = qb.query.ColumnExpr(
		fmt.Sprintf(
			"ts_rank(%s%s, websearch_to_tsquery('english', ?)) AS rank",
			tableAlias,
			vectorCol,
		),
		searchQuery,
	)

	qb.query = qb.query.Where(
		fmt.Sprintf("%s%s @@ websearch_to_tsquery('english', ?)", tableAlias, vectorCol),
		searchQuery,
	)
}

func (qb *QueryBuilder) buildFieldSearchConditions(searchQuery, tableAlias string) {
	config := qb.searchConfig
	var conditions []string
	var args []any

	for _, field := range config.SearchableFields {
		if strings.Contains(field.Name, ".") {
			continue
		}

		switch field.Type {
		case domaintypes.FieldTypeText:
			searchPattern := "%" + searchQuery + "%"
			conditions = append(conditions,
				fmt.Sprintf("%s%s ILIKE ?", tableAlias, field.Name))
			args = append(args, searchPattern)

		case domaintypes.FieldTypeComposite:
			exactCondition := fmt.Sprintf("%s%s = ?", tableAlias, field.Name)
			prefixCondition := fmt.Sprintf("%s%s ILIKE ?", tableAlias, field.Name)
			conditions = append(conditions,
				fmt.Sprintf("(%s OR %s)", exactCondition, prefixCondition))
			args = append(args, searchQuery, searchQuery+"%")

		case domaintypes.FieldTypeNumber:
			if _, err := strconv.ParseFloat(searchQuery, 64); err == nil {
				conditions = append(conditions,
					fmt.Sprintf("%s%s::text = ?", tableAlias, field.Name))
				args = append(args, searchQuery)
			} else {
				conditions = append(conditions,
					fmt.Sprintf("%s%s::text ILIKE ?", tableAlias, field.Name))
				args = append(args, "%"+searchQuery+"%")
			}
		case domaintypes.FieldTypeBoolean:
			conditions = append(conditions,
				fmt.Sprintf("%s%s = ?", tableAlias, field.Name))
			args = append(args, searchQuery)

		case domaintypes.FieldTypeEnum:
			conditions = append(conditions,
				fmt.Sprintf("LOWER(%s%s::text) = LOWER(?)", tableAlias, field.Name))
			args = append(args, searchQuery)

		case domaintypes.FieldTypeDate:
			conditions = append(conditions,
				fmt.Sprintf("%s%s::text ILIKE ?", tableAlias, field.Name))
			args = append(args, "%"+searchQuery+"%")
		}
	}

	if len(conditions) > 0 {
		whereClause := fmt.Sprintf("(%s)", strings.Join(conditions, " OR "))
		qb.query = qb.query.Where(whereClause, args...)
	}
}

func (qb *QueryBuilder) applySearchOrdering(searchQuery string) {
	if qb.searchConfig == nil {
		return
	}
	config := qb.searchConfig
	if config.UseSearchVector {
		qb.query = qb.query.Order("rank DESC NULLS LAST")
		return
	}

	tableAlias := qb.tableAlias
	if tableAlias != "" {
		tableAlias += "."
	}

	for _, field := range config.SearchableFields {
		if strings.Contains(field.Name, ".") {
			continue
		}
		if field.Type == domaintypes.FieldTypeText || field.Type == domaintypes.FieldTypeComposite {
			weight := field.Weight
			if weight == domaintypes.SearchWeightBlank {
				weight = domaintypes.SearchWeightD
			}
			exactExpr := fmt.Sprintf("CASE WHEN LOWER(%s%s) = LOWER(?) THEN %d ELSE 0 END DESC",
				tableAlias, field.Name, weight.GetScore())
			qb.query = qb.query.OrderExpr(exactExpr, searchQuery)
		}
	}
}

func (qb *QueryBuilder) applySimpleSearch(
	searchQuery string,
	searchableFields []string,
) *QueryBuilder {
	if qb.searchConfig != nil && len(qb.searchConfig.SearchableFields) > 0 {
		return qb.applyConfigBasedSearch(searchQuery)
	}

	if len(searchableFields) == 0 {
		return qb
	}

	var conditions []string
	var args []any
	for _, field := range searchableFields {
		dbField := qb.getDBField(field)
		fieldRef := qb.getFieldReference(dbField)

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

func (qb *QueryBuilder) applyConfigBasedSearch(searchQuery string) *QueryBuilder {
	config := qb.searchConfig
	tableAlias := qb.tableAlias
	if tableAlias != "" {
		tableAlias += "."
	}

	var conditions []string
	var args []any

	searchPattern := "%" + searchQuery + "%"

	for _, field := range config.SearchableFields {
		if strings.Contains(field.Name, ".") {
			continue
		}

		fieldRef := tableAlias + field.Name

		switch field.Type {
		case domaintypes.FieldTypeText, domaintypes.FieldTypeComposite:
			conditions = append(conditions, fmt.Sprintf("%s ILIKE ?", fieldRef))
			args = append(args, searchPattern)

		case domaintypes.FieldTypeEnum:
			conditions = append(conditions, fmt.Sprintf("%s::text ILIKE ?", fieldRef))
			args = append(args, searchPattern)

		case domaintypes.FieldTypeNumber:
			if _, err := strconv.ParseFloat(searchQuery, 64); err == nil {
				conditions = append(conditions, fmt.Sprintf("%s::text = ?", fieldRef))
				args = append(args, searchQuery)
			}

		case domaintypes.FieldTypeBoolean, domaintypes.FieldTypeDate:
			continue
		}
	}

	if len(conditions) > 0 {
		whereClause := fmt.Sprintf("(%s)", strings.Join(conditions, " OR "))
		qb.query = qb.query.Where(whereClause, args...)
	}

	return qb
}
