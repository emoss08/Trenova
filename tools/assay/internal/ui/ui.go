// Package ui is the single styling layer for everything assay prints to a
// human. Commands describe output in semantic terms — good, warn, bad, muted,
// a badge, a progress bar, a table — and the active styler decides how that
// looks. The default build renders through lipgloss; building with the
// assay_plain tag swaps in a stdlib-only styler with the same semantics, so
// the charm dependencies vanish from the binary entirely.
//
// Both stylers are constructed with an explicit enabled flag derived from
// --no-color and TTY detection, and neither queries the terminal: color
// support is decided by the caller, never sniffed with escape-sequence
// round-trips that could stall startup.
package ui

import (
	"fmt"
	"strings"
)

type Tone int

const (
	ToneGood Tone = iota
	ToneWarn
	ToneBad
	ToneMuted
	ToneAccent
)

type styler interface {
	paint(tone Tone, s string) string
	bold(s string) string
	badge(tone Tone, text string) string
	table(headers []string, rows [][]string) string
}

type Painter struct {
	s       styler
	enabled bool
}

func New(useColor bool) Painter {
	return Painter{s: newStyler(useColor), enabled: useColor}
}

func (p Painter) Enabled() bool { return p.enabled }

func (p Painter) Good(s string) string   { return p.s.paint(ToneGood, s) }
func (p Painter) Warn(s string) string   { return p.s.paint(ToneWarn, s) }
func (p Painter) Bad(s string) string    { return p.s.paint(ToneBad, s) }
func (p Painter) Muted(s string) string  { return p.s.paint(ToneMuted, s) }
func (p Painter) Accent(s string) string { return p.s.paint(ToneAccent, s) }
func (p Painter) Bold(s string) string   { return p.s.bold(s) }

func (p Painter) Goodf(format string, args ...any) string {
	return p.Good(fmt.Sprintf(format, args...))
}

func (p Painter) Mutedf(format string, args ...any) string {
	return p.Muted(fmt.Sprintf(format, args...))
}

// Badge renders a short, shouty label — SURVIVED, KILLED, FLAKY — that reads
// as a chip on capable terminals and as a bracketed tag everywhere else.
func (p Painter) Badge(tone Tone, text string) string { return p.s.badge(tone, text) }

// Bar renders a progress bar of the given cell width. The fraction is clamped;
// the bar itself is plain unicode so both stylers share it, with the filled
// span tinted when color is on.
func (p Painter) Bar(fraction float64, width int) string {
	if width <= 0 {
		return ""
	}
	if fraction < 0 {
		fraction = 0
	}
	if fraction > 1 {
		fraction = 1
	}

	filled := int(fraction*float64(width) + 0.5)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	if filled == 0 {
		return p.Muted(bar)
	}

	return p.s.paint(ToneAccent, bar[:filled*len("█")]) + p.Muted(bar[filled*len("█"):])
}

// Rule renders a soft horizontal divider.
func (p Painter) Rule(width int) string {
	if width <= 0 {
		return ""
	}

	return p.Muted(strings.Repeat("─", width))
}

// Table renders headers and rows as an aligned table. The charm build draws
// real borders; the plain build emits aligned columns that survive grep.
func (p Painter) Table(headers []string, rows [][]string) string {
	return p.s.table(headers, rows)
}
