package tablefilters

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/samber/lo"
	"github.com/uptrace/bun"
)

func BuildQuery(q *bun.SelectQuery, queryStr string, opts QueryOptions) (*bun.SelectQuery, error) {
	var params QueryParams
	if err := sonic.Unmarshal([]byte(queryStr), &params); err != nil {
		return nil, fmt.Errorf("invalid query format: %w", err)
	}

	// Apply filters
	if len(params.Filters) > 0 {
		conditions, args := buildFilterConditions(params.Filters, strings.ToUpper(string(params.JoinOperator)), opts)
		if conditions != "" {
			q = q.Where(conditions, args...)
		}
	}

	return q, nil
}

func buildFilterConditions(filters []*Filter, joinOperator string, opts QueryOptions) (string, []any) {
	if len(filters) == 0 {
		return "", nil
	}

	var conditions []string
	var args []any

	for _, filter := range filters {
		condition, filterArgs := buildFilterCondition(filter, opts)
		if condition != "" {
			conditions = append(conditions, condition)
			args = append(args, filterArgs...)
		}
	}

	if len(conditions) == 0 {
		return "", nil
	}

	operator := " AND "
	if strings.EqualFold(joinOperator, "or") {
		operator = " OR "
	}

	return fmt.Sprintf("(%s)", strings.Join(conditions, operator)), args
}

func resolveFieldPath(field string, mappings []RelationMapping) (string, string) {
	parts := strings.Split(field, ".")
	if len(parts) == 1 {
		return "", field
	}

	for i := range mappings {
		if mappings[i].From == parts[0] {
			return mappings[i].To, lo.SnakeCase(strings.Join(parts[1:], "_"))
		}
	}

	return "", field
}

func buildFilterCondition(filter *Filter, opts QueryOptions) (string, []any) {
	// Resolve the actual table and field from the relationship mappings
	tableAlias, fieldName := resolveFieldPath(filter.Field, opts.RelationMappings)
	if tableAlias == "" {
		tableAlias = opts.TableAlias
		fieldName = lo.SnakeCase(filter.Field)
	}

	// Build the fully qualified field name with single quotes
	field := fmt.Sprintf(`"%q"."%q"`, tableAlias, fieldName)

	// Check if this is an enum field
	isEnum := isEnumField(tableAlias, fieldName, opts.EnumFields)

	switch filter.Type {
	case "text":
		return buildTextFilter(field, filter, isEnum)
	case "number":
		return buildNumberFilter(field, filter)
	case "date":
		return buildDateFilter(field, filter)
	case "multi-select":
		return buildMultiSelectFilter(field, filter)
	case "select":
		return buildSelectFilter(field, filter, isEnum)
	case "boolean":
		return buildBooleanFilter(field, filter)
	default:
		return "", nil
	}
}

func buildTextFilter(field string, filter *Filter, isEnum bool) (string, []any) {
	if isEnum {
		field += "::text"
	}

	switch filter.Operator {
	case "iLike":
		return fmt.Sprintf("%s ILIKE ?", field), []any{"%" + filter.Value.(string) + "%"}
	case "notILike":
		return fmt.Sprintf("%s NOT ILIKE ?", field), []any{"%" + filter.Value.(string) + "%"}
	case "eq":
		return fmt.Sprintf("%s = ?", field), []any{filter.Value}
	case "ne":
		return fmt.Sprintf("%s != ?", field), []any{filter.Value}
	case "isEmpty":
		return fmt.Sprintf("(%s IS NULL OR %s = '')", field, field), nil
	case "isNotEmpty":
		return fmt.Sprintf("(%s IS NOT NULL AND %s != '')", field, field), nil
	default:
		return "", nil
	}
}

func buildNumberFilter(field string, filter *Filter) (string, []any) {
	switch filter.Operator {
	case "eq":
		return fmt.Sprintf("%s = ?", field), []any{filter.Value}
	case "ne":
		return fmt.Sprintf("%s != ?", field), []any{filter.Value}
	case "lt":
		return fmt.Sprintf("%s < ?", field), []any{filter.Value}
	case "lte":
		return fmt.Sprintf("%s <= ?", field), []any{filter.Value}
	case "gt":
		return fmt.Sprintf("%s > ?", field), []any{filter.Value}
	case "gte":
		return fmt.Sprintf("%s >= ?", field), []any{filter.Value}
	case "isEmpty":
		return fmt.Sprintf("%s IS NULL", field), nil
	case "isNotEmpty":
		return fmt.Sprintf("%s IS NOT NULL", field), nil
	case "isBetween":
		return fmt.Sprintf("(%s >= ? AND %s <= ?)", field, field),
			[]any{filter.Value, filter.ValueTo}
	default:
		return "", nil
	}
}

