package stringutils

import "strings"

// WithArticle is a noun as a sentence introduces it: "a shipment", "an
// invoice". It goes by the first letter, which is right for the nouns the
// product names its records by.
func WithArticle(noun string) string {
	trimmed := strings.TrimSpace(noun)
	if trimmed == "" {
		return ""
	}
	switch trimmed[0] {
	case 'a', 'e', 'i', 'o', 'u', 'A', 'E', 'I', 'O', 'U':
		return "an " + trimmed
	default:
		return "a " + trimmed
	}
}
