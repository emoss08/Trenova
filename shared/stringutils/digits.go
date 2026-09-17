package stringutils

import "strings"

func DigitsOnly(value string) string {
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func CollapseUpper(value string) string {
	return strings.ToUpper(strings.Join(strings.Fields(value), " "))
}
