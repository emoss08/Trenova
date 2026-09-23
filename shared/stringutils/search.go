package stringutils

import (
	"strings"
	"unicode"
)

func ContainsAny(corpus string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(corpus, needle) {
			return true
		}
	}
	return false
}

// SearchWords splits text into lower-case words. Anything that is not a
// letter or a digit separates words, so "on-time" and "on time" are the same
// two words and a key such as "stop-dwell-and-detention" is four.
func SearchWords(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

// MatchesWordPrefixes reports whether every word begins some word of the
// texts, so "deliver" finds "delivery" and "on" does not find "operations".
// No words matches everything.
func MatchesWordPrefixes(words []string, texts ...string) bool {
	if len(words) == 0 {
		return true
	}

	var corpus []string
	for _, text := range texts {
		corpus = append(corpus, SearchWords(text)...)
	}
	for _, word := range words {
		if !hasWordWithPrefix(corpus, word) {
			return false
		}
	}

	return true
}

func hasWordWithPrefix(corpus []string, prefix string) bool {
	for _, candidate := range corpus {
		if strings.HasPrefix(candidate, prefix) {
			return true
		}
	}

	return false
}
