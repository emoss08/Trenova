package memtable

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/stringutils"
)

var (
	errUnsupportedOperator = errors.New("unsupported operator")
	errInvalidValue        = errors.New("invalid value")
	errUnknownValue        = errors.New("unknown value")
)

func normalizedValues(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, stringutils.NormalizeIdentifier(value))
	}

	return out
}

func textPredicate[T any](
	field *Field[T],
	filter *domaintypes.FieldFilter,
) (predicate[T], error) {
	if field.Text == nil {
		return nil, errUnsupportedOperator
	}
	read := func(item *T) string { return strings.ToLower(field.Text(item)) }

	switch filter.Operator {
	case dbtype.OpIsNull:
		return func(item *T) bool { return field.Text(item) == "" }, nil
	case dbtype.OpIsNotNull:
		return func(item *T) bool { return field.Text(item) != "" }, nil
	case dbtype.OpIn, dbtype.OpNotIn:
		values, err := stringList(filter.Value)
		if err != nil {
			return nil, err
		}
		for idx := range values {
			values[idx] = strings.ToLower(values[idx])
		}
		want := filter.Operator == dbtype.OpIn

		return func(item *T) bool { return slices.Contains(values, read(item)) == want }, nil
	}

	value, err := stringValue(filter.Value)
	if err != nil {
		return nil, err
	}
	value = strings.ToLower(value)

	switch filter.Operator {
	case dbtype.OpEqual:
		return func(item *T) bool { return read(item) == value }, nil
	case dbtype.OpNotEqual:
		return func(item *T) bool { return read(item) != value }, nil
	case dbtype.OpContains, dbtype.OpLike, dbtype.OpILike:
		return func(item *T) bool { return strings.Contains(read(item), value) }, nil
	case dbtype.OpStartsWith:
		return func(item *T) bool { return strings.HasPrefix(read(item), value) }, nil
	case dbtype.OpEndsWith:
		return func(item *T) bool { return strings.HasSuffix(read(item), value) }, nil
	default:
		return nil, errUnsupportedOperator
	}
}

func enumPredicate[T any](
	field *Field[T],
	filter *domaintypes.FieldFilter,
) (predicate[T], error) {
	if field.Text == nil {
		return nil, errUnsupportedOperator
	}
	read := func(item *T) string { return stringutils.NormalizeIdentifier(field.Text(item)) }

	switch filter.Operator {
	case dbtype.OpIsNull:
		return func(item *T) bool { return field.Text(item) == "" }, nil
	case dbtype.OpIsNotNull:
		return func(item *T) bool { return field.Text(item) != "" }, nil
	case dbtype.OpEqual, dbtype.OpNotEqual:
		value, err := enumValue(field, filter.Value)
		if err != nil {
			return nil, err
		}
		want := filter.Operator == dbtype.OpEqual

		return func(item *T) bool { return (read(item) == value) == want }, nil
	case dbtype.OpIn, dbtype.OpNotIn:
		values, err := enumValues(field, filter.Value)
		if err != nil {
			return nil, err
		}
		want := filter.Operator == dbtype.OpIn

		return func(item *T) bool { return slices.Contains(values, read(item)) == want }, nil
	default:
		return nil, errUnsupportedOperator
	}
}

func setPredicate[T any](
	field *Field[T],
	filter *domaintypes.FieldFilter,
) (predicate[T], error) {
	if field.Set == nil {
		return nil, errUnsupportedOperator
	}
	has := func(item *T, values []string) bool {
		for _, member := range field.Set(item) {
			if slices.Contains(values, stringutils.NormalizeIdentifier(member)) {
				return true
			}
		}

		return false
	}

	switch filter.Operator {
	case dbtype.OpIsNull:
		return func(item *T) bool { return len(field.Set(item)) == 0 }, nil
	case dbtype.OpIsNotNull:
		return func(item *T) bool { return len(field.Set(item)) > 0 }, nil
	case dbtype.OpEqual, dbtype.OpContains, dbtype.OpNotEqual:
		value, err := enumValue(field, filter.Value)
		if err != nil {
			return nil, err
		}
		values := []string{value}
		want := filter.Operator != dbtype.OpNotEqual

		return func(item *T) bool { return has(item, values) == want }, nil
	case dbtype.OpIn, dbtype.OpNotIn:
		values, err := enumValues(field, filter.Value)
		if err != nil {
			return nil, err
		}
		want := filter.Operator == dbtype.OpIn

		return func(item *T) bool { return has(item, values) == want }, nil
	default:
		return nil, errUnsupportedOperator
	}
}

