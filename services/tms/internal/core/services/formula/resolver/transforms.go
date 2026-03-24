package resolver

import (
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type TransformFunc func(value any) (any, error)

var transforms = map[string]TransformFunc{
	"decimalToFloat64": decimalToFloat64,
	"int64ToFloat64":   int64ToFloat64,
	"int16ToFloat64":   int16ToFloat64,
	"stringToUpper":    stringToUpper,
	"stringToLower":    stringToLower,
	"unixToISO8601":    unixToISO8601,
}

func GetTransform(name string) (TransformFunc, bool) {
	fn, ok := transforms[name]
	return fn, ok
}

func RegisterTransform(name string, fn TransformFunc) {
	transforms[name] = fn
}

func decimalToFloat64(value any) (any, error) {
	switch v := value.(type) {
	case decimal.Decimal:
		f, _ := v.Float64()
		return f, nil
	case *decimal.Decimal:
		if v == nil {
			return 0.0, nil
		}
		f, _ := v.Float64()
		return f, nil
	case decimal.NullDecimal:
		if !v.Valid {
			return 0.0, nil
		}
		f, _ := v.Decimal.Float64()
		return f, nil
	case *decimal.NullDecimal:
		if v == nil || !v.Valid {
			return 0.0, nil
		}
		f, _ := v.Decimal.Float64()
		return f, nil
	default:
		return nil, fmt.Errorf("cannot convert %T to float64", value)
	}
}

func int64ToFloat64(value any) (any, error) {
	switch v := value.(type) {
	case int64:
		return float64(v), nil
	case *int64:
		if v == nil {
			return 0.0, nil
		}
		return float64(*v), nil
	case int:
		return float64(v), nil
	case *int:
		if v == nil {
			return 0.0, nil
		}
		return float64(*v), nil
	default:
		return nil, fmt.Errorf("cannot convert %T to float64", value)
	}
}

func int16ToFloat64(value any) (any, error) {
	switch v := value.(type) {
	case int16:
		return float64(v), nil
	case *int16:
		if v == nil {
			return 0.0, nil
		}
		return float64(*v), nil
	default:
		return nil, fmt.Errorf("cannot convert %T to float64", value)
	}
}

func stringToUpper(value any) (any, error) {
	switch v := value.(type) {
	case string:
		return strings.ToUpper(v), nil
	case *string:
		if v == nil {
			return "", nil
		}
		return strings.ToUpper(*v), nil
	default:
		return nil, fmt.Errorf("cannot convert %T to uppercase string", value)
	}
}

func stringToLower(value any) (any, error) {
	switch v := value.(type) {
	case string:
		return strings.ToLower(v), nil
	case *string:
		if v == nil {
			return "", nil
		}
		return strings.ToLower(*v), nil
	default:
		return nil, fmt.Errorf("cannot convert %T to lowercase string", value)
	}
}

func unixToISO8601(value any) (any, error) {
	switch v := value.(type) {
	case int64:
		return time.Unix(v, 0).UTC().Format(time.RFC3339), nil
	case *int64:
		if v == nil {
			return "", nil
		}
		return time.Unix(*v, 0).UTC().Format(time.RFC3339), nil
	default:
		return nil, fmt.Errorf("cannot convert %T to ISO8601", value)
	}
}
