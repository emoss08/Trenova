package stringutils_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
A reply that has fallen into a loop is caught, and cut where the loop begins.

The Homepage Widget Builder's last reply ended "time-time-time-…" for hundreds
of characters. A small model that starts repeating does not stop on its own,
and everything after the loop begins is noise a person has to scroll past.
*/
func TestDegenerateTail_CatchesAReplyStuckOnOneFragment(t *testing.T) {
	t.Parallel()

	prefix := "Your dashboard has three tiles: on-"
	text := prefix + strings.Repeat("time-", 40)

	cut, ok := stringutils.DegenerateTail(text)

	require.True(t, ok)
	assert.Equal(t, prefix, text[:cut])
}

func TestDegenerateTail_CatchesARepeatedSentence(t *testing.T) {
	t.Parallel()

	prefix := "Here is the plan. "
	text := prefix + strings.Repeat("I will check the report and get back to you. ", 6)

	cut, ok := stringutils.DegenerateTail(text)

	require.True(t, ok)
	assert.Equal(t, prefix, text[:cut])
}

// What a real answer repeats is not a loop: a table rule, a column of
// zeros, a short emphatic repetition, a list with a shared prefix.
func TestDegenerateTail_LeavesOrdinaryRepetitionAlone(t *testing.T) {
	t.Parallel()

	for name, text := range map[string]string{
		"table rule":  "| Customer | Loads |\n|" + strings.Repeat("---|", 30) + "\n| Acme | 4 |",
		"zeros":       "| " + strings.Repeat("0 | ", 30),
		"emphasis":    "No, no, no — the load is still at the shipper.",
		"list":        strings.Repeat("- Load 1234 delivered on time\n", 3),
		"horizontal":  "Summary\n" + strings.Repeat("=", 80),
		"short reply": "OK.",
		"empty":       "",
	} {
		_, ok := stringutils.DegenerateTail(text)
		assert.Falsef(t, ok, "%s read as a loop", name)
	}
}

func TestDegenerateTail_OnlyLooksAtTheEnd(t *testing.T) {
	t.Parallel()

	text := strings.Repeat("time-", 40) + " and then the answer went on normally to the end."

	_, ok := stringutils.DegenerateTail(text)

	assert.False(t, ok, "a loop the model already left is not the tail")
}
