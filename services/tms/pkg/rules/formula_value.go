package rules

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/shopspring/decimal"
)

var (
	ErrInexactNumber    = errors.New("a formula value must be exact, not a float")
	ErrUnsupportedValue = errors.New(
		"a formula value must be a number, text, flag, date, duration, list, map or nil",
	)
	ErrReservedKey = errors.New("a formula value's map key may not start with $")
)

type RawJSON []byte

func (r RawJSON) MarshalJSON() ([]byte, error) {
	if len(r) == 0 {
		return []byte("null"), nil
	}
	return r, nil
}

func (r *RawJSON) UnmarshalJSON(data []byte) error {
	*r = append((*r)[:0], data...)
	return nil
}

func EncodeValue(value any) ([]byte, error) {
	return AppendValue(make([]byte, 0, 256), value)
}

func AppendValue(dst []byte, value any) ([]byte, error) {
	return appendValue(dst, value, "")
}

func appendValue(dst []byte, value any, path string) ([]byte, error) {
	if out, ok := appendInteger(dst, value); ok {
		return out, nil
	}
	if out, ok := appendScalar(dst, value); ok {
		return out, nil
	}
	switch typed := value.(type) {
	case float64, float32:
		return nil, fmt.Errorf("%w: %s is %v", ErrInexactNumber, describePath(path), typed)
	case map[string]any:
		return appendStringMap(dst, typed, path)
	case []any:
		dst = append(dst, '[')
		for index, item := range typed {
			if index > 0 {
				dst = append(dst, ',')
			}
			var err error
			if dst, err = appendValue(dst, item, itemPath(path, index)); err != nil {
				return nil, err
			}
		}
		return append(dst, ']'), nil
	}
	return appendReflected(dst, reflect.ValueOf(value), path)
}

func appendInteger(dst []byte, value any) ([]byte, bool) {
	switch typed := value.(type) {
	case int:
		return strconv.AppendInt(dst, int64(typed), 10), true
	case int8:
		return strconv.AppendInt(dst, int64(typed), 10), true
	case int16:
		return strconv.AppendInt(dst, int64(typed), 10), true
	case int32:
		return strconv.AppendInt(dst, int64(typed), 10), true
	case int64:
		return strconv.AppendInt(dst, typed, 10), true
	case uint:
		return strconv.AppendUint(dst, uint64(typed), 10), true
	case uint8:
		return strconv.AppendUint(dst, uint64(typed), 10), true
	case uint16:
		return strconv.AppendUint(dst, uint64(typed), 10), true
	case uint32:
		return strconv.AppendUint(dst, uint64(typed), 10), true
	case uint64:
		return strconv.AppendUint(dst, typed, 10), true
	}
	return dst, false
}

func appendScalar(dst []byte, value any) ([]byte, bool) {
	switch typed := value.(type) {
	case nil:
		return append(dst, "null"...), true
	case RawJSON:
		if len(typed) == 0 {
			return append(dst, "null"...), true
		}
		return append(dst, typed...), true
	case bool:
		return strconv.AppendBool(dst, typed), true
	case string:
		return appendString(dst, typed), true
	case decimal.Decimal:
		return append(dst, typed.String()...), true
	case *decimal.Decimal:
		if typed == nil {
			return append(dst, "null"...), true
		}
		return append(dst, typed.String()...), true
	case decimal.NullDecimal:
		if !typed.Valid {
			return append(dst, "null"...), true
		}
		return append(dst, typed.Decimal.String()...), true
	case time.Time:
		return appendDate(dst, typed), true
	case *time.Time:
		if typed == nil {
			return append(dst, "null"...), true
		}
		return appendDate(dst, *typed), true
	case time.Duration:
		dst = append(dst, `{"$duration":"`...)
		dst = strconv.AppendInt(dst, int64(typed), 10)
		return append(dst, `"}`...), true
	}
	return dst, false
}

