package stringutils

import "strings"

func StripNUL(s string) string {
	if strings.IndexByte(s, 0) < 0 {
		return s
	}

	return strings.ReplaceAll(s, "\x00", "")
}
