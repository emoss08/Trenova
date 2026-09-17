package stringutils

import "strings"

func NeutralizeCloseTag(text, closeTag string) string {
	if closeTag == "" || !strings.Contains(text, closeTag) {
		return text
	}

	escaped := "<\\/" + strings.TrimPrefix(closeTag, "</")

	return strings.ReplaceAll(text, closeTag, escaped)
}
