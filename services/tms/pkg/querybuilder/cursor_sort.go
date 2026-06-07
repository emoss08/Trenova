package querybuilder

import (
	"fmt"
	"math"
	"strings"

	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
)

const (
	maxExactJSONInteger = 9007199254740991.0
	minExactJSONInteger = -9007199254740991.0
)

type CursorSortTerm struct {
	Field       string
	Direction   dbtype.SortDirection
	SQLField    string
	Alias       string
	NonNullable bool
	Integer     bool
}

type CursorSortPlan struct {
	Terms []CursorSortTerm
}

func (p CursorSortPlan) Shape() []pagination.CursorSortField {
	shape := make([]pagination.CursorSortField, 0, len(p.Terms))
	for _, term := range p.Terms {
		shape = append(shape, pagination.CursorSortField{
			Field:     term.Field,
			Direction: string(term.Direction),
		})
	}
	return shape
}

func (p CursorSortPlan) Columns() []pagination.CursorValueColumn {
	columns := make([]pagination.CursorValueColumn, 0, len(p.Terms))
	for _, term := range p.Terms {
		columns = append(columns, pagination.CursorValueColumn{
			SQLExpression: term.SQLField,
			Alias:         term.Alias,
		})
	}
	return columns
}

func ApplyFiltersWithoutSort[T domaintypes.PostgresSearchable](
	query *bun.SelectQuery,
	tableAlias string,
	filter *pagination.QueryOptions,
	entity T,
) *bun.SelectQuery {
	filterCopy := *filter
	filterCopy.Sort = nil
	filterCopy.Cursor = pagination.CursorInfo{}
	filterCopy.CursorSort = nil
	filterCopy.CursorColumns = nil
	filterCopy.CursorError = nil
	filterCopy.UseCursor = false

	return ApplyFilters(query, tableAlias, &filterCopy, entity)
}

func ApplyCursorFilters[T domaintypes.PostgresSearchable](
	query *bun.SelectQuery,
	tableAlias string,
	filter *pagination.QueryOptions,
	cursor pagination.CursorInfo,
	entity T,
) (*bun.SelectQuery, error) {
	fieldConfig := GetFieldConfiguration(entity)
	qb := NewWithPostgresSearch(query, tableAlias, fieldConfig, entity)
	qb.WithTraversalSupport(true)
	qb.ApplyTenantFilters(filter.TenantInfo)

	if len(filter.FieldFilters) > 0 {
		qb.ApplyFilters(filter.FieldFilters)
	}
	if len(filter.FilterGroups) > 0 {
		qb.ApplyFilterGroups(filter.FilterGroups)
	}
	if len(filter.GeoFilters) > 0 {
		qb.ApplyGeoFilters(filter.GeoFilters)
	}
	if len(filter.AggregateFilters) > 0 {
		qb.ApplyAggregateFilters(filter.AggregateFilters)
	}
	if filter.Query != "" {
		searchFields := ExtractSearchFields(fieldConfig)
		qb.ApplyTextSearchFilter(filter.Query, searchFields)
	}

	plan, err := qb.applyCursorSort(filter, cursor)
	if err != nil {
		return qb.GetQuery(), err
	}
	filter.CursorSort = plan.Shape()
	filter.CursorColumns = plan.Columns()

	return qb.GetQuery(), nil
}

func ApplyCursorSort[T domaintypes.PostgresSearchable](
	query *bun.SelectQuery,
	tableAlias string,
	filter *pagination.QueryOptions,
	cursor pagination.CursorInfo,
	entity T,
) error {
	fieldConfig := GetFieldConfiguration(entity)
	qb := NewWithPostgresSearch(query, tableAlias, fieldConfig, entity)
	qb.WithTraversalSupport(true)

	plan, err := qb.applyCursorSort(filter, cursor)
	if err != nil {
		return err
	}
	filter.CursorSort = plan.Shape()
	filter.CursorColumns = plan.Columns()

	return nil
}

