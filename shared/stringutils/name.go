package stringutils

import "strings"

func FirstName(fullName string) string {
	name := strings.TrimSpace(fullName)
	if name == "" {
		return ""
	}
	if space := strings.IndexFunc(name, isNameSeparator); space > 0 {
		return name[:space]
	}
	return name
}

func isNameSeparator(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n'
}
