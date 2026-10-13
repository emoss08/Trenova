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

func CountNonSpace(value string) int {
	count := 0
	for _, r := range value {
		if !unicode.IsSpace(r) {
			count++
		}
	}

	return count
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

// OneLine folds a value onto one line and cuts it to at most limit runes, so
// it cannot break the line it is written on.
func OneLine(value string, limit int) string {
	return TruncateRunes(CollapseWhitespace(value), limit)
}

// IsCompactToken reports whether value is a non-empty identifier of at most
// maxLength runes with no whitespace, such as a client-generated event id.
func IsCompactToken(value string, maxLength int) bool {
	if value == "" || maxLength <= 0 {
		return false
	}
	count := 0
	for _, r := range value {
		if unicode.IsSpace(r) || !unicode.IsPrint(r) {
			return false
		}
		count++
		if count > maxLength {
			return false
		}
	}
	return true
}
