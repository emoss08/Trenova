package ediconnectiondraft

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
)

const (
	maxQuoteRunes      = 400
	minQuoteRunes      = 3
	confidenceFloor    = 0.8
	redactedMarker     = "[redacted]"
	redactedPEMMarker  = "[redacted key or certificate]"
	isaSecurityPadding = 10
)

var (
	pemBlockRe  = regexp.MustCompile(`(?s)-----BEGIN [A-Z0-9 ]+-----.*?-----END [A-Z0-9 ]+-----`)
	secretLabel = regexp.MustCompile(
		`(?i)\b(password|passphrase|pass\s*word|pwd|passcode|secret|private\s+key|api\s+key|access\s+key|token|shared\s+key)(s?)(\s*(?:is|:|=|-)?\s*)([^\s,;]+)`,
	)
	quoteFolds = strings.NewReplacer(
		"‘", "'", "’", "'", "“", `"`, "”", `"`,
		"–", "-", "—", "-", " ", " ", "•", " ",
	)
)

type pageIndex struct {
	pages      map[int]string
	order      []int
	normalized map[int]string
}

func newPageIndex(pages []ediports.SpecPage) *pageIndex {
	index := &pageIndex{
		pages:      make(map[int]string, len(pages)),
		order:      make([]int, 0, len(pages)),
		normalized: make(map[int]string, len(pages)),
	}
	for _, page := range pages {
		if _, seen := index.pages[page.Number]; seen {
			index.pages[page.Number] += "\n" + page.Text
			index.normalized[page.Number] = normalizeForMatch(index.pages[page.Number])
			continue
		}
		index.pages[page.Number] = page.Text
		index.order = append(index.order, page.Number)
		index.normalized[page.Number] = normalizeForMatch(page.Text)
	}
	return index
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

func (p *pageIndex) locate(citation ediports.DraftCitation) citationCheck {
	quote := normalizeForMatch(citation.Quote)
	if len([]rune(quote)) < minQuoteRunes {
		return citationCheck{problem: "The citation has no quote to check"}
	}
	if len([]rune(citation.Quote)) > maxQuoteRunes {
		return citationCheck{
			problem: fmt.Sprintf("The quote is longer than %d characters", maxQuoteRunes),
		}
	}
	if text, ok := p.normalized[citation.Page]; ok && strings.Contains(text, quote) {
		return citationCheck{page: citation.Page, ok: true}
	}
	for _, number := range p.order {
		if strings.Contains(p.normalized[number], quote) {
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

func valueInQuote(spec fieldSpec, value, quote string) bool {
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
	if !isWordLike(needle) {
		return strings.Contains(haystack, needle)
	}
	for start := 0; ; {
		index := strings.Index(haystack[start:], needle)
		if index < 0 {
			return false
		}
		index += start
		end := index + len(needle)
		if boundary(haystack, index-1) && boundary(haystack, end) {
			return true
		}
		start = index + 1
	}
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

func redact(value string) string {
	value = pemBlockRe.ReplaceAllString(value, redactedPEMMarker)
	value = secretLabel.ReplaceAllString(value, "$1$2$3"+redactedMarker)
	return redactInterchangeSecurity(value)
}

func redactInterchangeSecurity(value string) string {
	var builder strings.Builder
	builder.Grow(len(value))
	for {
		index := strings.Index(value, "ISA")
		if index < 0 || index+3 >= len(value) {
			builder.WriteString(value)
			return builder.String()
		}
		builder.WriteString(value[:index])
		separator := value[index+3]
		if unicode.IsLetter(rune(separator)) || unicode.IsDigit(rune(separator)) ||
			unicode.IsSpace(rune(separator)) {
			builder.WriteString(value[index : index+3])
			value = value[index+3:]
			continue
		}
		parts := strings.SplitN(value[index:], string(separator), 6)
		if len(parts) < 6 {
			builder.WriteString(value[index:])
			return builder.String()
		}
		parts[2] = maskElement(parts[1], parts[2])
		parts[4] = maskElement(parts[3], parts[4])
		builder.WriteString(strings.Join(parts[:5], string(separator)))
		builder.WriteByte(separator)
		value = parts[5]
	}
}

func maskElement(qualifier, element string) string {
	if strings.TrimSpace(qualifier) == "00" || strings.TrimSpace(element) == "" {
		return element
	}
	return strings.Repeat("*", max(len(element), isaSecurityPadding))
}

func clampQuote(value string) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= maxQuoteRunes {
		return string(runes)
	}
	return string(runes[:maxQuoteRunes]) + "…"
}

func documentSource(page int, citation ediports.DraftCitation, note string) ediports.DraftSource {
	return ediports.DraftSource{
		Kind:    ediports.SourceKindDocument,
		Page:    page,
		Quote:   clampQuote(redact(citation.Quote)),
		Section: strings.TrimSpace(citation.Section),
		Note:    note,
	}
}
