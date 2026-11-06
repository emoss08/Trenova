package searchtext

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/emoss08/keel"
)

const (
	minPrefixLength    = 2
	fuzzyMinWordLength = 4
	fuzzyMaxWords      = 4
)

func Options(query string) []keel.SearchOption {
	modes := keel.SearchModes(keel.WebSearch)
	if !HasOperators(query) && len(query) >= minPrefixLength {
		modes = keel.SearchModes(keel.WebSearch, keel.Prefix)
	}

	return []keel.SearchOption{keel.SearchConfig("simple", "english"), modes}
}

func HasOperators(query string) bool {
	if strings.Contains(query, "\"") {
		return true
	}

	if strings.Contains(query, " OR ") || strings.Contains(query, " or ") {
		return true
	}

	for word := range strings.FieldsSeq(query) {
		if strings.HasPrefix(word, "-") && len(word) > 1 {
			return true
		}
	}

	return false
}

func FuzzyWords(query string) []string {
	if HasOperators(query) {
		return nil
	}

	words := make([]string, 0, fuzzyMaxWords)
	for _, word := range strings.FieldsFunc(strings.ToLower(query), wordBreak) {
		if len(words) == fuzzyMaxWords {
			break
		}
		if utf8.RuneCountInString(word) >= fuzzyMinWordLength && !slices.Contains(words, word) {
			words = append(words, word)
		}
	}

	return words
}

func MatchVector(col keel.TSVectorCol, query string) keel.Pred {
	return col.Match(query, Options(query)...)
}

func wordBreak(r rune) bool {
	return !unicode.IsLetter(r) && !unicode.IsDigit(r)
}
