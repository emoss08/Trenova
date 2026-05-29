package stringutils

import "strings"

func WithDefault(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}

	return trimmed
}

func SplitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{}
	}

	raw := strings.Split(value, ",")
	values := make([]string, 0, len(raw))
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item != "" {
			values = append(values, item)
		}
	}

	return values
}
