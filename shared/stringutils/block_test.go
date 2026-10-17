package stringutils

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTruncateAtBlock(t *testing.T) {
	t.Parallel()

	table := "## Charges\n\n| A | B |\n| --- | --- |\n| 1 | 2 |\n| 3 | 4 |"
	text := "Intro paragraph.\n\n" + table

	assert.Equal(t, text, TruncateAtBlock(text, len(text)))
	assert.Equal(t, "Intro paragraph.\n\n## Charges", TruncateAtBlock(text, 40))
	assert.Equal(
		t,
		"Intro paragraph.\n\n## Charges\n\n| A | B |\n| --- | --- |\n| 1 | 2 |",
		TruncateAtBlock(text, len(text)-3),
		"a table too long for the window is cut between rows, never inside one",
	)
	assert.Equal(t, "abc", TruncateAtBlock("abcdef", 3))
	assert.Equal(t, "é", TruncateAtBlock("éé", 3), "never splits a rune")
	assert.Empty(t, TruncateAtBlock("abc", 0))
}

func TestTruncateAtBlockKeepsMostOfTheWindow(t *testing.T) {
	t.Parallel()

	paragraph := strings.Repeat("Pickup at ACME dock 4 before noon. ", 120)
	header := "<!-- page header -->\nACME\n"

	tests := []struct {
		name     string
		value    string
		maxBytes int
		prefix   string
	}{
		{
			name:     "a first paragraph longer than the window",
			value:    paragraph,
			maxBytes: 2500,
			prefix:   "Pickup at ACME",
		},
		{
			name:     "a page header then a long paragraph, route budget",
			value:    header + paragraph,
			maxBytes: 800,
			prefix:   header,
		},
		{
			name:     "a page header then a long paragraph, extract budget",
			value:    header + paragraph,
			maxBytes: 2500,
			prefix:   header,
		},
		{
			name:     "multi-byte runes at the cut",
			value:    "Fracht: " + strings.Repeat("é€漢", 400),
			maxBytes: 801,
			prefix:   "Fracht: ",
		},
		{
			name:     "a window smaller than the first line",
			value:    "Load 48213 picks up at ACME\nsecond line",
			maxBytes: 10,
			prefix:   "Load 48213",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := TruncateAtBlock(tt.value, tt.maxBytes)

			require.True(t, utf8.ValidString(got), "never splits a rune")
			assert.LessOrEqual(t, len(got), tt.maxBytes)
			assert.GreaterOrEqual(
				t,
				len(got),
				tt.maxBytes/2,
				"a cut keeps at least half the window rather than only the page header",
			)
			assert.True(t, strings.HasPrefix(got, strings.TrimSpace(tt.prefix)), got)
		})
	}
}

func TestTruncateAtBlockPrefersALineBreakInTheBackHalf(t *testing.T) {
	t.Parallel()

	value := strings.Repeat("a", 60) + "\n" + strings.Repeat("b", 60)

	assert.Equal(t, strings.Repeat("a", 60), TruncateAtBlock(value, 100))
}
