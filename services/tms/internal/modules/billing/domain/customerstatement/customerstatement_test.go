package customerstatement

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMonthlyPeriodIsTheMonthBefore(t *testing.T) {
	t.Parallel()

	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	now := time.Date(2026, time.September, 3, 9, 0, 0, 0, chicago).Unix()
	period, due := MonthlyPeriod(now, chicago, 1)

	assert.True(t, due)
	assert.Equal(t, time.Date(2026, time.August, 1, 0, 0, 0, 0, chicago).Unix(), period.Start)
	assert.Equal(t, time.Date(2026, time.August, 31, 23, 59, 59, 0, chicago).Unix(), period.End)
}

func TestMonthlyPeriodWaitsForTheDay(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.September, 4, 12, 0, 0, 0, time.UTC).Unix()

	_, due := MonthlyPeriod(now, time.UTC, 5)
	assert.False(t, due)

	_, due = MonthlyPeriod(now+24*60*60, time.UTC, 5)
	assert.True(t, due)
}

func TestMonthlyPeriodUsesTheLocalDate(t *testing.T) {
	t.Parallel()

	tokyo, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)

	// 20:00 UTC on 31 August is already 1 September in Tokyo.
	now := time.Date(2026, time.August, 31, 20, 0, 0, 0, time.UTC).Unix()
	period, due := MonthlyPeriod(now, tokyo, 1)

	assert.True(t, due)
	assert.Equal(t, time.Date(2026, time.August, 1, 0, 0, 0, 0, tokyo).Unix(), period.Start)
}

func TestMonthlyPeriodAcrossTheYear(t *testing.T) {
	t.Parallel()

	now := time.Date(2027, time.January, 15, 0, 0, 0, 0, time.UTC).Unix()
	period, due := MonthlyPeriod(now, nil, 40)

	assert.True(t, due, "a day out of range is treated as the first")
	assert.Equal(t, time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC).Unix(), period.Start)
	assert.Equal(t, time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()-1, period.End)
}

func TestResolveFillsAMissingStartWithTheMonthBefore(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		end  time.Time
		want time.Time
	}{
		"september 30": {
			time.Date(2026, time.September, 30, 12, 0, 0, 0, time.UTC),
			time.Date(2026, time.August, 30, 12, 0, 0, 0, time.UTC),
		},
		"march 31 clamps to february 28": {
			time.Date(2026, time.March, 31, 12, 0, 0, 0, time.UTC),
			time.Date(2026, time.February, 28, 12, 0, 0, 0, time.UTC),
		},
		"march 31 in a leap year clamps to february 29": {
			time.Date(2028, time.March, 31, 12, 0, 0, 0, time.UTC),
			time.Date(2028, time.February, 29, 12, 0, 0, 0, time.UTC),
		},
		"january 15 crosses the year": {
			time.Date(2027, time.January, 15, 12, 0, 0, 0, time.UTC),
			time.Date(2026, time.December, 15, 12, 0, 0, 0, time.UTC),
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			period, err := Period{End: tc.end.Unix()}.Resolve(time.UTC)

			require.NoError(t, err)
			assert.Equal(t, tc.want.Unix(), period.Start)
			assert.Equal(t, tc.end.Unix(), period.End)
		})
	}
}

func TestResolveCountsTheMonthInTheGivenZone(t *testing.T) {
	t.Parallel()

	loc, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	end := time.Date(2026, time.March, 31, 23, 0, 0, 0, loc)

	period, err := Period{End: end.Unix()}.Resolve(loc)

	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, time.February, 28, 23, 0, 0, 0, loc).Unix(), period.Start)
}

func TestResolveKeepsAGivenStart(t *testing.T) {
	t.Parallel()

	period, err := Period{Start: 100, End: 200}.Resolve(time.UTC)

	require.NoError(t, err)
	assert.Equal(t, Period{Start: 100, End: 200}, period)
}

func TestResolveNeverStartsBeforeTheEpoch(t *testing.T) {
	t.Parallel()

	period, err := Period{End: 3600}.Resolve(time.UTC)

	require.NoError(t, err)
	assert.Equal(t, Period{Start: 0, End: 3600}, period)
}

func TestResolveRefusesABadSpan(t *testing.T) {
	t.Parallel()

	day := int64(24 * 60 * 60)
	cases := map[string]Period{
		"a negative start":      {Start: -1, End: 10},
		"a negative end":        {Start: 0, End: -1},
		"a start after the end": {Start: 20, End: 10},
		"longer than the most":  {Start: day, End: day + (MaxPeriodDays+1)*day},
	}
	for name, period := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := period.Resolve(time.UTC)

			var validation *errortypes.Error
			require.ErrorAs(t, err, &validation)
			assert.Equal(t, "startDate", validation.Field())
		})
	}
}

func TestResolveAllowsTheLongestSpan(t *testing.T) {
	t.Parallel()

	day := int64(24 * 60 * 60)

	_, err := Period{Start: day, End: day + MaxPeriodDays*day}.Resolve(time.UTC)

	require.NoError(t, err)
}
