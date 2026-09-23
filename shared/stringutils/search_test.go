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
