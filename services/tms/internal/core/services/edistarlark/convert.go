package edistarlark

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/shopspring/decimal"
	"go.starlark.net/starlark"
)

func toFrozenStarlarkValue(value any) (starlark.Value, error) {
	converted, err := toStarlarkValue(value)
	if err != nil {
		return nil, err
	}
	converted.Freeze()
	return converted, nil
}

func toStarlarkValue(value any) (starlark.Value, error) {
	switch typed := value.(type) {
	case nil:
		return starlark.None, nil
	case starlark.Value:
		return typed, nil
	case string:
		return starlark.String(typed), nil
	case bool:
		return starlark.Bool(typed), nil
	case int:
		return starlark.MakeInt(typed), nil
	case int8:
		return starlark.MakeInt(int(typed)), nil
	case int16:
		return starlark.MakeInt(int(typed)), nil
	case int32:
		return starlark.MakeInt(int(typed)), nil
	case int64:
		return starlark.MakeInt64(typed), nil
	case uint:
		return starlark.MakeUint(typed), nil
	case uint8:
		return starlark.MakeUint(uint(typed)), nil
	case uint16:
		return starlark.MakeUint(uint(typed)), nil
	case uint32:
		return starlark.MakeUint(uint(typed)), nil
	case uint64:
		return starlark.MakeUint64(typed), nil
	case float32:
		return starlark.Float(typed), nil
	case float64:
		return starlark.Float(typed), nil
	case decimal.Decimal:
		return starlark.String(typed.String()), nil
	case decimal.NullDecimal:
		if !typed.Valid {
			return starlark.None, nil
		}
		return starlark.String(typed.Decimal.String()), nil
	case fmt.Stringer:
		return starlark.String(typed.String()), nil
	case map[string]any:
		return mapToStarlark(typed)
	case []any:
		return sliceToStarlark(typed)
	}

	normalized, err := normalizeForStarlark(value)
	if err != nil {
		return nil, err
	}
	return toStarlarkValue(normalized)
}

func normalizeForStarlark(value any) (any, error) {
	bytes, err := sonic.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal Starlark context value %T: %w", value, err)
	}

	var normalized any
	if err = sonic.Unmarshal(bytes, &normalized); err != nil {
		return nil, fmt.Errorf("unmarshal Starlark context value %T: %w", value, err)
	}
	return normalized, nil
}

func mapToStarlark(values map[string]any) (starlark.Value, error) {
	dict := starlark.NewDict(len(values))
	for key, value := range values {
		converted, err := toStarlarkValue(value)
		if err != nil {
			return nil, fmt.Errorf("convert key %q: %w", key, err)
		}
		if err = dict.SetKey(starlark.String(key), converted); err != nil {
			return nil, fmt.Errorf("set key %q: %w", key, err)
		}
	}
	return dict, nil
}

func sliceToStarlark(values []any) (starlark.Value, error) {
	items := make([]starlark.Value, 0, len(values))
	for i, value := range values {
		converted, err := toStarlarkValue(value)
		if err != nil {
			return nil, fmt.Errorf("convert index %d: %w", i, err)
		}
		items = append(items, converted)
	}
	return starlark.NewList(items), nil
}

func scalarString(value starlark.Value) (string, bool) {
	switch typed := value.(type) {
	case starlark.NoneType:
		return "", true
	case starlark.String:
		return strings.TrimSpace(string(typed)), true
	case starlark.Int:
		if intValue, ok := typed.Int64(); ok {
			return strconv.FormatInt(intValue, 10), true
		}
		return typed.BigInt().String(), true
	case starlark.Float:
		floatValue := float64(typed)
		if math.IsNaN(floatValue) || math.IsInf(floatValue, 0) {
			return "", false
		}
		return trimFloat(floatValue), true
	case starlark.Bool:
		if bool(typed) {
			return "Y", true
		}
		return "N", true
	default:
		return "", false
	}
}

func stringify(value starlark.Value) string {
	switch typed := value.(type) {
	case starlark.NoneType:
		return ""
	case starlark.String:
		return string(typed)
	case starlark.Int:
		if intValue, ok := typed.Int64(); ok {
			return strconv.FormatInt(intValue, 10)
		}
		return typed.BigInt().String()
	case starlark.Float:
		return trimFloat(float64(typed))
	case starlark.Bool:
		if bool(typed) {
			return "Y"
		}
		return "N"
	default:
		return strings.TrimSpace(value.String())
	}
}

func isEmptyValue(value starlark.Value) bool {
	if value == nil || value == starlark.None {
		return true
	}
	if sized, ok := value.(interface{ Len() int }); ok && sized.Len() == 0 {
		return true
	}
	return strings.TrimSpace(stringify(value)) == ""
}

func trimFloat(value float64) string {
	if value == float64(int64(value)) {
		return strconv.FormatInt(int64(value), 10)
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}
