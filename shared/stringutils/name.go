package stringutils

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxInitials = 2

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

func Initials(name string) string {
	words := strings.FieldsFunc(name, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	var out strings.Builder
	for idx, word := range words {
		if idx == maxInitials {
			break
		}
		first, _ := utf8.DecodeRuneInString(word)
		out.WriteRune(unicode.ToUpper(first))
	}

	return out.String()
}
