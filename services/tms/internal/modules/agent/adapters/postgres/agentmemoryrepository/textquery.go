package agentmemoryrepository

import (
	"strings"
	"unicode/utf8"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	maxQueryWords    = 16
	minPrefixRunes   = 2
	minTermRunes     = 3
	maxFallbackTerms = 8
	websearchConfig  = "websearch_to_tsquery('simple', ?)"
	tsqueryConfig    = "to_tsquery('simple', ?)"
)

type textMatch struct {
	tsquery string
	args    []any
}

type textQuery struct {
	raw       string
	words     []string
	prefixes  []string
	terms     []string
	operators bool
}

// newTextQuery reads a recall query. terms are its meaningful words, the
// question's own words left out; the looser pass needs a memory to share two
// of them, so "the" and "for" never match a memory on their own.
func newTextQuery(query string, terms []string) textQuery {
	raw := strings.TrimSpace(query)
	words := sliceutils.Dedupe(stringutils.SearchWords(raw))
	if len(words) > maxQueryWords {
		words = words[:maxQueryWords]
	}

	prefixes := make([]string, 0, len(words))
	for _, word := range words {
		if utf8.RuneCountInString(word) >= minPrefixRunes {
			prefixes = append(prefixes, word+":*")
		}
	}

	return textQuery{
		raw:       raw,
		words:     words,
		prefixes:  prefixes,
		terms:     fallbackTerms(terms),
		operators: hasWebsearchOperators(raw),
	}
}

func fallbackTerms(terms []string) []string {
	kept := make([]string, 0, min(len(terms), maxFallbackTerms))
	for _, term := range sliceutils.Dedupe(terms) {
		words := stringutils.SearchWords(term)
		if len(words) != 1 || words[0] != term || utf8.RuneCountInString(term) < minTermRunes {
			continue
		}
		kept = append(kept, term+":*")
		if len(kept) == maxFallbackTerms {
			break
		}
	}

	return kept
}

func (m *textMatch) matches(vector keel.TSVectorCol) keel.Pred {
	if m == nil {
		return nil
	}

	return keel.Raw("? @@ "+m.tsquery, append([]any{vector}, m.args...)...)
}

func (m *textMatch) rank(vector keel.TSVectorCol) keel.Expr {
	return keel.Raw("ts_rank_cd(?, "+m.tsquery+")", append([]any{vector}, m.args...)...)
}

func (q textQuery) empty() bool { return q.raw == "" }

func (q textQuery) unmatchable() bool { return len(q.words) == 0 }

func (q textQuery) primary() *textMatch {
	if q.operators || len(q.prefixes) == 0 {
		return &textMatch{tsquery: websearchConfig, args: []any{q.raw}}
	}

	return &textMatch{
		tsquery: "(" + websearchConfig + " || " + tsqueryConfig + ")",
		args:    []any{q.raw, strings.Join(q.prefixes, " & ")},
	}
}

// fallback is the pass tried when no memory matches every word: any memory
// sharing two of the query's meaningful words, so a question still finds the
// memories that share most of it without one common word matching them all.
func (q textQuery) fallback() *textMatch {
	if q.operators || len(q.terms) < 2 {
		return nil
	}

	pairs := make([]string, 0, len(q.terms)*(len(q.terms)-1)/2)
	for i := range q.terms {
		for j := i + 1; j < len(q.terms); j++ {
			pairs = append(pairs, "("+q.terms[i]+" & "+q.terms[j]+")")
		}
	}

	return &textMatch{tsquery: tsqueryConfig, args: []any{strings.Join(pairs, " | ")}}
}

func hasWebsearchOperators(query string) bool {
	if strings.Contains(query, `"`) {
		return true
	}
	for field := range strings.FieldsSeq(query) {
		if strings.EqualFold(field, "or") {
			return true
		}
		if len(field) > 1 && strings.HasPrefix(field, "-") {
			return true
		}
	}

	return false
}
