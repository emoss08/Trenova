package agentmemoryrepository

import (
	"strings"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTextQuery_MatchesTheWholeQueryOrEveryWordAsAPrefix(t *testing.T) {
	t.Parallel()

	query := newTextQuery("  Acme dock-hours ", nil)
	require.False(t, query.empty())
	require.False(t, query.unmatchable())

	primary := query.primary()
	assert.Equal(t, keel.WebSearch, primary.mode)
	assert.Equal(t, "Acme dock-hours", primary.text)
	assert.Equal(t, "acme:* & dock:* & hours:*", primary.prefixes)

	assert.Nil(t, query.fallback(), "with no meaningful words named there is no looser pass")
}

func TestTextQuery_FallsBackToAnyTwoMeaningfulWords(t *testing.T) {
	t.Parallel()

	query := newTextQuery(
		"what are the dock hours for acme",
		[]string{"dock", "hour", "acme", "acme", "a b", "it"},
	)

	fallback := query.fallback()
	require.NotNil(t, fallback)
	assert.Equal(t, keel.TSQuery, fallback.mode)
	assert.Empty(t, fallback.prefixes)
	assert.Equal(t,
		"(dock:* & hour:*) | (dock:* & acme:*) | (hour:* & acme:*)",
		fallback.text,
		"a memory must share two words; a repeat, a phrase and a short word are not terms",
	)
}

func TestTextQuery_OneMeaningfulWordHasNoLooserPass(t *testing.T) {
	t.Parallel()

	query := newTextQuery("what is the plan for acme", []string{"plan"})
	assert.Nil(t, query.fallback(), "a single word would match every memory that uses it")
}

func TestTextQuery_BoundsTheFallbackTerms(t *testing.T) {
	t.Parallel()

	terms := []string{"aaa", "bbb", "ccc", "ddd", "eee", "fff", "ggg", "hhh", "iii", "jjj"}
	query := newTextQuery(strings.Join(terms, " "), terms)
	assert.Len(t, query.terms, maxFallbackTerms)
	pairs := strings.Count(query.fallback().text, "&")
	assert.Equal(t, maxFallbackTerms*(maxFallbackTerms-1)/2, pairs)
}

func TestTextQuery_KeepsWebsearchOperatorsWhole(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{`"dock 9"`, "acme -friday", "acme OR beta", "acme or beta"} {
		query := newTextQuery(raw, nil)
		primary := query.primary()
		assert.Equal(t, keel.WebSearch, primary.mode, raw)
		assert.Equal(t, raw, primary.text, raw)
		assert.Empty(t, primary.prefixes, raw)
		assert.Nil(t, query.fallback(), "an exclusion is never undone by a looser pass: %s", raw)
	}
}

func TestTextQuery_OneWordHasNoLooserPass(t *testing.T) {
	t.Parallel()

	query := newTextQuery("deliver", nil)
	primary := query.primary()
	assert.Equal(t, "deliver", primary.text)
	assert.Equal(t, "deliver:*", primary.prefixes, "deliver finds delivery through its prefix")
	assert.Nil(t, query.fallback())
}

func TestTextQuery_SingleLettersAreNotPrefixes(t *testing.T) {
	t.Parallel()

	query := newTextQuery("a b", nil)
	assert.Empty(t, query.primary().prefixes,
		"a one-letter prefix would match nearly every memory")
	assert.Nil(t, query.fallback())
}

func TestTextQuery_PunctuationAloneMatchesNothing(t *testing.T) {
	t.Parallel()

	query := newTextQuery("?!,", nil)
	assert.False(t, query.empty())
	assert.True(t, query.unmatchable())
	assert.True(t, newTextQuery("   ", nil).empty())
}

func TestTextQuery_BoundsTheWordsAndDropsRepeats(t *testing.T) {
	t.Parallel()

	query := newTextQuery(strings.Repeat("acme ", 3)+strings.Repeat("word ", 1)+
		"a1 a2 a3 a4 a5 a6 a7 a8 a9 b1 b2 b3 b4 b5 b6 b7 b8", nil)
	assert.Len(t, query.words, maxQueryWords)
	assert.Equal(t, "acme", query.words[0])
	assert.Equal(t, "word", query.words[1], "repeats are dropped and the order typed is kept")
}

func TestTextQuery_RendersOneSimpleSearch(t *testing.T) {
	t.Parallel()

	m := agentdb.Memories
	primary := newTextQuery("Acme dock-hours", nil).primary()
	sql, args := keeltest.Render(t, m.Pick(m.ID).
		Where(primary.matches(m.SearchVector)).
		Order(keel.Desc(primary.rank(m.SearchVector))).
		Limit(1))

	assert.Contains(t, sql, `@@ (websearch_to_tsquery('simple'::regconfig, $`)
	assert.Contains(t, sql, `) || to_tsquery('simple'::regconfig, $`)
	assert.Contains(t, sql, `ts_rank_cd(`)
	assert.Contains(t, args, "Acme dock-hours")
	assert.Contains(t, args, "acme:* & dock:* & hours:*")
}
