package retrievalservice

import (
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	MaxSnippetChars = 600
	minTermRunes    = 3
	ellipsis        = "…"
)

var queryStopWords = []string{
	"about", "all", "and", "any", "are", "can", "did", "does", "for", "from", "had", "has",
	"have", "how", "into", "its", "not", "our", "that", "the", "their", "them", "then",
	"there", "these", "they", "this", "was", "were", "what", "when", "where", "which",
	"who", "why", "will", "with", "you", "your",
}

func QueryTerms(query string) []string {
	words := stringutils.SearchWords(query)
	terms := make([]string, 0, len(words))
	for _, word := range words {
		if utf8.RuneCountInString(word) < minTermRunes ||
			slices.Contains(queryStopWords, word) ||
			slices.Contains(terms, word) {
			continue
		}
		terms = append(terms, word)
	}

	return terms
}

func termHits(text string, terms []string) int {
	lower := strings.ToLower(text)
	hits := 0
	for _, term := range terms {
		hits += strings.Count(lower, term)
	}

	return hits
}

func BestChunk(chunks []Chunk, terms []string, preferred int) (int, bool) {
	best, bestHits := -1, 0
	for idx, chunk := range chunks {
		if chunk.Body == "" {
			continue
		}
		if hits := termHits(chunk.Body, terms); hits > bestHits {
			best, bestHits = idx, hits
		}
	}

	switch {
	case best >= 0:
		return best, true
	case preferred >= 0 && preferred < len(chunks) && chunks[preferred].Body != "":
		return preferred, true
	}

	for idx, chunk := range chunks {
		if chunk.Body != "" {
			return idx, true
		}
	}

	return 0, false
}

func Snippet(body string, terms []string) string {
	text := stringutils.CollapseWhitespace(body)
	runes := []rune(text)
	if len(runes) <= MaxSnippetChars {
		return text
	}

	start := 0
	if at := firstTermRune(strings.ToLower(text), terms); at > 0 {
		start = max(0, at-MaxSnippetChars/3)
	}

	limit := MaxSnippetChars - 2*utf8.RuneCountInString(ellipsis)
	start = min(start, len(runes))
	end := min(start+limit, len(runes))
	start = max(0, end-limit)

	snippet := strings.TrimSpace(string(runes[start:end]))
	if start > 0 {
		snippet = ellipsis + snippet
	}
	if end < len(runes) {
		snippet += ellipsis
	}

	return snippet
}

func firstTermRune(lower string, terms []string) int {
	first := -1
	for _, term := range terms {
		at := strings.Index(lower, term)
		if at < 0 {
			continue
		}
		runeAt := utf8.RuneCountInString(lower[:at])
		if first < 0 || runeAt < first {
			first = runeAt
		}
	}

	return first
}
