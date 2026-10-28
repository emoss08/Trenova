package ediconnectiondraft

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/emoss08/trenova/internal/modules/edi/domain/x12field"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediconnectiondraftports"
)

const (
	maxQuoteRunes   = 400
	minQuoteRunes   = 3
	confidenceFloor = 0.8
)

var (
	quoteFolds = strings.NewReplacer(
		"‘", "'", "’", "'", "“", `"`, "”", `"`,
		"–", "-", "—", "-", " ", " ", "•", " ",
	)
)

type pageIndex struct {
	pages      map[int]string
	order      []int
	normalized map[int]string
	redacted   map[int]string
}

func newPageIndex(pages []ediconnectiondraftports.SpecPage) *pageIndex {
	index := &pageIndex{
		pages:      make(map[int]string, len(pages)),
		order:      make([]int, 0, len(pages)),
		normalized: make(map[int]string, len(pages)),
		redacted:   make(map[int]string, len(pages)),
	}
	for _, page := range pages {
		if _, seen := index.pages[page.Number]; seen {
			index.pages[page.Number] += "\n" + page.Text
		} else {
			index.pages[page.Number] = page.Text
			index.order = append(index.order, page.Number)
		}
		text := index.pages[page.Number]
		index.normalized[page.Number] = normalizeForMatch(text)
		index.redacted[page.Number] = normalizeForMatch(x12field.RedactSecrets(text))
	}
	return index
}

func (p *pageIndex) onPage(number int, quote string) bool {
	if text, ok := p.normalized[number]; ok && strings.Contains(text, quote) {
		return true
	}
	return strings.Contains(p.redacted[number], quote)
}

func normalizeForMatch(value string) string {
	value = quoteFolds.Replace(value)
	var builder strings.Builder
	builder.Grow(len(value))
	space := false
	for _, r := range value {
		if unicode.IsSpace(r) {
			space = true
			continue
		}
		if space && builder.Len() > 0 {
			builder.WriteByte(' ')
		}
		space = false
		builder.WriteRune(unicode.ToLower(r))
	}
	return builder.String()
}

type citationCheck struct {
	page    int
	ok      bool
	problem string
}

func (p *pageIndex) locate(citation ediconnectiondraftports.DraftCitation) citationCheck {
	quote := normalizeForMatch(citation.Quote)
	if len([]rune(quote)) < minQuoteRunes {
		return citationCheck{problem: "The citation has no quote to check"}
	}
	if len([]rune(citation.Quote)) > maxQuoteRunes {
		return citationCheck{
			problem: fmt.Sprintf("The quote is longer than %d characters", maxQuoteRunes),
		}
	}
	if p.onPage(citation.Page, quote) {
		return citationCheck{page: citation.Page, ok: true}
	}
	for _, number := range p.order {
		if p.onPage(number, quote) {
			return citationCheck{
				page: number,
				ok:   true,
				problem: fmt.Sprintf(
					"Cited as page %d; the quote is on page %d",
					citation.Page,
					number,
				),
			}
		}
	}
	if _, ok := p.normalized[citation.Page]; !ok {
		return citationCheck{
			problem: fmt.Sprintf("The document has no page %d", citation.Page),
		}
	}
	return citationCheck{
		problem: fmt.Sprintf("The quote does not appear on page %d", citation.Page),
	}
}

func valueInQuote(spec *fieldSpec, value, quote string) bool {
	if spec.matches != nil && spec.matches(value, quote) {
		return true
	}
	normalizedQuote := normalizeForMatch(quote)
	if value != "\n" {
		if needle := normalizeForMatch(value); needle != "" &&
			containsToken(normalizedQuote, needle) {
			return true
		}
	}
	for _, phrase := range spec.spelled[value] {
		if strings.Contains(normalizedQuote, normalizeForMatch(phrase)) {
			return true
		}
	}
	return false
}

func containsToken(haystack, needle string) bool {
	return tokenIndex(haystack, needle, 0) >= 0
}

func tokenIndex(haystack, needle string, from int) int {
	wordLike := isWordLike(needle)
	for start := from; start <= len(haystack); {
		index := strings.Index(haystack[start:], needle)
		if index < 0 {
			return -1
		}
		index += start
		if !wordLike ||
			(boundary(haystack, index-1) && boundary(haystack, index+len(needle))) {
			return index
		}
		start = index + 1
	}
	return -1
}

func isWordLike(value string) bool {
	for _, r := range value {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func boundary(value string, index int) bool {
	if index < 0 || index >= len(value) {
		return true
	}
	r := rune(value[index])
	return !unicode.IsLetter(r) && !unicode.IsDigit(r)
}

func clampQuote(value string) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= maxQuoteRunes {
		return string(runes)
	}
	return string(runes[:maxQuoteRunes]) + "…"
}

func documentSource(
	page int,
	citation ediconnectiondraftports.DraftCitation,
	note string,
) ediconnectiondraftports.DraftSource {
	return ediconnectiondraftports.DraftSource{
		Kind:    ediconnectiondraftports.SourceKindDocument,
		Page:    page,
		Quote:   clampQuote(x12field.RedactSecrets(citation.Quote)),
		Section: strings.TrimSpace(citation.Section),
		Note:    note,
	}
}
