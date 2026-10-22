package resolver

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/modules/rating/app/formula/errors"
	"github.com/shopspring/decimal"
)

type TransformFunc func(value any) (any, error)

var (
	transformsMu sync.RWMutex
	transforms   = map[string]TransformFunc{
		"decimal":       exactDecimal,
		"integer":       exactInteger,
		"stringToUpper": stringToUpper,
		"stringToLower": stringToLower,
		"unixToISO8601": unixToISO8601,
	}
)

func GetTransform(name string) (TransformFunc, bool) {
	transformsMu.RLock()
	defer transformsMu.RUnlock()
	fn, ok := transforms[name]
	return fn, ok
}

func RegisterTransform(name string, fn TransformFunc) {
	transformsMu.Lock()
	defer transformsMu.Unlock()
	transforms[name] = fn
}

func exactDecimal(value any) (any, error) {
	switch v := value.(type) {
	case decimal.Decimal:
		return v, nil
	case *decimal.Decimal:
		if v != nil {
			return *v, nil
		}
	case decimal.NullDecimal:
		if v.Valid {
			return v.Decimal, nil
		}
	case *decimal.NullDecimal:
		if v != nil && v.Valid {
			return v.Decimal, nil
		}
	default:
		return nil, fmt.Errorf("expected a number, got %s", errors.DescribeValue(value))
	}
	return nil, nil //nolint:nilnil // an empty decimal is the formula's null
}

func exactInteger(value any) (any, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case *int64:
		if v != nil {
			return *v, nil
		}
	case int:
		return int64(v), nil
	case *int:
		if v != nil {
			return int64(*v), nil
		}
	case int16:
		return int64(v), nil
	case *int16:
		if v != nil {
			return int64(*v), nil
		}
	case int32:
		return int64(v), nil
	case *int32:
		if v != nil {
			return int64(*v), nil
		}
	default:
		return nil, fmt.Errorf("expected a whole number, got %s", errors.DescribeValue(value))
	}
	return nil, nil //nolint:nilnil // an empty integer is the formula's null
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
		return nil, fmt.Errorf("expected text to uppercase, got %s", errors.DescribeValue(value))
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
		return nil, fmt.Errorf("expected text to lowercase, got %s", errors.DescribeValue(value))
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
		return nil, fmt.Errorf("expected a date, got %s", errors.DescribeValue(value))
	}
}
