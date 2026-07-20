package maputils

import (
	"fmt"
	"maps"
	"reflect"
	"strconv"
	"strings"
)

func Path(root any, path string) any {
	current := root
	for part := range strings.SplitSeq(path, ".") {
		if part == "" {
			continue
		}
		switch typed := current.(type) {
		case map[string]any:
			current = typed[part]
		case []any:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= len(typed) {
				return nil
			}
			current = typed[index]
		default:
			return nil
		}
	}
	return current
}

func CloneShallow(input map[string]any) map[string]any {
	output := make(map[string]any, len(input))
	maps.Copy(output, input)
	return output
}

func WithoutFuncValues(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}

	output := make(map[string]any, len(input))
	for key, value := range input {
		if value != nil && reflect.TypeOf(value).Kind() == reflect.Func {
			continue
		}
		output[key] = value
	}

	return output
}

func IntValue(input map[string]any, key string) (value int64, ok bool) {
	if len(input) == 0 {
		return 0, false
	}
	switch typed := input[key].(type) {
	case int:
		return int64(typed), true
	case int32:
		return int64(typed), true
	case int64:
		return typed, true
	case float64:
		if typed != float64(int64(typed)) {
			return 0, false
		}
		return int64(typed), true
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func BoolValue(input map[string]any, key string) (value, ok bool) {
	if len(input) == 0 {
		return false, false
	}
	switch value := input[key].(type) {
	case bool:
		return value, true
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(value))
		if err != nil {
			return false, false
		}
		return parsed, true
	default:
		return false, false
	}
}

func StringValue(input map[string]any, key string) string {
	if len(input) == 0 {
		return ""
	}
	switch value := input[key].(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(value)
	case fmt.Stringer:
		return strings.TrimSpace(value.String())
	case int:
		return strconv.Itoa(value)
	case int64:
		return strconv.FormatInt(value, 10)
	case float64:
		if value == float64(int64(value)) {
			return strconv.FormatInt(int64(value), 10)
		}
		return strings.TrimSpace(strconv.FormatFloat(value, 'f', -1, 64))
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}
