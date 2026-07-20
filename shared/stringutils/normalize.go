package stringutils

import (
	"strings"
	"unicode"
)

func NormalizeIdentifier(value string) string {
	buf := make([]rune, 0, len(value))
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			buf = append(buf, unicode.ToUpper(r))
		}
	}
	return string(buf)
}

func TruncateRunes(value string, maxLength int) string {
	if maxLength <= 0 {
		return ""
	}

	runes := []rune(value)
	if len(runes) <= maxLength {
		return value
	}

	return string(runes[:maxLength])
}

func Ellipsize(value string, maxLength int) string {
	value = strings.TrimSpace(value)
	truncated := TruncateRunes(value, maxLength)
	if truncated == value {
		return value
	}

	return strings.TrimRight(truncated, " ") + "…"
}
