package tableinsightservice

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeriesBoundsWeeksStartOnMondayInTheZone(t *testing.T) {
	t.Parallel()

	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	now := time.Date(2026, time.October, 8, 3, 30, 0, 0, time.UTC)

	bounds, err := SeriesBounds(now, "America/Chicago", SeriesWeek, 3)
	require.NoError(t, err)

	require.Len(t, bounds, 4)
	assert.Equal(t, time.Date(2026, time.September, 21, 0, 0, 0, 0, chicago).Unix(), bounds[0])
	assert.Equal(t, time.Date(2026, time.October, 5, 0, 0, 0, 0, chicago).Unix(), bounds[2])
	assert.Equal(t, time.Date(2026, time.October, 12, 0, 0, 0, 0, chicago).Unix(), bounds[3])
}

func TestSeriesBoundsMonthsFollowCalendarLengthsAcrossDaylightSaving(t *testing.T) {
	t.Parallel()

	newYork, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	now := time.Date(2026, time.March, 20, 12, 0, 0, 0, newYork)

	bounds, err := SeriesBounds(now, "America/New_York", SeriesMonth, 2)
	require.NoError(t, err)

	assert.Equal(t, []int64{
		time.Date(2026, time.February, 1, 0, 0, 0, 0, newYork).Unix(),
		time.Date(2026, time.March, 1, 0, 0, 0, 0, newYork).Unix(),
		time.Date(2026, time.April, 1, 0, 0, 0, 0, newYork).Unix(),
	}, bounds)
	assert.Equal(t, int64(31*24*3600-3600), bounds[2]-bounds[1])
}

func TestSeriesBoundsRefusesWhatItCannotChart(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.October, 8, 0, 0, 0, 0, time.UTC)
	cases := map[string]struct {
		timezone string
		interval SeriesInterval
		periods  int
	}{
		"no periods":       {interval: SeriesWeek, periods: 0},
		"too many periods": {interval: SeriesWeek, periods: 61},
		"unknown zone":     {timezone: "Mars/Olympus", interval: SeriesWeek, periods: 4},
		"unknown interval": {interval: "fortnight", periods: 4},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := SeriesBounds(now, tc.timezone, tc.interval, tc.periods)
			assert.Error(t, err)
		})
	}
}
