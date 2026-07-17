package compiler

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/shopspring/decimal"
)

func coerceScalar(
	fieldType reportcatalog.FieldType,
	field *reportcatalog.Field,
	value any,
) (any, error) {
	//nolint:exhaustive // json fields are rejected in the default branch
	switch fieldType {
	case reportcatalog.FieldString, reportcatalog.FieldRef:
		s, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected a string, got %T", value)
		}
		return s, nil
	case reportcatalog.FieldEnum:
		s, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected a string, got %T", value)
		}
		if field != nil && len(field.EnumValues) > 0 && !isEnumValue(field, s) {
			return nil, fmt.Errorf("%q is not a valid value for this field", s)
		}
		return s, nil
	case reportcatalog.FieldBool:
		b, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("expected a boolean, got %T", value)
		}
		return b, nil
	case reportcatalog.FieldInt, reportcatalog.FieldEpoch:
		return coerceInteger(value)
	case reportcatalog.FieldDecimal:
		return coerceDecimal(value)
	default:
		return nil, fmt.Errorf("fields of type %q cannot be compared to a value", fieldType)
	}
}

func isEnumValue(field *reportcatalog.Field, value string) bool {
	for i := range field.EnumValues {
		if field.EnumValues[i].Value == value {
			return true
		}
	}
	return false
}

// jsonNumber matches encoding/json.Number (and sonic's alias of it) without
// importing the lint-forbidden encoding/json package. gqlgen decodes the Any
// and JSON scalars with UseNumber, so client-supplied params and filter
// values arrive as json.Number rather than float64.
type jsonNumber interface {
	Int64() (int64, error)
	Float64() (float64, error)
	String() string
}

func coerceInteger(value any) (int64, error) {
	switch n := value.(type) {
	case int64:
		return n, nil
	case int:
		return int64(n), nil
	case float64:
		if n != float64(int64(n)) {
			return 0, fmt.Errorf("expected an integer, got %v", n)
		}
		return int64(n), nil
	case jsonNumber:
		parsed, err := n.Int64()
		if err != nil {
			return 0, fmt.Errorf("expected an integer, got %q", n.String())
		}
		return parsed, nil
	case string:
		parsed, err := strconv.ParseInt(n, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("expected an integer, got %q", n)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("expected an integer, got %T", value)
	}
}

func coerceDecimal(value any) (decimal.Decimal, error) {
	switch n := value.(type) {
	case float64:
		return decimal.NewFromFloat(n), nil
	case int64:
		return decimal.NewFromInt(n), nil
	case int:
		return decimal.NewFromInt(int64(n)), nil
	case jsonNumber:
		parsed, err := decimal.NewFromString(n.String())
		if err != nil {
			return decimal.Decimal{}, fmt.Errorf("expected a numeric value, got %q", n.String())
		}
		return parsed, nil
	case string:
		parsed, err := decimal.NewFromString(n)
		if err != nil {
			return decimal.Decimal{}, fmt.Errorf("expected a numeric value, got %q", n)
		}
		return parsed, nil
	default:
		return decimal.Decimal{}, fmt.Errorf("expected a numeric value, got %T", value)
	}
}

func coerceList(field *reportcatalog.Field, value any) ([]any, error) {
	raw, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("expected a list, got %T", value)
	}
	if len(raw) == 0 {
		return nil, errors.New("list must not be empty")
	}

	coerced := make([]any, 0, len(raw))
	for _, item := range raw {
		c, err := coerceScalar(field.Type, field, item)
		if err != nil {
			return nil, err
		}
		coerced = append(coerced, c)
	}
	return coerced, nil
}

func coerceFilterValue(op dbtype.Operator, field *reportcatalog.Field, value any) error {
	//nolint:exhaustive // scalar operators are coerced in the default branch
	switch op {
	case dbtype.OpIn, dbtype.OpNotIn:
		_, err := coerceList(field, value)
		return err
	case dbtype.OpDateRange:
		_, _, err := coerceDateRange(value, time.UTC)
		return err
	case dbtype.OpLastNDays, dbtype.OpNextNDays:
		n, err := coerceInteger(value)
		if err != nil {
			return err
		}
		if n <= 0 || n > 3650 {
			return errors.New("day count must be between 1 and 3650")
		}
		return nil
	case dbtype.OpContains, dbtype.OpStartsWith, dbtype.OpEndsWith,
		dbtype.OpLike, dbtype.OpILike:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected a string pattern, got %T", value)
		}
		return nil
	default:
		_, err := coerceScalar(field.Type, field, value)
		return err
	}
}

func coerceDateRange(value any, loc *time.Location) (start, end int64, err error) {
	raw, ok := value.([]any)
	if !ok || len(raw) != 2 {
		return 0, 0, errors.New("date range expects a [start, end] pair")
	}

	start, err = coerceDateBoundary(raw[0], loc, false)
	if err != nil {
		return 0, 0, fmt.Errorf("range start: %w", err)
	}
	end, err = coerceDateBoundary(raw[1], loc, true)
	if err != nil {
		return 0, 0, fmt.Errorf("range end: %w", err)
	}
	if end <= start {
		return 0, 0, errors.New("range end must be after range start")
	}
	return start, end, nil
}

func coerceDateBoundary(value any, loc *time.Location, endExclusive bool) (int64, error) {
	switch v := value.(type) {
	case string:
		day, err := time.ParseInLocation("2006-01-02", v, loc)
		if err != nil {
			return 0, fmt.Errorf("expected YYYY-MM-DD or epoch seconds, got %q", v)
		}
		if endExclusive {
			day = day.AddDate(0, 0, 1)
		}
		return day.Unix(), nil
	default:
		return coerceInteger(value)
	}
}

func coerceParamValue(param *report.ParameterDef, value any) (any, error) {
	if param.Multi {
		raw, ok := value.([]any)
		if !ok {
			return nil, fmt.Errorf("expected a list, got %T", value)
		}
		if len(raw) == 0 {
			return nil, errors.New("list must not be empty")
		}
		coerced := make([]any, 0, len(raw))
		for _, item := range raw {
			c, err := coerceScalar(param.Type, nil, item)
			if err != nil {
				return nil, err
			}
			if err = checkAllowedValue(param, c); err != nil {
				return nil, err
			}
			coerced = append(coerced, c)
		}
		return coerced, nil
	}

	coerced, err := coerceScalar(param.Type, nil, value)
	if err != nil {
		return nil, err
	}
	if err = checkAllowedValue(param, coerced); err != nil {
		return nil, err
	}
	return coerced, nil
}

// checkAllowedValue enforces the parameter's declared value set. Allowed
// values are stored as strings; coerced values are compared through their
// canonical string form so int/decimal parameters can carry allow-lists too.
func checkAllowedValue(param *report.ParameterDef, coerced any) error {
	if len(param.AllowedValues) == 0 {
		return nil
	}

	var canonical string
	switch v := coerced.(type) {
	case string:
		canonical = v
	case int64:
		canonical = strconv.FormatInt(v, 10)
	case decimal.Decimal:
		canonical = v.String()
	case bool:
		canonical = strconv.FormatBool(v)
	default:
		return fmt.Errorf("values of type %T cannot carry an allow-list", coerced)
	}

	if slices.Contains(param.AllowedValues, canonical) {
		return nil
	}
	return fmt.Errorf("%q is not one of the allowed values", canonical)
}