func appendStringMap(dst []byte, values map[string]any, path string) ([]byte, error) {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	dst = append(dst, '{')
	for index, key := range keys {
		if strings.HasPrefix(key, "$") {
			return nil, fmt.Errorf("%w: %s", ErrReservedKey, describePath(fieldPath(path, key)))
		}
		if index > 0 {
			dst = append(dst, ',')
		}
		dst = appendString(dst, key)
		dst = append(dst, ':')
		var err error
		if dst, err = appendValue(dst, values[key], fieldPath(path, key)); err != nil {
			return nil, err
		}
	}
	return append(dst, '}'), nil
}

func appendReflected(dst []byte, value reflect.Value, path string) ([]byte, error) {
	switch value.Kind() { //nolint:exhaustive // every other kind is refused below
	case reflect.Pointer, reflect.Interface:
		if value.IsNil() {
			return append(dst, "null"...), nil
		}
		return appendValue(dst, value.Elem().Interface(), path)
	case reflect.Slice, reflect.Array:
		if value.Kind() == reflect.Slice && value.IsNil() {
			return append(dst, "[]"...), nil
		}
		dst = append(dst, '[')
		for index := range value.Len() {
			if index > 0 {
				dst = append(dst, ',')
			}
			var err error
			dst, err = appendValue(dst, value.Index(index).Interface(), itemPath(path, index))
			if err != nil {
				return nil, err
			}
		}
		return append(dst, ']'), nil
	case reflect.Map:
		if value.Type().Key().Kind() != reflect.String {
			break
		}
		values := make(map[string]any, value.Len())
		iter := value.MapRange()
		for iter.Next() {
			values[iter.Key().String()] = iter.Value().Interface()
		}
		return appendStringMap(dst, values, path)
	case reflect.String:
		return appendString(dst, value.String()), nil
	case reflect.Bool:
		return strconv.AppendBool(dst, value.Bool()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.AppendInt(dst, value.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.AppendUint(dst, value.Uint(), 10), nil
	case reflect.Float32, reflect.Float64:
		return nil, fmt.Errorf(
			"%w: %s is %v",
			ErrInexactNumber,
			describePath(path),
			value.Interface(),
		)
	}
	if !value.IsValid() {
		return append(dst, "null"...), nil
	}
	return nil, fmt.Errorf("%w: %s is a %s", ErrUnsupportedValue, describePath(path), value.Type())
}

func appendDate(dst []byte, value time.Time) []byte {
	dst = append(dst, `{"$date":"`...)
	dst = value.AppendFormat(dst, time.RFC3339Nano)
	return append(dst, `"}`...)
}

const hexDigits = "0123456789abcdef"

func appendString(dst []byte, text string) []byte {
	dst = append(dst, '"')
	start := 0
	for index := 0; index < len(text); {
		c := text[index]
		if c < utf8.RuneSelf {
			if c >= 0x20 && c != '"' && c != '\\' {
				index++
				continue
			}
			dst = append(dst, text[start:index]...)
			switch c {
			case '"', '\\':
				dst = append(dst, '\\', c)
			case '\n':
				dst = append(dst, '\\', 'n')
			case '\r':
				dst = append(dst, '\\', 'r')
			case '\t':
				dst = append(dst, '\\', 't')
			default:
				dst = append(dst, '\\', 'u', '0', '0', hexDigits[c>>4], hexDigits[c&0xf])
			}
			index++
			start = index
			continue
		}
		r, size := utf8.DecodeRuneInString(text[index:])
		if r == utf8.RuneError && size == 1 {
			dst = append(dst, text[start:index]...)
			dst = append(dst, `�`...)
			index += size
			start = index
			continue
		}
		index += size
	}
	dst = append(dst, text[start:]...)
	return append(dst, '"')
}

func fieldPath(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

func itemPath(path string, index int) string {
	return path + "[" + strconv.Itoa(index) + "]"
}

func describePath(path string) string {
	if path == "" {
		return "the value"
	}
	return path
}
