//go:build !assay_plain

package ui

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

// charmStyler renders through lipgloss v2, whose styles are pure string
// transforms with no terminal interrogation — exactly the property that keeps
// startup instant. The ANSI 16-color palette is deliberate: those slots are
// themed by the user's terminal, so output follows their scheme instead of
// imposing hex colors that may clash with a light background. Color use is
// decided entirely by the enabled flag the caller derived from --no-color and
// TTY detection.
type charmStyler struct {
	enabled bool
	tones   map[Tone]lipgloss.Style
	chips   map[Tone]lipgloss.Style
	boldS   lipgloss.Style
	head    lipgloss.Style
	edge    lipgloss.Style
	cell    lipgloss.Style
}

func newStyler(useColor bool) styler {
	tone := func(code string) lipgloss.Style {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(code))
	}
	chip := func(code string) lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color(code)).
			Padding(0, 1).
			Bold(true)
	}

	return &charmStyler{
		enabled: useColor,
		tones: map[Tone]lipgloss.Style{
			ToneGood:   tone("2"),
			ToneWarn:   tone("3"),
			ToneBad:    tone("1"),
			ToneMuted:  lipgloss.NewStyle().Faint(true),
			ToneAccent: tone("6"),
		},
		chips: map[Tone]lipgloss.Style{
			ToneGood:   chip("2"),
			ToneWarn:   chip("3"),
			ToneBad:    chip("1"),
			ToneMuted:  chip("8"),
			ToneAccent: chip("6"),
		},
		boldS: lipgloss.NewStyle().Bold(true),
		head:  lipgloss.NewStyle().Bold(true).Padding(0, 1),
		edge:  lipgloss.NewStyle().Faint(true),
		cell:  lipgloss.NewStyle().Padding(0, 1),
	}
}

func (c *charmStyler) paint(tone Tone, s string) string {
	if !c.enabled {
		return s
	}

	return c.tones[tone].Render(s)
}

func (c *charmStyler) bold(s string) string {
	if !c.enabled {
		return s
	}

	return c.boldS.Render(s)
}

func (c *charmStyler) badge(tone Tone, text string) string {
	if !c.enabled {
		return "[" + text + "]"
	}

	return c.chips[tone].Render(text)
}

func (c *charmStyler) table(headers []string, rows [][]string) string {
	edge := c.edge
	head := c.head
	if !c.enabled {
		edge = lipgloss.NewStyle()
		head = lipgloss.NewStyle().Padding(0, 1)
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(edge).
		Headers(headers...).
		Rows(rows...).
		StyleFunc(func(row, _ int) lipgloss.Style {
			if row == table.HeaderRow {
				return head
			}

			return c.cell
		})

	return t.Render()
}
