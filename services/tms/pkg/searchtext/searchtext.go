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
	query = VectorQuery(query)

	return col.Match(query, Options(query)...)
}

func RankVector(col keel.TSVectorCol, query string) keel.Expr {
	query = VectorQuery(query)

	return col.Rank(query, Options(query)...)
}

// VectorQuery is the websearch query a text search sends: the query as
// written, or any of the identifiers it lists.
func VectorQuery(query string) string {
	if terms := IdentifierTerms(query); terms != nil {
		return strings.Join(terms, " or ")
	}

	return query
}

// MaxIdentifierTerms bounds how many identifiers one search looks for.
const MaxIdentifierTerms = 50

const (
	minIdentifierLength = 3
	maxIdentifierLength = 64
)

// IdentifierTerms are the identifiers a query lists when it is nothing but two
// or more of them (pro numbers, BOLs, reference numbers) separated by spaces,
// commas, semicolons or pipes. A search finds a record matching any of them:
// the query as written matches only a record holding all of them, which is
// none. It is nil for any other query and past MaxIdentifierTerms.
func IdentifierTerms(query string) []string {
	if HasOperators(query) {
		return nil
	}
	tokens := strings.FieldsFunc(query, func(r rune) bool {
		return unicode.IsSpace(r) || r == ',' || r == ';' || r == '|'
	})
	if len(tokens) < 2 || len(tokens) > MaxIdentifierTerms {
		return nil
	}
	terms := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if !identifierLike(token) {
			return nil
		}
		if !slices.Contains(terms, token) {
			terms = append(terms, token)
		}
	}
	if len(terms) < 2 {
		return nil
	}

	return terms
}

func identifierLike(token string) bool {
	length := utf8.RuneCountInString(token)
	if length < minIdentifierLength || length > maxIdentifierLength {
		return false
	}
	digit := false
	for _, r := range token {
		switch {
		case unicode.IsDigit(r):
			digit = true
		case unicode.IsLetter(r), r == '-', r == '_', r == '.', r == '/', r == '#':
		default:
			return false
		}
	}

	return digit
}

func wordBreak(r rune) bool {
	return !unicode.IsLetter(r) && !unicode.IsDigit(r)
}
