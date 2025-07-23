// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package conversion

import (
	"github.com/shopspring/decimal"
)

// * ToFloat64 safely converts various numeric types to float64
func ToFloat64(v any) (float64, bool) { //nolint:cyclop,funlen // we need to keep this function long
	if v == nil {
		return 0, false
	}

	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int8:
		return float64(val), true
	case int16:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	case uint:
		return float64(val), true
	case uint8:
		return float64(val), true
	case uint16:
		return float64(val), true
	case uint32:
		return float64(val), true
	case uint64:
		return float64(val), true

	// Pointer types
	case *float64:
		if val != nil {
			return *val, true
		}
		return 0, false
	case *float32:
		if val != nil {
			return float64(*val), true
		}
		return 0, false
	case *int:
		if val != nil {
			return float64(*val), true
		}
		return 0, false
	case *int16:
		if val != nil {
			return float64(*val), true
		}
		return 0, false
	case *int32:
		if val != nil {
			return float64(*val), true
		}
		return 0, false
	case *int64:
		if val != nil {
			return float64(*val), true
		}
		return 0, false

	// Decimal types
	case decimal.Decimal:
		f, _ := val.Float64()
		return f, true
	case decimal.NullDecimal:
		if val.Valid {
			f, _ := val.Decimal.Float64()
			return f, true
		}
		return 0, false

	default:
		return 0, false
	}
}

// * ToFloat64OrZero converts to float64, returning 0 if conversion fails
func ToFloat64OrZero(v any) float64 {
	f, _ := ToFloat64(v)
	return f
}

// * ToInt64 safely converts various numeric types to int64
func ToInt64(v any) (int64, bool) { //nolint:cyclop,funlen // we need to keep this function long
	if v == nil {
		return 0, false
	}

	switch val := v.(type) {
	case int64:
		return val, true
	case int:
		return int64(val), true
	case int8:
		return int64(val), true
	case int16:
		return int64(val), true
	case int32:
		return int64(val), true
	case uint:
		return int64(val), true
	case uint8:
		return int64(val), true
	case uint16:
		return int64(val), true
	case uint32:
		return int64(val), true
	case uint64:
		// Check for overflow
		if val <= 9223372036854775807 {
			return int64(val), true
		}
		return 0, false
	case float32:
		return int64(val), true
	case float64:
		return int64(val), true

	// Pointer types
	case *int64:
		if val != nil {
			return *val, true
		}
		return 0, false
	case *int:
		if val != nil {
			return int64(*val), true
		}
		return 0, false
	case *int16:
		if val != nil {
			return int64(*val), true
		}
		return 0, false
	case *int32:
		if val != nil {
			return int64(*val), true
		}
		return 0, false

	// Decimal types
	case decimal.Decimal:
		return val.IntPart(), true
	case decimal.NullDecimal:
		if val.Valid {
			return val.Decimal.IntPart(), true
		}
		return 0, false

	default:
		return 0, false
	}
}

// * ToBool safely converts various types to bool
func ToBool(v any) (bool, bool) {
	if v == nil {
		return false, false
	}

	switch val := v.(type) {
	case bool:
		return val, true
	case *bool:
		if val != nil {
			return *val, true
		}
		return false, false
	case int, int8, int16, int32, int64:
		i, _ := ToInt64(val)
		return i != 0, true
	case uint, uint8, uint16, uint32, uint64:
		i, _ := ToInt64(val)
		return i != 0, true
	case float32, float64:
		f, _ := ToFloat64(val)
		return f != 0, true
	case string:
		return val != "", true
	default:
		return false, false
	}
}
