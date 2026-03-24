package formulatypes

import "strings"

func SetNestedValue(m map[string]any, path string, value any) {
	parts := strings.Split(path, ".")
	current := m

	for i, part := range parts {
		part = strings.TrimSuffix(part, "[]")

		if i == len(parts)-1 {
			current[part] = value
			return
		}

		if _, exists := current[part]; !exists {
			current[part] = make(map[string]any)
		}

		if nested, ok := current[part].(map[string]any); ok {
			current = nested
		} else {
			return
		}
	}
}
