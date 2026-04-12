package stringutils

import "unicode"

func NormalizeIdentifier(value string) string {
	buf := make([]rune, 0, len(value))
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			buf = append(buf, unicode.ToUpper(r))
		}
	}
	return string(buf)
}
