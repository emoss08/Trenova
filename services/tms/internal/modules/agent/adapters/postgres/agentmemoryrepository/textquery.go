package agentmemoryrepository

import (
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	maxQueryWords   = 16
	minPrefixRunes  = 2
	websearchConfig = "websearch_to_tsquery('simple', ?)"
	tsqueryConfig   = "to_tsquery('simple', ?)"
)

type textMatch struct {
	tsquery string
	args    []any
}

type textQuery struct {
	raw       string
	words     []string
	prefixes  []string
	operators bool
}

func newTextQuery(query string) textQuery {
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
		operators: hasWebsearchOperators(raw),
	}
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

func (q textQuery) fallback() *textMatch {
	if q.operators || len(q.prefixes) < 2 {
		return nil
	}

	return &textMatch{tsquery: tsqueryConfig, args: []any{strings.Join(q.prefixes, " | ")}}
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
