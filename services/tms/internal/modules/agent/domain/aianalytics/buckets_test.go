package aianalytics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBucketStartsBeginAtTheViewersLocalBoundaries(t *testing.T) {
	t.Parallel()

	newYork, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	until := time.Date(2026, 10, 14, 15, 20, 0, 0, newYork).Unix()

	starts := BucketStarts(WindowWeek, newYork, until)

	require.Len(t, starts, 28)
	assert.Equal(t, time.Date(2026, 10, 14, 12, 0, 0, 0, newYork).Unix(), starts[27])
	assert.Equal(t, time.Date(2026, 10, 14, 6, 0, 0, 0, newYork).Unix(), starts[26])
	assert.Equal(t, time.Date(2026, 10, 7, 18, 0, 0, 0, newYork).Unix(), starts[0])
}

func TestBucketStartsFollowHalfHourZones(t *testing.T) {
	t.Parallel()

	kolkata, err := time.LoadLocation("Asia/Kolkata")
	require.NoError(t, err)
	until := time.Date(2026, 10, 14, 9, 45, 0, 0, kolkata).Unix()

	starts := BucketStarts(WindowDay, kolkata, until)

	require.Len(t, starts, 24)
	assert.Equal(t, time.Date(2026, 10, 14, 9, 0, 0, 0, kolkata).Unix(), starts[23])
	assert.Equal(t, time.Date(2026, 10, 13, 10, 0, 0, 0, kolkata).Unix(), starts[0])
}

func TestDailyBucketStartsAreLocalMidnightsAcrossADaylightSavingChange(t *testing.T) {
	t.Parallel()

	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	until := time.Date(2026, 11, 3, 8, 0, 0, 0, chicago).Unix()

	starts := BucketStarts(WindowMonth, chicago, until)

	require.Len(t, starts, 30)
	for idx, start := range starts {
		local := time.Unix(start, 0).In(chicago)
		assert.Zero(t, local.Hour()*60+local.Minute(), "bucket %d starts at %s", idx, local)
	}
	assert.Equal(t, time.Date(2026, 11, 1, 0, 0, 0, 0, chicago).Unix(), starts[27])
	assert.Equal(t, int64(25*3600), starts[28]-starts[27])
}

func TestBucketStartsInUTCAreWholeBuckets(t *testing.T) {
	t.Parallel()

	starts := BucketStarts(WindowMonth, time.UTC, 1_790_000_000)

	require.Len(t, starts, 30)
	assert.Equal(t, int64(1_790_000_000-1_790_000_000%86400), starts[29])
	assert.Equal(t, int64(86400), starts[1]-starts[0])
}
