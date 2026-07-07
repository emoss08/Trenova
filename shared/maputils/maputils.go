package maputils

import (
	"fmt"
	"maps"
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
