package querybuilder

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/uptrace/bun"
)

const (
	MaxFilters          = 50
	MaxSortFields       = 10
	MaxGeoFilters       = 10
	MaxAggregateFilters = 10
)

type QueryBuilder struct {
	query            *bun.SelectQuery
	tableAlias       string
	fieldConfig      *domaintypes.FieldConfiguration
	searchConfig     *domaintypes.PostgresSearchConfig
	appliedJoins     map[string]bool
	traversalEnabled bool
	entity           any
	invalidFields    []string
}

func New(
	query *bun.SelectQuery,
	tableAlias string,
	fieldConfig *domaintypes.FieldConfiguration,
) *QueryBuilder {
	return &QueryBuilder{
		query:            query,
		tableAlias:       tableAlias,
		fieldConfig:      fieldConfig,
		appliedJoins:     make(map[string]bool),
		traversalEnabled: false,
		invalidFields:    make([]string, 0),
	}
}

func NewWithPostgresSearch[T domaintypes.PostgresSearchable](
	query *bun.SelectQuery,
	tableAlias string,
	fieldConfig *domaintypes.FieldConfiguration,
	entity T,
) *QueryBuilder {
	config := entity.GetPostgresSearchConfig()

	return &QueryBuilder{
		query:            query,
		tableAlias:       tableAlias,
		fieldConfig:      fieldConfig,
		searchConfig:     &config,
		appliedJoins:     make(map[string]bool),
		traversalEnabled: false,
		entity:           entity,
		invalidFields:    make([]string, 0),
	}
}

func (qb *QueryBuilder) ApplyFilters(filters []domaintypes.FieldFilter) *QueryBuilder {
	if len(filters) > MaxFilters {
		filters = filters[:MaxFilters]
	}

	for _, filter := range filters {
		field := filter.Field

		if qb.traversalEnabled && strings.Contains(field, ".") {
			qb.applyTraversalFilter(field, filter)
		} else {
			fieldForCheck := field
			if strings.Contains(field, ".") {
				parts := strings.Split(field, ".")
				parts[0] = stringutils.ConvertCamelToSnake(parts[0])
				fieldForCheck = strings.Join(parts, ".")
			}

			if !qb.fieldConfig.FilterableFields[fieldForCheck] {
				qb.invalidFields = append(qb.invalidFields, field)
				continue
			}
			dbField := qb.getDBField(field)
			isEnum := qb.isEnumField(field)
			qb.applyFilter(dbField, filter.Operator, filter.Value, isEnum)
		}
	}
	return qb
}

