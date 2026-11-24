package stringutils

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func UpperFirst(text string) string {
	first, size := utf8.DecodeRuneInString(text)
	if size == 0 {
		return text
	}

	return string(unicode.ToUpper(first)) + text[size:]
}

func WholeWordsWithin(text string, most int) string {
	if utf8.RuneCountInString(text) <= most {
		return text
	}
	cut := TruncateRunes(text, most+1)
	if space := strings.LastIndexByte(cut, ' '); space > 0 {
		cut = cut[:space]
	}

	return strings.TrimRight(TruncateRunes(cut, most), " ,;:-")
}
