package filtercatalog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
"today" is the obvious thing to send, and refusing it cost a round trip.

Asked which drivers hold a current hazmat endorsement, the model filtered
hazmatExpiry with "today". The tool refused, naming what would work; the model
read the correction and asked again, and the second attempt answered correctly.
That is the refusal doing its job — but the question was answered on the second
call, billed twice, and on a flakier model the extra turn is where the
conversation dies.

The server has a clock. It can resolve the word.
*/
func TestCoerceDateValue_AcceptsTheWordsPeopleUseForDates(t *testing.T) {
	t.Parallel()

	for _, word := range []string{"today", "Today", " today ", "now"} {
		_, err := CoerceDate("profile.hazmatExpiry", word, Clock{})
		assert.NoError(t, err, "%q should resolve rather than be refused", word)
	}
}

func TestCoerceDateValue_ResolvesNamedDaysAgainstTheServerClock(t *testing.T) {
	t.Parallel()

	const now int64 = 1789776000 // 2026-09-19 00:00 UTC
	const day int64 = 86400

	today, ok := NamedDay("today", Clock{Now: now})
	require.True(t, ok)
	assert.Equal(t, now, today)

	tomorrow, ok := NamedDay("tomorrow", Clock{Now: now})
	require.True(t, ok)
	assert.Equal(t, now+day, tomorrow)

	yesterday, ok := NamedDay("yesterday", Clock{Now: now})
	require.True(t, ok)
	assert.Equal(t, now-day, yesterday)
}

// A named day resolves to midnight, so "expiring on or after today" includes
// something that expires later today rather than starting from this instant.
func TestNamedDay_ResolvesToTheStartOfTheDay(t *testing.T) {
	t.Parallel()

	midMorning := int64(1789815600) // 2026-09-19 11:00 UTC
	resolved, ok := NamedDay("today", Clock{Now: midMorning})

	require.True(t, ok)
	assert.Equal(t, int64(1789776000), resolved)
}

// Anything that is not a date is still refused, and the refusal still names
// every form that would have worked.
func TestCoerceDateValue_StillRefusesWhatIsNotADate(t *testing.T) {
	t.Parallel()

	_, err := CoerceDate("profile.hazmatExpiry", "soon", Clock{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "YYYY-MM-DD")
	assert.Contains(t, err.Error(), "today")
}

// "today" is the organization's today, and a bare date is midnight there.
func TestCatalog_ReadsNamedAndCalendarDaysInTheOrganizationsZone(t *testing.T) {
	t.Parallel()

	ny := Clock{Now: lateEveningNewYork, Timezone: "America/New_York"}

	today, ok := NamedDay("today", ny)
	require.True(t, ok)
	assert.Equal(t, int64(1789790400), today, "still the 19th in New York")

	seconds, err := CoerceDate("expiresAt", "2026-09-19", ny)
	require.NoError(t, err)
	assert.Equal(t, int64(1789790400), seconds)
}
