package mdtable

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFind_ReadsATableWithItsLeadAndBounds(t *testing.T) {
	t.Parallel()

	text := "Here are the late loads:\n" +
		"| Pro | Status |\n" +
		"| --- | :---: |\n" +
		"| **A1** | Late |\n" +
		"| [B2](artifact:x) | Late |\n" +
		"\nThat is all."

	tables := Find(text)

	require.Len(t, tables, 1)
	table := tables[0]
	assert.Equal(t, "Here are the late loads:", table.Lead)
	assert.Equal(t, []string{"Pro", "Status"}, table.Header)
	assert.Equal(t, [][]string{{"**A1**", "Late"}, {"[B2](artifact:x)", "Late"}}, table.Rows)
	assert.Equal(t,
		"| Pro | Status |\n| --- | :---: |\n| **A1** | Late |\n| [B2](artifact:x) | Late |\n",
		text[table.Start:table.End])
	assert.Equal(t, "a1", Plain(table.Rows[0][0]))
	assert.Equal(t, "b2", Plain(table.Rows[1][0]))
}

func TestFind_LeavesCodeAndLoosePipesAlone(t *testing.T) {
	t.Parallel()

	text := "```\n| a | b |\n| --- | --- |\n| 1 | 2 |\n```\n" +
		"Either A | B, not both.\nNo table here."

	assert.Empty(t, Find(text))
}

func TestFind_ATableThatEndsTheText(t *testing.T) {
	t.Parallel()

	text := "| a | b |\n|---|---|\n| 1 | 2 |"

	tables := Find(text)

	require.Len(t, tables, 1)
	assert.Equal(t, len(text), tables[0].End)
	assert.Empty(t, tables[0].Lead)
}