func boolPredicate[T any](
	field *Field[T],
	filter *domaintypes.FieldFilter,
) (predicate[T], error) {
	if field.Bool == nil {
		return nil, errUnsupportedOperator
	}

	value, err := boolValue(filter.Value)
	if err != nil {
		return nil, err
	}

	switch filter.Operator {
	case dbtype.OpEqual:
		return func(item *T) bool { return field.Bool(item) == value }, nil
	case dbtype.OpNotEqual:
		return func(item *T) bool { return field.Bool(item) != value }, nil
	default:
		return nil, errUnsupportedOperator
	}
}

func numberPredicate[T any](
	field *Field[T],
	filter *domaintypes.FieldFilter,
) (predicate[T], error) {
	if field.Number == nil {
		return nil, errUnsupportedOperator
	}

	switch filter.Operator {
	case dbtype.OpIsNull:
		return func(item *T) bool {
			_, ok := field.Number(item)
			return !ok
		}, nil
	case dbtype.OpIsNotNull:
		return func(item *T) bool {
			_, ok := field.Number(item)
			return ok
		}, nil
	}

	value, err := numberValue(filter.Value)
	if err != nil {
		return nil, err
	}

	var test func(float64) bool
	switch filter.Operator {
	case dbtype.OpEqual:
		test = func(n float64) bool { return n == value }
	case dbtype.OpNotEqual:
		test = func(n float64) bool { return n != value }
	case dbtype.OpGreaterThan:
		test = func(n float64) bool { return n > value }
	case dbtype.OpGreaterThanOrEqual:
		test = func(n float64) bool { return n >= value }
	case dbtype.OpLessThan:
		test = func(n float64) bool { return n < value }
	case dbtype.OpLessThanOrEqual:
		test = func(n float64) bool { return n <= value }
	default:
		return nil, errUnsupportedOperator
	}

	return func(item *T) bool {
		n, ok := field.Number(item)
		return ok && test(n)
	}, nil
}

func enumValue[T any](field *Field[T], raw any) (string, error) {
	value, err := stringValue(raw)
	if err != nil {
		return "", err
	}

	normalized := stringutils.NormalizeIdentifier(value)
	if normalized == "" {
		return "", errInvalidValue
	}
	if len(field.Values) > 0 && !slices.Contains(field.Values, normalized) {
		return "", errUnknownValue
	}

	return normalized, nil
}

func enumValues[T any](field *Field[T], raw any) ([]string, error) {
	values, err := stringList(raw)
	if err != nil {
		return nil, err
	}

	out := make([]string, 0, len(values))
	for _, value := range values {
		normalized, valueErr := enumValue(field, value)
		if valueErr != nil {
			return nil, valueErr
		}
		out = append(out, normalized)
	}

	return out, nil
}

func stringValue(raw any) (string, error) {
	switch value := raw.(type) {
	case string:
		return strings.TrimSpace(value), nil
	case bool:
		return strconv.FormatBool(value), nil
	case int:
		return strconv.Itoa(value), nil
	case int64:
		return strconv.FormatInt(value, 10), nil
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64), nil
	case fmt.Stringer:
		return strings.TrimSpace(value.String()), nil
	default:
		return "", errInvalidValue
	}
}

func stringList(raw any) ([]string, error) {
	switch value := raw.(type) {
	case []string:
		if len(value) == 0 {
			return nil, errInvalidValue
		}

		return slices.Clone(value), nil
	case []int64:
		if len(value) == 0 {
			return nil, errInvalidValue
		}

		out := make([]string, 0, len(value))
		for _, item := range value {
			out = append(out, strconv.FormatInt(item, 10))
		}

		return out, nil
	case []any:
		if len(value) == 0 {
			return nil, errInvalidValue
		}

		out := make([]string, 0, len(value))
		for _, item := range value {
			text, err := stringValue(item)
			if err != nil {
				return nil, err
			}
			out = append(out, text)
		}

		return out, nil
	default:
		text, err := stringValue(raw)
		if err != nil || text == "" {
			return nil, errInvalidValue
		}

		return []string{text}, nil
	}
}

func boolValue(raw any) (bool, error) {
	switch value := raw.(type) {
	case bool:
		return value, nil
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(value))
		if err != nil {
			return false, errInvalidValue
		}

		return parsed, nil
	default:
		return false, errInvalidValue
	}
}

func numberValue(raw any) (float64, error) {
	switch value := raw.(type) {
	case float64:
		return value, nil
	case int:
		return float64(value), nil
	case int64:
		return float64(value), nil
	case string:
		return parseNumber(value)
	case fmt.Stringer:
		return parseNumber(value.String())
	default:
		return 0, errInvalidValue
	}
}

func parseNumber(raw string) (float64, error) {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return 0, errInvalidValue
	}

	return parsed, nil
}
