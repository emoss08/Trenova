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

// NormalizeEmailList normalizes each address and drops blanks and
// duplicates, keeping the order the addresses were given in. It is the
// []string counterpart to SplitEmailList, for callers that already hold a
// list rather than a free-form field.
func NormalizeEmailList(values []string) []string {
	return dedupeAddresses(NormalizeEmailAddresses(values))
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

	return dedupeAddresses(NormalizeEmailAddresses(parts))
}

func dedupeAddresses(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, item := range values {
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

func EmailDomain(address string) string {
	at := strings.LastIndexByte(address, '@')
	if at < 0 {
		return ""
	}
	return strings.TrimSuffix(NormalizeEmailAddress(address[at+1:]), ">")
}