func (qb *QueryBuilder) applyCursorSort(
	filter *pagination.QueryOptions,
	cursor pagination.CursorInfo,
) (*CursorSortPlan, error) {
	plan, err := qb.buildCursorSortPlan(normalizeCursorSort(filter.Sort))
	if err != nil {
		return nil, err
	}

	shape := plan.Shape()
	if cursor.After != "" {
		if err = pagination.ValidateCursorSort(cursor.Cursor, shape); err != nil {
			return nil, errortypes.NewValidationError(
				"after",
				errortypes.ErrInvalid,
				"Cursor sort does not match request sort",
			)
		}
		qb.applyCursorPredicate(plan, cursor.Cursor.Values)
	}

	qb.applyCursorOrder(plan)

	return plan, nil
}

func normalizeCursorSort(sorts []domaintypes.SortField) []domaintypes.SortField {
	normalized := make([]domaintypes.SortField, 0, min(len(sorts)+1, MaxSortFields))
	seen := make(map[string]bool, len(sorts)+1)
	nonIDCount := 0

	if len(sorts) == 0 {
		normalized = append(normalized, domaintypes.SortField{
			Field:     "createdAt",
			Direction: dbtype.SortDirectionDesc,
		})
		seen["createdAt"] = true
		nonIDCount++
	}

	for _, sort := range sorts {
		if sort.Field == "" || seen[sort.Field] {
			continue
		}
		if sort.Field != "id" && nonIDCount >= MaxSortFields-1 {
			continue
		}
		normalized = append(normalized, sort)
		seen[sort.Field] = true
		if sort.Field != "id" {
			nonIDCount++
		}
	}

	if !seen["id"] {
		direction := dbtype.SortDirectionDesc
		if len(normalized) > 0 {
			direction = normalized[len(normalized)-1].Direction
		}
		normalized = append(normalized, domaintypes.SortField{
			Field:     "id",
			Direction: direction,
		})
	}

	return normalized
}

func (qb *QueryBuilder) buildCursorSortPlan(
	sorts []domaintypes.SortField,
) (*CursorSortPlan, error) {
	terms := make([]CursorSortTerm, 0, len(sorts))
	for i, sort := range sorts {
		if !isValidSortDirection(sort.Direction) {
			return nil, errortypes.NewValidationError(
				"sort.direction",
				errortypes.ErrInvalid,
				"Sort direction is not supported",
			)
		}

		fieldForCheck := sort.Field
		if strings.Contains(sort.Field, ".") {
			fieldForCheck = normalizeFieldForCheck(sort.Field)
		}
		if !qb.fieldConfig.SortableFields[fieldForCheck] {
			return nil, errortypes.NewValidationError(
				"sort.field",
				errortypes.ErrInvalid,
				"Sort field is not supported",
			)
		}

		sqlField := qb.cursorSortSQLField(sort.Field)
		if sqlField == "" {
			return nil, errortypes.NewValidationError(
				"sort.field",
				errortypes.ErrInvalid,
				"Sort field is not supported",
			)
		}

		terms = append(terms, CursorSortTerm{
			Field:       sort.Field,
			Direction:   sort.Direction,
			SQLField:    sqlField,
			Alias:       fmt.Sprintf("__cursor_value_%d", i),
			NonNullable: qb.cursorSortFieldNonNullable(sort.Field),
			Integer:     qb.cursorSortFieldInteger(sort.Field),
		})
	}

	return &CursorSortPlan{Terms: terms}, nil
}

func (qb *QueryBuilder) cursorSortSQLField(field string) string {
	if qb.traversalEnabled && strings.Contains(field, ".") && qb.entity != nil &&
		qb.searchConfig != nil {
		sqlField, joinDefs := qb.resolveNestedField(field)
		if sqlField == "" {
			return ""
		}
		for _, joinDef := range joinDefs {
			joinKey := fmt.Sprintf("%s_%s", joinDef.Table, joinDef.Alias)
			if !qb.appliedJoins[joinKey] {
				qb.applyNestedFieldJoins([]domaintypes.JoinStep{joinDef})
				qb.appliedJoins[joinKey] = true
			}
		}
		return sqlField
	}

	return qb.getFieldReference(qb.getDBField(field))
}

func (qb *QueryBuilder) cursorSortFieldNonNullable(field string) bool {
	if strings.Contains(field, ".") || qb.fieldConfig == nil {
		return false
	}

	return qb.fieldConfig.NonNullableFields[field]
}

