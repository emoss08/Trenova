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

func FormatEmailAddress(name, address string) string {
	name = strings.TrimSpace(name)
	address = strings.TrimSpace(address)
	if name == "" {
		return address
	}
	return name + " <" + address + ">"
}
