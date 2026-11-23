package stringutils

import (
	"strings"
	"unicode"
)

func IndexWords(text, words string) int {
	if words == "" {
		return -1
	}
	for offset := 0; offset < len(text); {
		idx := strings.Index(text[offset:], words)
		if idx < 0 {
			return -1
		}
		start := offset + idx
		if wordBoundary(text, start-1) && wordBoundary(text, start+len(words)) {
			return start
		}
		offset = start + 1
	}

	return -1
}

func ContainsWords(text, words string) bool {
	return IndexWords(text, words) >= 0
}

func wordBoundary(text string, idx int) bool {
	if idx < 0 || idx >= len(text) {
		return true
	}
	r := rune(text[idx])

	return !unicode.IsLetter(r) && !unicode.IsDigit(r)
}
