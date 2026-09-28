package stringutils

import (
	"strings"
	"unicode/utf8"
)

func IndexFold(text, needle string) int {
	if needle == "" {
		return -1
	}
	lowered := strings.ToLower(text)
	if len(lowered) != len(text) {
		return strings.Index(text, needle)
	}

	return strings.Index(lowered, strings.ToLower(needle))
}

func ExcerptAround(text string, start, end, context int) string {
	if start < 0 || end > len(text) || start > end {
		return ""
	}
	from := max(0, start-max(context, 0))
	for from > 0 && !utf8.RuneStart(text[from]) {
		from--
	}
	to := min(len(text), end+max(context, 0))
	for to < len(text) && !utf8.RuneStart(text[to]) {
		to++
	}

	return CollapseWhitespace(text[from:to])
}
