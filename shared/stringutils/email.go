package stringutils

import "strings"

func NormalizeEmailAddress(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func NormalizeEmailAddresses(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		normalized := NormalizeEmailAddress(value)
		if normalized != "" {
			result = append(result, normalized)
		}
	}
	return result
}

// SplitEmailList parses a free-form recipient list separated by commas,
// semicolons, newlines, or tabs into normalized, deduplicated addresses.
func SplitEmailList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\t'
	})

	normalized := NormalizeEmailAddresses(parts)
	result := make([]string, 0, len(normalized))
	seen := make(map[string]struct{}, len(normalized))
	for _, item := range normalized {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}

	return result
}

func FormatEmailAddress(name, address string) string {
	name = strings.TrimSpace(name)
	address = strings.TrimSpace(address)
	if name == "" {
		return address
	}
	return name + " <" + address + ">"
}
