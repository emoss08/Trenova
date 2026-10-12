package agentmemoryrepository

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTextQuery_MatchesTheWholeQueryOrEveryWordAsAPrefix(t *testing.T) {
	t.Parallel()

	query := newTextQuery("  Acme dock-hours ")
	require.False(t, query.empty())
	require.False(t, query.unmatchable())

	primary := query.primary()
	assert.Equal(t,
		"(websearch_to_tsquery('simple', ?) || to_tsquery('simple', ?))",
		primary.tsquery,
	)
	assert.Equal(t, []any{"Acme dock-hours", "acme:* & dock:* & hours:*"}, primary.args)

	fallback := query.fallback()
	require.NotNil(t, fallback, "a question of several words falls back to any of them")
	assert.Equal(t, "to_tsquery('simple', ?)", fallback.tsquery)
	assert.Equal(t, []any{"acme:* | dock:* | hours:*"}, fallback.args)
}

func TestTextQuery_KeepsWebsearchOperatorsWhole(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{`"dock 9"`, "acme -friday", "acme OR beta", "acme or beta"} {
		query := newTextQuery(raw)
		primary := query.primary()
		assert.Equal(t, "websearch_to_tsquery('simple', ?)", primary.tsquery, raw)
		assert.Equal(t, []any{raw}, primary.args, raw)
		assert.Nil(t, query.fallback(), "an exclusion is never undone by a looser pass: %s", raw)
	}
}

func TestTextQuery_OneWordHasNoLooserPass(t *testing.T) {
	t.Parallel()

	query := newTextQuery("deliver")
	assert.Equal(t, []any{"deliver", "deliver:*"}, query.primary().args,
		"deliver finds delivery through its prefix")
	assert.Nil(t, query.fallback())
}

func TestTextQuery_SingleLettersAreNotPrefixes(t *testing.T) {
	t.Parallel()

	query := newTextQuery("a b")
	assert.Equal(t, "websearch_to_tsquery('simple', ?)", query.primary().tsquery,
		"a one-letter prefix would match nearly every memory")
	assert.Nil(t, query.fallback())
}

func TestTextQuery_PunctuationAloneMatchesNothing(t *testing.T) {
	t.Parallel()

	query := newTextQuery("?!,")
	assert.False(t, query.empty())
	assert.True(t, query.unmatchable())
	assert.True(t, newTextQuery("   ").empty())
}

func TestTextQuery_BoundsTheWordsAndDropsRepeats(t *testing.T) {
	t.Parallel()

	query := newTextQuery(strings.Repeat("acme ", 3) + strings.Repeat("word ", 1) +
		"a1 a2 a3 a4 a5 a6 a7 a8 a9 b1 b2 b3 b4 b5 b6 b7 b8")
	assert.Len(t, query.words, maxQueryWords)
	assert.Equal(t, "acme", query.words[0])
	assert.Equal(t, "word", query.words[1], "repeats are dropped and the order typed is kept")
}