func (qb *QueryBuilder) cursorSortFieldInteger(field string) bool {
	if strings.Contains(field, ".") || qb.fieldConfig == nil {
		return false
	}

	return qb.fieldConfig.IntegerFields[field]
}

func (qb *QueryBuilder) applyCursorOrder(plan *CursorSortPlan) {
	for _, term := range plan.Terms {
		order := fmt.Sprintf("%s %s", term.SQLField, strings.ToUpper(string(term.Direction)))
		if !term.NonNullable {
			order += " NULLS LAST"
		}
		qb.query = qb.query.Order(order)
	}
}

func (qb *QueryBuilder) applyCursorPredicate(plan *CursorSortPlan, values []any) {
	if len(plan.Terms) == 0 || len(plan.Terms) != len(values) {
		return
	}

	normalizedValues := normalizeCursorPredicateValues(plan.Terms, values)
	if sql, args, ok := cursorTuplePredicate(plan.Terms, normalizedValues); ok {
		qb.query = qb.query.Where(sql, args...)
		return
	}

	qb.query = qb.query.WhereGroup(" AND ", func(cq *bun.SelectQuery) *bun.SelectQuery {
		for i := range plan.Terms {
			sql, args := cursorPredicateBranch(plan.Terms, normalizedValues, i)
			cq = cq.WhereOr(sql, args...)
		}
		return cq
	})
}

func normalizeCursorPredicateValues(terms []CursorSortTerm, values []any) []any {
	normalized := make([]any, len(values))
	for i, value := range values {
		normalized[i] = normalizeCursorPredicateValue(terms[i], value)
	}

	return normalized
}

func normalizeCursorPredicateValue(term CursorSortTerm, value any) any {
	if !term.Integer {
		return value
	}

	switch typed := value.(type) {
	case float64:
		if math.Trunc(typed) == typed && typed >= minExactJSONInteger && typed <= maxExactJSONInteger {
			return int64(typed)
		}
	case float32:
		value64 := float64(typed)
		if math.Trunc(value64) == value64 &&
			value64 >= minExactJSONInteger &&
			value64 <= maxExactJSONInteger {
			return int64(value64)
		}
	}

	return value
}

func cursorTuplePredicate(
	terms []CursorSortTerm,
	values []any,
) (string, []any, bool) {
	if len(terms) == 0 || len(terms) != len(values) {
		return "", nil, false
	}

	direction := terms[0].Direction
	if direction != dbtype.SortDirectionAsc && direction != dbtype.SortDirectionDesc {
		return "", nil, false
	}

	fields := make([]string, 0, len(terms))
	placeholders := make([]string, 0, len(values))
	args := make([]any, 0, len(values))
	for i, term := range terms {
		if !term.NonNullable || term.Direction != direction || values[i] == nil {
			return "", nil, false
		}
		fields = append(fields, term.SQLField)
		placeholders = append(placeholders, "?")
		args = append(args, values[i])
	}

	operator := ">"
	if direction == dbtype.SortDirectionDesc {
		operator = "<"
	}

	return fmt.Sprintf(
		"(%s) %s (%s)",
		strings.Join(fields, ", "),
		operator,
		strings.Join(placeholders, ", "),
	), args, true
}

func cursorPredicateBranch(
	terms []CursorSortTerm,
	values []any,
	index int,
) (string, []any) {
	parts := make([]string, 0, index+1)
	args := make([]any, 0, index+1)

	for i := range index {
		parts = append(parts, terms[i].SQLField+" IS NOT DISTINCT FROM ?")
		args = append(args, values[i])
	}

	comparison, comparisonArgs := cursorComparison(terms[index], values[index])
	parts = append(parts, comparison)
	args = append(args, comparisonArgs...)

	return "(" + strings.Join(parts, " AND ") + ")", args
}

func cursorComparison(term CursorSortTerm, value any) (string, []any) {
	if value == nil {
		return "FALSE", nil
	}

	operator := ">"
	if term.Direction == dbtype.SortDirectionDesc {
		operator = "<"
	}

	return fmt.Sprintf("(%s %s ? OR %s IS NULL)", term.SQLField, operator, term.SQLField),
		[]any{value}
}
