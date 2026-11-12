package recordcaseservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func snoozedItem(until int64) *watchtower.Item {
	return &watchtower.Item{Status: watchtower.StatusSnoozed, SnoozedUntil: &until}
}

/*
A case is put away only while nothing about its record is in front of anyone.
It comes back at the soonest time anything about it wakes, and one open item
(a new signal, a change to decide) puts it in front again at once.
*/
func TestAwayOf_ACaseIsAwayOnlyWhileEverythingAboutItIsSnoozed(t *testing.T) {
	t.Parallel()

	assert.Nil(t, awayOf(nil), "a case with nothing about it was not put away")

	away := awayOf([]*watchtower.Item{snoozedItem(900), snoozedItem(600)})
	require.NotNil(t, away)
	assert.Equal(t, int64(600), away.Until, "it comes back when the first of them wakes")

	assert.Nil(t, awayOf([]*watchtower.Item{
		snoozedItem(600),
		{Status: watchtower.StatusOpen},
	}), "something new about the record brings the case back")
}
