/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package maputils

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
		// Handle {"long": value} format
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
