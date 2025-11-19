package maputils

import "fmt"

func ExtractStringField(data map[string]any, key string) string {
	val, ok := data[key].(string)

	if !ok {
		return ""
	}

	return val
}

func ExtractInt64Field(field any) int64 {
	if field == nil {
		return 0
	}
	switch v := field.(type) {
	case int64:
		return v
	case float64:
		return int64(v)
	case int:
		return int64(v)
	case map[string]any:
		if longVal, ok := v["long"]; ok {
			switch lv := longVal.(type) {
			case int64:
				return lv
			case float64:
				return int64(lv)
			}
		}
	}
	return 0
}

func GetString(data map[string]any, key string) (string, error) {
	val, ok := data[key]
	if !ok {
		return "", fmt.Errorf("missing field: %s", key)
	}
	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("field %s is not a string", key)
	}
	return str, nil
}

func GetInt(data map[string]any, key string) (int, error) {
	val, ok := data[key]
	if !ok {
		return 0, fmt.Errorf("missing field: %s", key)
	}
	switch v := val.(type) {
	case float64:
		return int(v), nil
	case int:
		return v, nil
	case int64:
		return int(v), nil
	default:
		return 0, fmt.Errorf("field %s is not a number", key)
	}
}

func GetBool(data map[string]any, key string) (bool, error) {
	val, ok := data[key]
	if !ok {
		return false, fmt.Errorf("missing field: %s", key)
	}
	b, ok := val.(bool)
	if !ok {
		return false, fmt.Errorf("field %s is not a boolean", key)
	}
	return b, nil
}

func GetMap(data map[string]any, key string) (map[string]any, error) {
	val, ok := data[key]
	if !ok {
		return nil, fmt.Errorf("missing field: %s", key)
	}
	m, ok := val.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("field %s is not a map", key)
	}
	return m, nil
}

func GetArray(data map[string]any, key string) ([]any, error) {
	val, ok := data[key]
	if !ok {
		return nil, nil
	}
	arr, ok := val.([]any)
	if !ok {
		return nil, fmt.Errorf("field %s is not an array", key)
	}
	return arr, nil
}
