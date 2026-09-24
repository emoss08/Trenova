package stringutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearchWords_SplitsOnAnythingButLettersAndDigits(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"on", "time"}, SearchWords("On-time"))
	assert.Equal(t, []string{"stop", "dwell", "and", "detention"}, SearchWords("stop-dwell-and-detention"))
	assert.Empty(t, SearchWords(" -- "))
}

func TestMatchesWordPrefixes(t *testing.T) {
	t.Parallel()

	assert.True(t, MatchesWordPrefixes(nil, "anything"), "no words matches everything")
	assert.True(t, MatchesWordPrefixes(SearchWords("deliver"), "Delivered, Not Billed"))
	assert.True(t, MatchesWordPrefixes(SearchWords("on time"), "Late or on-time stops"))
	assert.True(t, MatchesWordPrefixes(SearchWords("dwell exposure"), "Dwell & Detention Exposure"))
	assert.False(t, MatchesWordPrefixes(SearchWords("on"), "Operations"),
		"a word matches the start of a word, not its middle")
	assert.False(t, MatchesWordPrefixes(SearchWords("dwell billing"), "Dwell & Detention Exposure"),
		"every word must match")
	assert.True(t, MatchesWordPrefixes(SearchWords("detention"), "Dwell", "stop-dwell-and-detention"),
		"any of the texts can carry a word")
}

func TestSearchableKey_ReadsACamelCaseKeyAsWords(t *testing.T) {
	t.Parallel()

	key := SearchableKey("scheduledWindowEnd")
	assert.True(t, MatchesWordPrefixes(SearchWords("window"), key))
	assert.True(t, MatchesWordPrefixes(SearchWords("scheduled end"), key))
	assert.True(t, MatchesWordPrefixes(SearchWords("scheduledWindowEnd"), key),
		"the key itself still finds it")
	assert.False(t, MatchesWordPrefixes(SearchWords("indow"), key))

	dotted := SearchableKey("destinationStop.actualArrival")
	assert.True(t, MatchesWordPrefixes(SearchWords("arrival"), dotted))
	assert.True(t, MatchesWordPrefixes(SearchWords("destination stop"), dotted))
	assert.True(t, MatchesWordPrefixes(SearchWords("edi inbound"), SearchableKey("EDIInboundFile")))
	assert.Empty(t, SearchableKey(""))
}
