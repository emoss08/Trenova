package stringutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
