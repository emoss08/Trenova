//go:build assay_plain

package ui

import (
	"strings"
	"unicode/utf8"
)

// plainStyler is the stdlib-only styler behind the assay_plain build tag. It
// keeps the same semantic palette using raw SGR sequences and renders tables
// as aligned columns, so a charm-free build loses polish, not information.
type plainStyler struct {
	enabled bool
}

func newStyler(useColor bool) styler {
	return &plainStyler{enabled: useColor}
}

var plainCodes = map[Tone]string{
	ToneGood:   "\033[32m",
	ToneWarn:   "\033[33m",
	ToneBad:    "\033[31m",
	ToneMuted:  "\033[2m",
	ToneAccent: "\033[36m",
}

const plainReset = "\033[0m"

func (p *plainStyler) paint(tone Tone, s string) string {
	if !p.enabled || s == "" {
		return s
	}

	return plainCodes[tone] + s + plainReset
}

func (p *plainStyler) bold(s string) string {
	if !p.enabled || s == "" {
		return s
	}

	return "\033[1m" + s + plainReset
}

func (p *plainStyler) badge(tone Tone, text string) string {
	if !p.enabled {
		return "[" + text + "]"
	}

	return plainCodes[tone] + "\033[1;7m " + text + " " + plainReset
}

func (p *plainStyler) table(headers []string, rows [][]string) string {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = utf8.RuneCountInString(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && utf8.RuneCountInString(cell) > widths[i] {
				widths[i] = utf8.RuneCountInString(cell)
			}
		}
	}

	var b strings.Builder
	writeRow := func(cells []string, decorate func(string) string) {
		for i, cell := range cells {
			if i > 0 {
				b.WriteString("  ")
			}
			pad := 0
			if i < len(widths) {
				pad = widths[i] - utf8.RuneCountInString(cell)
			}
			b.WriteString(decorate(cell))
			if i < len(cells)-1 {
				b.WriteString(strings.Repeat(" ", pad))
			}
		}
		b.WriteString("\n")
	}

	writeRow(headers, p.bold)
	for _, row := range rows {
		writeRow(row, func(s string) string { return s })
	}

	return strings.TrimSuffix(b.String(), "\n")
}
