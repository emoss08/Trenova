package stringutils

import (
	"strings"
	"unicode"
)

// Slugify turns text into a file- and URL-safe name: lower case, letters and
// digits kept, every other run collapsed to one hyphen, cut to maxLen without
// ending on a hyphen. Empty input, or input with nothing to keep, yields "".
func Slugify(text string, maxLen int) string {
	var builder strings.Builder
	builder.Grow(len(text))
	pendingHyphen := false

	for _, r := range strings.ToLower(text) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if pendingHyphen && builder.Len() > 0 {
				builder.WriteByte('-')
			}
			pendingHyphen = false
			builder.WriteRune(r)
			continue
		}
		pendingHyphen = true
	}

	slug := builder.String()
	if maxLen > 0 && len(slug) > maxLen {
		slug = strings.TrimRight(slug[:maxLen], "-")
	}

	return slug
}
