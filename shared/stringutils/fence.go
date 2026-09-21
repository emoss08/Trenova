package stringutils

import "strings"

func NeutralizeCloseTag(text, closeTag string) string {
	if closeTag == "" || !strings.Contains(text, closeTag) {
		return text
	}

	escaped := "<\\/" + strings.TrimPrefix(closeTag, "</")

	return strings.ReplaceAll(text, closeTag, escaped)
}

// RestoreCloseTag undoes NeutralizeCloseTag, for reading the text back.
func RestoreCloseTag(text, closeTag string) string {
	if closeTag == "" {
		return text
	}

	escaped := "<\\/" + strings.TrimPrefix(closeTag, "</")
	if !strings.Contains(text, escaped) {
		return text
	}

	return strings.ReplaceAll(text, escaped, closeTag)
}