func (qb *QueryBuilder) ApplySort(sorts []domaintypes.SortField) *QueryBuilder {
	if len(sorts) > MaxSortFields {
		sorts = sorts[:MaxSortFields]
	}

	for _, sort := range sorts {
		if !isValidSortDirection(sort.Direction) {
			qb.invalidFields = append(qb.invalidFields, sort.Field)
			continue
		}

		fieldForCheck := sort.Field
		if strings.Contains(sort.Field, ".") {
			fieldForCheck = normalizeFieldForCheck(sort.Field)
		}

		if !qb.fieldConfig.SortableFields[fieldForCheck] {
			qb.invalidFields = append(qb.invalidFields, sort.Field)
			continue
		}

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
func (qb *QueryBuilder) applyTraversalSort(sort domaintypes.SortField) {
	sqlField, joinDefs := qb.resolveNestedField(sort.Field)
	if len(joinDefs) == 0 {
		qb.applySortField(sort)
		return
	}

	// Apply joins if not already applied
	for _, joinDef := range joinDefs {
		joinKey := fmt.Sprintf("%s_%s", joinDef.Table, joinDef.Alias)
		if !qb.appliedJoins[joinKey] {
			qb.applyNestedFieldJoins([]domaintypes.JoinStep{joinDef})
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

func (qb *QueryBuilder) GetInvalidFields() []string {
	return qb.invalidFields
}

func (qb *QueryBuilder) HasInvalidFields() bool {
	return len(qb.invalidFields) > 0
}

func (qb *QueryBuilder) applySortField(sort domaintypes.SortField) {
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

func isValidSortDirection(direction dbtype.SortDirection) bool {
	switch direction {
	case dbtype.SortDirectionAsc, dbtype.SortDirectionDesc:
		return true
	default:
		return false
	}
}

func normalizeFieldForCheck(field string) string {
	if !strings.Contains(field, ".") {
		return field
	}

	parts := strings.Split(field, ".")
	parts[0] = stringutils.ConvertCamelToSnake(parts[0])
	return strings.Join(parts, ".")
}

func (qb *QueryBuilder) getDBField(apiField string) string {
	// Normalize field name for lookup
	normalizedField := apiField
	if strings.Contains(apiField, ".") {
		parts := strings.Split(apiField, ".")
		parts[0] = stringutils.ConvertCamelToSnake(parts[0])
		normalizedField = strings.Join(parts, ".")
	}

	if nestedDef, exists := qb.fieldConfig.NestedFields[normalizedField]; exists {
		qb.applyNestedFieldJoins(nestedDef.RequiredJoins)
		return nestedDef.DatabaseField
	}

	if dbField, exists := qb.fieldConfig.FieldMap[apiField]; exists {
		return dbField
	}

	if strings.Contains(apiField, ".") {
		return normalizedField
	}

	return apiField
}

func (qb *QueryBuilder) applyNestedFieldJoins(joins []domaintypes.JoinStep) {
	for _, join := range joins {
		joinKey := fmt.Sprintf("%s_%s", join.Table, join.Alias)
		if qb.appliedJoins[joinKey] {
			continue
		}

		var joinType string
		switch join.JoinType {
		case dbtype.JoinTypeLeft:
			joinType = "LEFT"
		case dbtype.JoinTypeRight:
			joinType = "RIGHT"
		case dbtype.JoinTypeInner:
			joinType = "INNER"
		default:
			joinType = string(join.JoinType)
		}

		condition := strings.ReplaceAll(join.Condition, "main", qb.tableAlias)
		joinClause := fmt.Sprintf("%s JOIN %s AS %s ON %s",
			joinType, join.Table, join.Alias, condition)
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
	operator dbtype.Operator,
	value any,
	isEnum bool,
) {
	fieldRef := qb.getFieldReference(dbField)

	if isEnum &&
		(qb.isStringOperation(operator) || operator == dbtype.OpIn || operator == dbtype.OpNotIn) {
		fieldRef = fmt.Sprintf("%s::text", fieldRef)
	}

	qb.applyStatement(operator, fieldRef, value)
}

func (qb *QueryBuilder) applyStatement( //nolint:cyclop,funlen // operator handling requires many cases
	operator dbtype.Operator,
	fieldRef string,
	value any,
) {
	switch operator { //nolint:exhaustive // count operators handled in ApplyAggregateFilters
	case dbtype.OpEqual:
		if qb.isEmptyEnumValue(value) {
			return
		}
		qb.query = qb.query.Where(fmt.Sprintf("%s = ?", fieldRef), value)
	case dbtype.OpNotEqual:
		if qb.isEmptyEnumValue(value) {
			return
		}
		qb.query = qb.query.Where(fmt.Sprintf("%s != ?", fieldRef), value)
	case dbtype.OpGreaterThan:
		qb.query = qb.query.Where(fmt.Sprintf("%s > ?", fieldRef), value)
	case dbtype.OpGreaterThanOrEqual:
		qb.query = qb.query.Where(fmt.Sprintf("%s >= ?", fieldRef), value)
	case dbtype.OpLessThan:
		qb.query = qb.query.Where(fmt.Sprintf("%s < ?", fieldRef), value)
	case dbtype.OpLessThanOrEqual:
		qb.query = qb.query.Where(fmt.Sprintf("%s <= ?", fieldRef), value)
	case dbtype.OpContains:
		qb.query = qb.query.Where(
			fmt.Sprintf("%s ILIKE ?", fieldRef),
			fmt.Sprintf("%%%v%%", value),
		)
	case dbtype.OpStartsWith:
		qb.query = qb.query.Where(fmt.Sprintf("%s ILIKE ?", fieldRef), fmt.Sprintf("%v%%", value))
	case dbtype.OpEndsWith:
		qb.query = qb.query.Where(fmt.Sprintf("%s ILIKE ?", fieldRef), fmt.Sprintf("%%%v", value))
	case dbtype.OpLike:
		qb.query = qb.query.Where(fmt.Sprintf("%s LIKE ?", fieldRef), value)
	case dbtype.OpILike:
		qb.query = qb.query.Where(fmt.Sprintf("%s ILIKE ?", fieldRef), value)
	case dbtype.OpIn:
		qb.query = qb.query.Where(fmt.Sprintf("%s IN (?)", fieldRef), bun.In(value))
	case dbtype.OpNotIn:
		qb.query = qb.query.Where(fmt.Sprintf("%s NOT IN (?)", fieldRef), bun.In(value))
	case dbtype.OpIsNull:
		qb.query = qb.query.Where(fmt.Sprintf("%s IS NULL", fieldRef))
	case dbtype.OpIsNotNull:
		qb.query = qb.query.Where(fmt.Sprintf("%s IS NOT NULL", fieldRef))
	case dbtype.OpDateRange:
		qb.applyDateRangeFilter(fieldRef, value)
	case dbtype.OpLastNDays:
		days := extractDays(value)
		startTime := time.Now().AddDate(0, 0, -days).Unix()
		qb.query = qb.query.Where(fmt.Sprintf("%s >= ?", fieldRef), startTime)
	case dbtype.OpNextNDays:
		days := extractDays(value)
		endTime := time.Now().AddDate(0, 0, days).Unix()
		qb.query = qb.query.Where(fmt.Sprintf("%s <= ?", fieldRef), endTime)
	case dbtype.OpToday:
		start, end := getDayBounds(time.Now())
		cond := fmt.Sprintf("%s >= ? AND %s <= ?", fieldRef, fieldRef)
		qb.query = qb.query.Where(cond, start, end)
	case dbtype.OpYesterday:
		start, end := getDayBounds(time.Now().AddDate(0, 0, -1))
		cond := fmt.Sprintf("%s >= ? AND %s <= ?", fieldRef, fieldRef)
		qb.query = qb.query.Where(cond, start, end)
	case dbtype.OpTomorrow:
		start, end := getDayBounds(time.Now().AddDate(0, 0, 1))
		cond := fmt.Sprintf("%s >= ? AND %s <= ?", fieldRef, fieldRef)
		qb.query = qb.query.Where(cond, start, end)
	}
}

func (qb *QueryBuilder) isStringOperation(operator dbtype.Operator) bool {
	switch operator { //nolint:exhaustive // We only support the operators we need
	case dbtype.OpContains,
		dbtype.OpEqual,
		dbtype.OpStartsWith,
		dbtype.OpEndsWith,
		dbtype.OpLike,
		dbtype.OpILike:
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

	qb.applyFromDateFilter(fieldRef, dateRange)
	qb.applyToDateFilter(fieldRef, dateRange)
}

func (qb *QueryBuilder) applyFromDateFilter(fieldRef string, dateRange map[string]any) {
	fromVal, hasFrom := dateRange["from"]
	if !hasFrom {
		return
	}

	from, ok := fromVal.(float64)
	if ok {
		qb.query = qb.query.Where(fmt.Sprintf("%s >= ?", fieldRef), from)
	}
}

func (qb *QueryBuilder) applyToDateFilter(fieldRef string, dateRange map[string]any) {
	toVal, hasTo := dateRange["to"]
	if !hasTo {
		return
	}

	to, ok := toVal.(float64)
	if ok {
		endExclusive := to + 86400 // 1 day in seconds to make it inclusive
		qb.query = qb.query.Where(fmt.Sprintf("%s < ?", fieldRef), endExclusive)
	}
}

func extractDays(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case map[string]any:
		if days, ok := v["days"]; ok {
			return extractDays(days)
		}
	}
	return 0
}

func getDayBounds(t time.Time) (startUnix, endUnix int64) {
	start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	end := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
	return start.Unix(), end.Unix()
}

func (qb *QueryBuilder) applyTraversalFilter(fieldPath string, filter domaintypes.FieldFilter) {
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
	filter domaintypes.FieldFilter,
) {
	sqlField, joinDefs := qb.resolveNestedField(fieldPath)
	if len(joinDefs) == 0 {
		return
	}

	for _, joinDef := range joinDefs {
		joinKey := fmt.Sprintf("%s_%s", joinDef.Table, joinDef.Alias)
		if !qb.appliedJoins[joinKey] {
			qb.applyNestedFieldJoins([]domaintypes.JoinStep{joinDef})
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
		if strings.EqualFold(rel.Field, relName) && rel.Type == dbtype.RelationshipTypeCustom {
			return rel.IsEnum
		}
	}

	return false
}

func (qb *QueryBuilder) resolveNestedField(fieldPath string) (string, []domaintypes.JoinStep) {
	parts := strings.Split(fieldPath, ".")
	if len(parts) < 2 || qb.searchConfig == nil {
		return fieldPath, nil
	}

	rel := qb.findRelationship(parts[0])
	if rel == nil {
		return "", nil
	}

	if rel.Type == dbtype.RelationshipTypeCustom {
		return qb.resolveCustomRelationship(rel, parts)
	}

	return qb.resolveStandardRelationship(rel, parts)
}

func (qb *QueryBuilder) findRelationship(name string) *domaintypes.RelationshipDefintion {
	for i := range qb.searchConfig.Relationships {
		if strings.EqualFold(qb.searchConfig.Relationships[i].Field, name) {
			return qb.searchConfig.Relationships[i]
		}
	}
	return nil
}

func (qb *QueryBuilder) resolveCustomRelationship(
	rel *domaintypes.RelationshipDefintion,
	parts []string,
) (string, []domaintypes.JoinStep) {
	joinDefs := make([]domaintypes.JoinStep, 0, len(rel.CustomJoinPath))

	for _, step := range rel.CustomJoinPath {
		joinDef := domaintypes.JoinStep{
			Table:     step.Table,
			Alias:     step.Alias,
			Condition: step.Condition,
			JoinType:  step.JoinType,
		}
		if joinDef.JoinType == "" {
			joinDef.JoinType = dbtype.JoinTypeLeft
		}
		joinDefs = append(joinDefs, joinDef)
	}

	fieldName := parts[len(parts)-1]
	targetFieldName := rel.TargetField
	if targetFieldName == "" {
		targetFieldName = stringutils.ConvertCamelToSnake(fieldName)
	}

	finalAlias := rel.Alias
	if finalAlias == "" && len(rel.CustomJoinPath) > 0 {
		finalAlias = rel.CustomJoinPath[len(rel.CustomJoinPath)-1].Alias
	}

	if finalAlias == "" {
		return "", nil
	}

	return fmt.Sprintf("%s.%s", finalAlias, targetFieldName), joinDefs
}

func (qb *QueryBuilder) resolveStandardRelationship(
	rel *domaintypes.RelationshipDefintion,
	parts []string,
) (string, []domaintypes.JoinStep) {
	var joinDefs []domaintypes.JoinStep

	switch rel.Type { //nolint:exhaustive // RelationshipTypeCustom handled by resolveCustomRelationship
	case dbtype.RelationshipTypeBelongsTo:
		joinDefs = append(joinDefs, domaintypes.JoinStep{
			Table:    rel.TargetTable,
			Alias:    rel.Alias,
			JoinType: dbtype.JoinTypeLeft,
			Condition: fmt.Sprintf("%s.%s = %s.%s",
				qb.tableAlias, rel.ForeignKey, rel.Alias, rel.ReferenceKey),
		})

	case dbtype.RelationshipTypeHasOne, dbtype.RelationshipTypeHasMany:
		joinDefs = append(joinDefs, domaintypes.JoinStep{
			Table:    rel.TargetTable,
			Alias:    rel.Alias,
			JoinType: dbtype.JoinTypeLeft,
			Condition: fmt.Sprintf("%s.%s = %s.%s",
				rel.Alias, rel.ForeignKey, qb.tableAlias, rel.ReferenceKey),
		})

	case dbtype.RelationshipTypeManyToMany:
		joinDefs = append(joinDefs,
			domaintypes.JoinStep{
				Table:    rel.JoinTable,
				Alias:    rel.JoinTableAlias,
				JoinType: dbtype.JoinTypeLeft,
				Condition: fmt.Sprintf("%s.%s = %s.%s",
					qb.tableAlias, rel.ReferenceKey, rel.JoinTableAlias, rel.JoinTableSourceKey),
			},
			domaintypes.JoinStep{
				Table:    rel.TargetTable,
				Alias:    rel.Alias,
				JoinType: dbtype.JoinTypeLeft,
				Condition: fmt.Sprintf("%s.%s = %s.%s",
					rel.JoinTableAlias, rel.JoinTableTargetKey, rel.Alias, rel.ReferenceKey),
			},
		)

	default:
		return "", nil
	}

	sqlFieldName := stringutils.ConvertCamelToSnake(parts[len(parts)-1])
	return fmt.Sprintf("%s.%s", rel.Alias, sqlFieldName), joinDefs
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

func (qb *QueryBuilder) ApplyTenantFilters(tenantInfo pagination.TenantInfo) *QueryBuilder {
	qb.query = qb.query.Where(
		fmt.Sprintf("%s.organization_id = ?", qb.tableAlias),
		tenantInfo.OrgID,
	)
	qb.query = qb.query.Where(
		fmt.Sprintf("%s.business_unit_id = ?", qb.tableAlias),
		tenantInfo.BuID,
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

	tableAlias := qb.tableAlias
	if tableAlias != "" {
		tableAlias += "."
	}

	qb.buildSearchVectorConditions(searchQuery, tableAlias)
	qb.query = qb.query.Order("rank DESC NULLS LAST")

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

func (qb *QueryBuilder) ApplyFilterGroups(groups []domaintypes.FilterGroup) *QueryBuilder {
	if len(groups) > MaxFilters {
		groups = groups[:MaxFilters]
	}

	for _, group := range groups {
		if len(group.Filters) == 0 {
			continue
		}

		var conditions []string
		var args []any

		for _, filter := range group.Filters {
			fieldForCheck := normalizeFieldForCheck(filter.Field)

			if !qb.fieldConfig.FilterableFields[fieldForCheck] {
				qb.invalidFields = append(qb.invalidFields, filter.Field)
				continue
			}

			condition, filterArgs := qb.buildFilterCondition(filter)
			if condition != "" {
				conditions = append(conditions, condition)
				args = append(args, filterArgs...)
			}
		}

		if len(conditions) > 0 {
			orClause := "(" + strings.Join(conditions, " OR ") + ")"
			qb.query = qb.query.Where(orClause, args...)
		}
	}
	return qb
}

func (qb *QueryBuilder) buildFilterCondition(
	filter domaintypes.FieldFilter,
) (condition string, args []any) {
	dbField := qb.getDBField(filter.Field)
	fieldRef := qb.getFieldReference(dbField)
	isEnum := qb.isEnumField(filter.Field)

	isStringOp := qb.isStringOperation(filter.Operator)
	isInOp := filter.Operator == dbtype.OpIn || filter.Operator == dbtype.OpNotIn
	if isEnum && (isStringOp || isInOp) {
		fieldRef = fmt.Sprintf("%s::text", fieldRef)
	}

	return qb.buildConditionStatement(filter.Operator, fieldRef, filter.Value)
}

func (qb *QueryBuilder) buildConditionStatement( //nolint:cyclop,funlen // operator handling requires many cases
	operator dbtype.Operator,
	fieldRef string,
	value any,
) (condition string, args []any) {
	switch operator { //nolint:exhaustive // count and daterange operators handled elsewhere
	case dbtype.OpEqual:
		if qb.isEmptyEnumValue(value) {
			return "", nil
		}
		return fmt.Sprintf("%s = ?", fieldRef), []any{value}
	case dbtype.OpNotEqual:
		if qb.isEmptyEnumValue(value) {
			return "", nil
		}
		return fmt.Sprintf("%s != ?", fieldRef), []any{value}
	case dbtype.OpGreaterThan:
		return fmt.Sprintf("%s > ?", fieldRef), []any{value}
	case dbtype.OpGreaterThanOrEqual:
		return fmt.Sprintf("%s >= ?", fieldRef), []any{value}
	case dbtype.OpLessThan:
		return fmt.Sprintf("%s < ?", fieldRef), []any{value}
	case dbtype.OpLessThanOrEqual:
		return fmt.Sprintf("%s <= ?", fieldRef), []any{value}
	case dbtype.OpContains:
		return fmt.Sprintf("%s ILIKE ?", fieldRef), []any{fmt.Sprintf("%%%v%%", value)}
	case dbtype.OpStartsWith:
		return fmt.Sprintf("%s ILIKE ?", fieldRef), []any{fmt.Sprintf("%v%%", value)}
	case dbtype.OpEndsWith:
		return fmt.Sprintf("%s ILIKE ?", fieldRef), []any{fmt.Sprintf("%%%v", value)}
	case dbtype.OpLike:
		return fmt.Sprintf("%s LIKE ?", fieldRef), []any{value}
	case dbtype.OpILike:
		return fmt.Sprintf("%s ILIKE ?", fieldRef), []any{value}
	case dbtype.OpIn:
		return fmt.Sprintf("%s IN (?)", fieldRef), []any{bun.In(value)}
	case dbtype.OpNotIn:
		return fmt.Sprintf("%s NOT IN (?)", fieldRef), []any{bun.In(value)}
	case dbtype.OpIsNull:
		return fmt.Sprintf("%s IS NULL", fieldRef), nil
	case dbtype.OpIsNotNull:
		return fmt.Sprintf("%s IS NOT NULL", fieldRef), nil
	case dbtype.OpLastNDays:
		days := extractDays(value)
		startTime := time.Now().AddDate(0, 0, -days).Unix()
		return fmt.Sprintf("%s >= ?", fieldRef), []any{startTime}
	case dbtype.OpNextNDays:
		days := extractDays(value)
		endTime := time.Now().AddDate(0, 0, days).Unix()
		return fmt.Sprintf("%s <= ?", fieldRef), []any{endTime}
	case dbtype.OpToday:
		start, end := getDayBounds(time.Now())
		return fmt.Sprintf("%s >= ? AND %s <= ?", fieldRef, fieldRef), []any{start, end}
	case dbtype.OpYesterday:
		start, end := getDayBounds(time.Now().AddDate(0, 0, -1))
		return fmt.Sprintf("%s >= ? AND %s <= ?", fieldRef, fieldRef), []any{start, end}
	case dbtype.OpTomorrow:
		start, end := getDayBounds(time.Now().AddDate(0, 0, 1))
		return fmt.Sprintf("%s >= ? AND %s <= ?", fieldRef, fieldRef), []any{start, end}
	default:
		return "", nil
	}
}

func (qb *QueryBuilder) ApplyGeoFilters(geoFilters []domaintypes.GeoFilter) *QueryBuilder {
	if len(geoFilters) > MaxGeoFilters {
		geoFilters = geoFilters[:MaxGeoFilters]
	}

	for _, gf := range geoFilters {
		fieldForCheck := normalizeFieldForCheck(gf.Field)
		if !qb.fieldConfig.GeoFilterableFields[fieldForCheck] {
			qb.invalidFields = append(qb.invalidFields, gf.Field)
			continue
		}

		dbField := qb.getDBField(gf.Field)
		fieldRef := qb.getFieldReference(dbField)
		radiusMeters := gf.RadiusKm * 1000

		geoCondition := fmt.Sprintf(
			"ST_DWithin(%s, ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography, ?)",
			fieldRef,
		)
		qb.query = qb.query.Where(
			geoCondition,
			gf.Center.Longitude,
			gf.Center.Latitude,
			radiusMeters,
		)
	}
	return qb
}

func (qb *QueryBuilder) ApplyAggregateFilters(
	aggFilters []domaintypes.AggregateFilter,
) *QueryBuilder {
	if len(aggFilters) > MaxAggregateFilters {
		aggFilters = aggFilters[:MaxAggregateFilters]
	}

	for _, af := range aggFilters {
		rel := qb.findRelationship(af.Relation)
		if rel == nil {
			qb.invalidFields = append(qb.invalidFields, af.Relation)
			continue
		}

		var operator string
		switch af.Operator { //nolint:exhaustive // only count operators are valid here
		case dbtype.OpCountGt:
			operator = ">"
		case dbtype.OpCountLt:
			operator = "<"
		case dbtype.OpCountEq:
			operator = "="
		case dbtype.OpCountGte:
			operator = ">="
		case dbtype.OpCountLte:
			operator = "<="
		default:
			continue
		}

		subquery := fmt.Sprintf(
			"(SELECT COUNT(*) FROM %s WHERE %s = %s.%s) %s ?",
			rel.TargetTable,
			rel.ForeignKey,
			qb.tableAlias,
			rel.ReferenceKey,
			operator,
		)
		qb.query = qb.query.Where(subquery, af.Value)
	}
	return qb
}

func (qb *QueryBuilder) GetDistinctValues(field string) *QueryBuilder {
	fieldForCheck := normalizeFieldForCheck(field)
	if !qb.fieldConfig.FilterableFields[fieldForCheck] {
		qb.invalidFields = append(qb.invalidFields, field)
		return qb
	}

	dbField := qb.getDBField(field)
	fieldRef := qb.getFieldReference(dbField)

	qb.query = qb.query.ColumnExpr(fmt.Sprintf("DISTINCT %s", fieldRef))
	qb.query = qb.query.Order(fieldRef)

	return qb
}
