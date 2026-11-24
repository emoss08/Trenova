package boardwording_test

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardwording"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const day = int64(24 * 60 * 60)

func expiryOn(t *testing.T, ends time.Time, now time.Time) boardreading.Reading {
	t.Helper()
	countdown := boardwording.Countdown{
		Ends:     ends.Unix(),
		Span:     30 * day,
		Date:     true,
		Location: ends.Location(),
		Ending:   "Expires {0}",
		Ended:    "Expired {0}",
	}

	return countdown.Read(context.Background(), now.Unix())
}

func TestCountdownOfADateReadsWholeDaysInTheOrganizationsZone(t *testing.T) {
	t.Parallel()

	zone, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	now := time.Date(2026, time.November, 23, 14, 48, 0, 0, zone)

	left := expiryOn(t, time.Date(2026, time.December, 1, 19, 3, 0, 0, zone), now)
	assert.Equal(t, boardcatalog.Days(8), left.Value)
	assert.Equal(t, "{0} left", left.Standing.Text)
	assert.Equal(t, "Expires {0}", left.Fact.Text)
	assert.True(t, left.Fact.Day)
	assert.Equal(t, &boardreading.WindowFacts{Left: 8 * day, Span: 30 * day}, left.Window)
	assert.Empty(t, left.Tone, "eight days is not yet under a week")

	lapsed := expiryOn(t, time.Date(2026, time.November, 15, 19, 3, 0, 0, zone), now)
	assert.Equal(t, boardcatalog.Days(-8), lapsed.Value)
	assert.Equal(t, "{0} lapsed", lapsed.Standing.Text)
	assert.Equal(t, "Expired {0}", lapsed.Fact.Text)
	assert.Equal(t, boardcatalog.ToneBad, lapsed.Tone)
	assert.True(t, lapsed.Window.Lapsed())

	soon := expiryOn(t, time.Date(2026, time.November, 24, 0, 5, 0, 0, zone), now)
	assert.Equal(t, boardcatalog.Days(1), soon.Value)
	assert.Equal(t, boardcatalog.ToneWarn, soon.Tone)

	today := expiryOn(t, time.Date(2026, time.November, 23, 9, 0, 0, 0, zone), now)
	assert.Equal(t, boardcatalog.Days(0), today.Value)
	assert.Equal(t, "Ends today", today.Standing.Text)
	assert.False(t, today.Window.Lapsed(), "a date is good through its own day")
}

func TestCountdownOfAMomentReadsItsDuration(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.November, 23, 12, 0, 0, 0, time.UTC)
	countdown := boardwording.Countdown{
		Ends:   now.Add(90 * time.Minute).Unix(),
		Ending: "Ends {0}",
		Ended:  "Ended {0}",
	}
	reading := countdown.Read(context.Background(), now.Unix())

	assert.Equal(t, boardcatalog.Duration(90*60), reading.Value)
	assert.False(t, reading.Fact.Day)
	assert.Equal(t, int64(0), reading.Window.Span)
}