func buildDateFilter(field string, filter *Filter) (string, []any) {
	// Convert ISO8601 string to Unix timestamp
	parseAndConvertToUnix := func(dateStr any) (int64, error) {
		if str, ok := dateStr.(string); ok {
			t, err := time.Parse(time.RFC3339, str)
			if err != nil {
				return 0, err
			}
			return t.Unix(), nil
		}
		return 0, ErrInvalidDateRange
	}

	switch filter.Operator {
	case "eq":
		if ts, err := parseAndConvertToUnix(filter.Value); err == nil {
			return fmt.Sprintf(`%s = ?`, field), []any{ts}
		}
	case "ne":
		if ts, err := parseAndConvertToUnix(filter.Value); err == nil {
			return fmt.Sprintf(`%s != ?`, field), []any{ts}
		}
	case "lt":
		if ts, err := parseAndConvertToUnix(filter.Value); err == nil {
			return fmt.Sprintf(`%s < ?`, field), []any{ts}
		}
	case "lte":
		if ts, err := parseAndConvertToUnix(filter.Value); err == nil {
			return fmt.Sprintf(`%s <= ?`, field), []any{ts}
		}
	case "gt":
		if ts, err := parseAndConvertToUnix(filter.Value); err == nil {
			return fmt.Sprintf(`%s > ?`, field), []any{ts}
		}
	case "gte":
		if ts, err := parseAndConvertToUnix(filter.Value); err == nil {
			return fmt.Sprintf(`%s >= ?`, field), []any{ts}
		}
	case "isEmpty":
		return fmt.Sprintf(`%s IS NULL`, field), nil
	case "isNotEmpty":
		return fmt.Sprintf(`%s IS NOT NULL`, field), nil
	case "isBetween":
		startTs, err1 := parseAndConvertToUnix(filter.Value)
		endTs, err2 := parseAndConvertToUnix(filter.ValueTo)
		if err1 == nil && err2 == nil {
			return fmt.Sprintf(`(%s >= ? AND %s <= ?)`, field, field),
				[]any{startTs, endTs}
		}
	}

	return "", nil
}

func buildMultiSelectFilter(field string, filter *Filter) (string, []any) {
	switch filter.Operator {
	case "in":
		values, ok := filter.Value.([]any)
		if !ok {
			return "", nil
		}
		placeholders := make([]string, len(values))
		args := make([]any, len(values))
		for i, v := range values {
			placeholders[i] = "?"
			args[i] = v
		}
		// Removed double quoting as field is already properly quoted
		return fmt.Sprintf(`%s IN (%s)`, field, strings.Join(placeholders, ", ")), args

	case "notIn":
		values, ok := filter.Value.([]any)
		if !ok {
			return "", nil
		}
		placeholders := make([]string, len(values))
		args := make([]any, len(values))
		for i, v := range values {
			placeholders[i] = "?"
			args[i] = v
		}
		return fmt.Sprintf(`%s NOT IN (%s)`, field, strings.Join(placeholders, ", ")), args

	case "isEmpty":
		return fmt.Sprintf(`%s IS NULL`, field), nil

	case "isNotEmpty":
		return fmt.Sprintf(`%s IS NOT NULL`, field), nil

	default:
		return "", nil
	}
}

func buildSelectFilter(field string, filter *Filter, isEnum bool) (string, []any) {
	if isEnum {
		field += "::text"
	}

	switch filter.Operator {
	case "in":
		return buildMultiSelectFilter(field, filter)
	case "eq":
		return fmt.Sprintf("%s = ?", field), []any{filter.Value}
	case "ne":
		return fmt.Sprintf("%s != ?", field), []any{filter.Value}
	case "isEmpty":
		return fmt.Sprintf("%s IS NULL", field), nil
	case "isNotEmpty":
		return fmt.Sprintf("%s IS NOT NULL", field), nil
	default:
		return "", nil
	}
}

func buildBooleanFilter(field string, filter *Filter) (string, []any) {
	switch filter.Operator {
	case "eq":
		return fmt.Sprintf("%s = ?", field), []any{filter.Value}
	case "ne":
		return fmt.Sprintf("%s != ?", field), []any{filter.Value}
	default:
		return "", nil
	}
}

// GetTotalCount retrieves the total count of records for the query
func GetTotalCount(ctx context.Context, q *bun.SelectQuery) (int, error) {
	return q.Count(ctx)
}

func isEnumField(tableAlias, fieldName string, enumFields []EnumField) bool {
	for _, ef := range enumFields {
		if ef.TableAlias == tableAlias {
			fieldName = strings.TrimPrefix(fieldName, tableAlias+".")
			for _, field := range ef.Fields {
				if field == fieldName {
					return true
				}
			}
		}
	}
	return false
}
