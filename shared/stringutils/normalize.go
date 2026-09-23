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

// FirstSentence returns the text up to and including its first full stop,
// or the whole text when it has none.
func FirstSentence(text string) string {
	if index := strings.IndexByte(text, '.'); index > 0 {
		return text[:index+1]
	}

	return text
}

// CollapseWhitespace folds every run of whitespace, line breaks included,
// into one space and trims the ends, so a value fits on a single line.
func CollapseWhitespace(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
