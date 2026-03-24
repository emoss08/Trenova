package stringutils

import "strings"

func FilterEmpty(values []string) []string {
	var result []string
	for _, v := range values {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
