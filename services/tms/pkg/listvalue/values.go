package listvalue

import (
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/keel/schema"
)

const maxExactJSONInteger = 1 << 53

type Kind uint8

const (
	KindText Kind = iota + 1
	KindInt
	KindFloat
	KindBool
	KindTime
)

func KindOf(col *schema.Column, text bool) (Kind, bool) {
	if text {
		return KindText, true
	}

	switch col.Kind {
	case schema.KindString:
		return KindText, true
	case schema.KindInt16, schema.KindInt32, schema.KindInt64:
		return KindInt, true
	case schema.KindFloat32, schema.KindFloat64:
		return KindFloat, true
	case schema.KindBool:
		return KindBool, true
	case schema.KindTime:
		return KindTime, true
	case schema.KindOther,
		schema.KindBytes,
		schema.KindJSON,
		schema.KindStringArray,
		schema.KindInt64Array,
		schema.KindTSVector:
		return 0, false
	default:
		return 0, false
	}
}

func Convert(kind Kind, value any) (any, bool) {
	switch kind {
	case KindText:
		return ToText(value)
	case KindInt:
		return toInt(value)
	case KindFloat:
		return ToFloat(value)
	case KindBool:
		return toBool(value)
	case KindTime:
		return toTime(value)
	default:
		return nil, false
	}
}

func ConvertList(kind Kind, value any) (any, bool) {
	items, ok := listItems(value)
	if !ok {
		return nil, false
	}

	switch kind {
	case KindText:
		return convertEach[string](items, ToText)
	case KindInt:
		return convertEach[int64](items, toInt)
	case KindFloat:
		return convertEach[float64](items, ToFloat)
	case KindBool:
		return convertEach[bool](items, toBool)
	case KindTime:
		return convertEach[time.Time](items, toTime)
	default:
		return nil, false
	}
}

func convertEach[V any](items []any, conv func(any) (any, bool)) (any, bool) {
	out := make([]V, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		converted, ok := conv(item)
		if !ok {
			return nil, false
		}
		typed, ok := converted.(V)
		if !ok {
			return nil, false
		}
		out = append(out, typed)
	}

	return out, true
}

func listItems(value any) ([]any, bool) {
	if value == nil {
		return nil, false
	}

	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return []any{value}, true
	}

	out := make([]any, rv.Len())
	for i := range out {
		out[i] = rv.Index(i).Interface()
	}

	return out, true
}

type jsonNumber interface {
	Int64() (int64, error)
	Float64() (float64, error)
	String() string
}

func ToText(value any) (any, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case jsonNumber:
		return v.String(), true
	case bool:
		return strconv.FormatBool(v), true
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), true
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32), true
	}

	if n, ok := integerOf(value); ok {
		return strconv.FormatInt(n, 10), true
	}
	if s, ok := stringOf(value); ok {
		return s, true
	}

	return nil, false
}

func toInt(value any) (any, bool) {
	if n, ok := integerOf(value); ok {
		return n, true
	}

	switch v := value.(type) {
	case float64:
		return wholeNumber(v)
	case float32:
		return wholeNumber(float64(v))
	case jsonNumber:
		if n, err := v.Int64(); err == nil {
			return n, true
		}
		return nil, false
	case string:
		trimmed := strings.TrimSpace(v)
		if n, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
			return n, true
		}
		if f, err := strconv.ParseFloat(trimmed, 64); err == nil {
			return wholeNumber(f)
		}
	}

	return nil, false
}

func wholeNumber(f float64) (any, bool) {
	if math.Trunc(f) != f || math.Abs(f) > maxExactJSONInteger {
		return nil, false
	}

	return int64(f), true
}

func ToFloat(value any) (any, bool) {
	if n, ok := integerOf(value); ok {
		return float64(n), true
	}

	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case jsonNumber:
		if f, err := v.Float64(); err == nil {
			return f, true
		}
	case string:
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			return f, true
		}
	}

	return nil, false
}

func toBool(value any) (any, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		b, err := strconv.ParseBool(strings.TrimSpace(v))
		if err != nil {
			return nil, false
		}
		return b, true
	}

	return nil, false
}

func toTime(value any) (any, bool) {
	switch v := value.(type) {
	case time.Time:
		return v, true
	case string:
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(v))
		if err != nil {
			return nil, false
		}
		return t, true
	}

	if n, ok := toInt(value); ok {
		seconds, isInt := n.(int64)
		if isInt {
			return time.Unix(seconds, 0).UTC(), true
		}
	}

	return nil, false
}

func integerOf(value any) (int64, bool) {
	rv := reflect.ValueOf(value)
	switch rv.Kind() { //nolint:exhaustive // only integer kinds are integers
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int(), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u := rv.Uint()
		if u > math.MaxInt64 {
			return 0, false
		}
		return int64(u), true
	default:
		return 0, false
	}
}

func stringOf(value any) (string, bool) {
	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.String {
		return "", false
	}

	return rv.String(), true
}
