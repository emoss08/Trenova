package stringutils

import "strings"

func ContainsAny(corpus string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(corpus, needle) {
			return true
		}
	}
	return false
}
