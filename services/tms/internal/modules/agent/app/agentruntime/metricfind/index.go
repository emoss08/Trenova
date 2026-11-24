package metricfind

import (
	"slices"
	"strings"
	"sync"
	"unicode"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentsearch"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardregistry"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
)

const (
	nameWeight    = 4
	figureWeight  = 3
	wordsWeight   = 2
	phraseBonus   = 4
	namedOutright = 100
	minScore      = 8
	minStemRunes  = 5
	cutoffDivisor = 3
)

var calendarSpans = map[string]struct{}{
	"week": {}, "month": {}, "quarter": {}, "year": {},
}

type entry struct {
	metric  boardcatalog.Metric
	weights map[string]int
	phrases [][]string
}

type scored struct {
	entry *entry
	score int
	order int
}

var index = sync.OnceValue(func() []entry {
	metrics := boardregistry.Catalog().Metrics()
	entries := make([]entry, 0, len(metrics))
	for idx := range metrics {
		entries = append(entries, indexed(&metrics[idx]))
	}

	return entries
})

func indexed(metric *boardcatalog.Metric) entry {
	out := entry{metric: *metric, weights: make(map[string]int, 32)}
	weigh := func(text string, weight int) {
		for token := range agentsearch.TokenSet(text) {
			out.weights[token] = max(out.weights[token], weight)
		}
	}
	describe := func(text string, weight int) {
		for token := range agentsearch.TokenSet(text) {
			if _, span := calendarSpans[agentsearch.Singularize(token)]; span {
				continue
			}
			out.weights[token] = max(out.weights[token], weight)
		}
	}
	for _, name := range slices.Concat([]string{metric.Name}, metric.Aliases) {
		split := agentsearch.SplitCamel(name)
		weigh(split, nameWeight)
		phrase := agentsearch.Tokens(split)
		if name == metric.Name || len(phrase) > 1 {
			out.phrases = append(out.phrases, phrase)
		}
	}
	for _, figure := range metric.Figures {
		weigh(agentsearch.SplitCamel(figure.Name), figureWeight)
		describe(figure.Label, wordsWeight)
	}
	describe(metric.Words, wordsWeight)

	return out
}

func Matching(need string, limit int) []boardcatalog.Metric {
	if limit <= 0 || strings.TrimSpace(need) == "" {
		return nil
	}
	terms := agentsearch.Terms(need)
	named := namedIn(need)
	entries := index()
	ranked := make([]scored, 0, len(entries))
	for idx := range entries {
		item := &entries[idx]
		score, strong := item.score(terms)
		if slices.Contains(named, item.metric.Name) {
			score, strong = score+namedOutright, true
		}
		if score >= minScore || (strong && score > 0) {
			ranked = append(ranked, scored{entry: item, score: score, order: idx})
		}
	}
	if len(ranked) == 0 {
		return nil
	}
	slices.SortStableFunc(ranked, func(a, b scored) int {
		if a.score != b.score {
			return b.score - a.score
		}

		return a.order - b.order
	})
	floor := ranked[0].score / cutoffDivisor
	out := make([]boardcatalog.Metric, 0, min(limit, len(ranked)))
	for _, candidate := range ranked {
		if candidate.score < floor || len(out) == limit {
			break
		}
		out = append(out, candidate.entry.metric)
	}

	return out
}

func Mentions(text string) bool {
	return len(Matching(text, 1)) > 0
}

func (e *entry) score(terms map[string]struct{}) (int, bool) {
	total := 0
	for term := range terms {
		if singular := agentsearch.Singularize(term); singular != term {
			if _, folded := terms[singular]; folded {
				continue
			}
		}
		total += e.weightOf(term)
	}
	strong := false
	for _, phrase := range e.phrases {
		if len(phrase) == 0 {
			continue
		}
		whole := true
		for _, token := range phrase {
			if _, ok := terms[token]; !ok {
				if _, ok = terms[agentsearch.Singularize(token)]; !ok {
					whole = false
					break
				}
			}
		}
		if whole {
			strong = true
			break
		}
	}
	if strong {
		total += phraseBonus
	}

	return total, strong
}

func (e *entry) weightOf(term string) int {
	if weight, ok := e.weights[term]; ok {
		return weight
	}
	if len([]rune(term)) < minStemRunes {
		return 0
	}
	best := 0
	for token, weight := range e.weights {
		if len([]rune(token)) < minStemRunes {
			continue
		}
		if strings.HasPrefix(term, token) || strings.HasPrefix(token, term) {
			best = max(best, weight-1)
		}
	}

	return best
}

func namedIn(need string) []string {
	words := strings.FieldsFunc(need, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	named := make([]string, 0, 2)
	metrics := boardregistry.Catalog().Metrics()
	for idx := range metrics {
		metric := &metrics[idx]
		names := slices.Concat([]string{metric.Name}, metric.Aliases)
		for _, word := range words {
			if slices.ContainsFunc(names, func(name string) bool {
				return strings.EqualFold(name, word) &&
					(name == metric.Name || strings.ContainsFunc(name, unicode.IsUpper))
			}) {
				named = append(named, metric.Name)
				break
			}
		}
	}

	return named
}
