package stringutils

import "strings"

func Truncate(value string, maxLength int) string {
	if len(value) > maxLength {
		return value[:maxLength] + "..."
	}
	return value
}

func TruncateAndTrim(value string, maxLength int) string {
	value = strings.TrimSpace(value)

	return Truncate(value, maxLength)
}
