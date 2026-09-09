package ifta_test

import (
	"testing"
	"time"
	_ "time/tzdata"

	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const secondsPerDay = int64(86_400)

func mustLocation(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	require.NoError(t, err)
	return loc
}

func TestPeriod_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		period ifta.Period
		want   error
	}{
		{"valid", ifta.NewPeriod(2026, 3), nil},
		{"year too early", ifta.NewPeriod(1999, 1), ifta.ErrPeriodYearOutOfRange},
		{"year too late", ifta.NewPeriod(2101, 1), ifta.ErrPeriodYearOutOfRange},
		{"quarter zero", ifta.NewPeriod(2026, 0), ifta.ErrPeriodQuarterInvalid},
		{"quarter five", ifta.NewPeriod(2026, 5), ifta.ErrPeriodQuarterInvalid},
		{"boundary years", ifta.NewPeriod(2000, 4), nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.period.Validate()
			if tt.want == nil {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, tt.want)
		})
	}
}

func TestPeriod_KeyAndLabel(t *testing.T) {
	t.Parallel()

	period := ifta.NewPeriod(2026, 3)
	assert.Equal(t, "2026Q3", period.Key())
	assert.Equal(t, "Q3 2026", period.Label())
	assert.Equal(t, time.July, period.StartMonth())
}

func TestPeriod_PreviousNext(t *testing.T) {
	t.Parallel()

	assert.Equal(t, ifta.NewPeriod(2025, 4), ifta.NewPeriod(2026, 1).Previous())
	assert.Equal(t, ifta.NewPeriod(2026, 2), ifta.NewPeriod(2026, 3).Previous())
	assert.Equal(t, ifta.NewPeriod(2027, 1), ifta.NewPeriod(2026, 4).Next())
	assert.Equal(t, ifta.NewPeriod(2026, 4), ifta.NewPeriod(2026, 3).Next())
	assert.True(t, ifta.NewPeriod(2026, 1).Before(ifta.NewPeriod(2026, 2)))
	assert.True(t, ifta.NewPeriod(2025, 4).Before(ifta.NewPeriod(2026, 1)))
	assert.False(t, ifta.NewPeriod(2026, 2).Before(ifta.NewPeriod(2026, 2)))
	assert.True(t, ifta.NewPeriod(2026, 2).Equal(ifta.NewPeriod(2026, 2)))
}

func TestPeriod_BoundsUTC(t *testing.T) {
	t.Parallel()

	start, end := ifta.NewPeriod(2026, 1).Bounds(time.UTC)
	assert.Equal(t, int64(1_767_225_600), start, "2026-01-01T00:00:00Z")
	assert.Equal(t, int64(1_775_001_600), end, "2026-04-01T00:00:00Z")

	start, end = ifta.NewPeriod(2026, 1).Bounds(nil)
	assert.Equal(t, int64(1_767_225_600), start, "nil location falls back to UTC")
	assert.Equal(t, int64(1_775_001_600), end)
}

func TestPeriod_BoundsAcrossDST(t *testing.T) {
	t.Parallel()

	days := map[int]int64{1: 90, 2: 91, 3: 92, 4: 92}
	dstShift := map[int]int64{1: -3600, 2: 0, 3: 0, 4: 3600}

	for _, zone := range []string{"America/New_York", "America/Los_Angeles"} {
		for quarter := 1; quarter <= 4; quarter++ {
			t.Run(zone+"/Q"+string(rune('0'+quarter)), func(t *testing.T) {
				t.Parallel()

				loc := mustLocation(t, zone)
				period := ifta.NewPeriod(2026, quarter)
				start, end := period.Bounds(loc)

				startLocal := time.Unix(start, 0).In(loc)
				assert.Equal(t, 2026, startLocal.Year())
				assert.Equal(t, period.StartMonth(), startLocal.Month())
				assert.Equal(t, 1, startLocal.Day())
				assert.Equal(t, 0, startLocal.Hour(), "start is local midnight")

				endLocal := time.Unix(end, 0).In(loc)
				assert.Equal(t, 1, endLocal.Day())
				assert.Equal(t, 0, endLocal.Hour(), "end is local midnight")
				assert.Equal(t, period.Next().StartMonth(), endLocal.Month())

				assert.Equal(t, days[quarter]*secondsPerDay+dstShift[quarter], end-start,
					"quarter length reflects the DST transition inside it")

				assert.True(t, period.Contains(start, loc))
				assert.False(t, period.Contains(end, loc), "bounds are half-open")
				assert.True(t, period.Contains(end-1, loc))
			})
		}
	}
}

func TestPeriod_BoundsChainWithoutGaps(t *testing.T) {
	t.Parallel()

	loc := mustLocation(t, "America/New_York")
	for quarter := 1; quarter <= 4; quarter++ {
		period := ifta.NewPeriod(2026, quarter)
		_, end := period.Bounds(loc)
		nextStart, _ := period.Next().Bounds(loc)
		assert.Equal(t, end, nextStart, "Q%d end must equal next start", quarter)
	}
}

func TestPeriodOf_LocalMidnightEdge(t *testing.T) {
	t.Parallel()

	ny := mustLocation(t, "America/New_York")
	la := mustLocation(t, "America/Los_Angeles")

	lateMarch := time.Date(2026, time.March, 31, 23, 59, 0, 0, ny).Unix()
	assert.Equal(t, ifta.NewPeriod(2026, 1), ifta.PeriodOf(lateMarch, ny),
		"23:59 local on Mar 31 is still Q1 in New York")
	assert.Equal(t, ifta.NewPeriod(2026, 2), ifta.PeriodOf(lateMarch, time.UTC),
		"the same instant is already Apr 1 in UTC")
	assert.Equal(t, ifta.NewPeriod(2026, 1), ifta.PeriodOf(lateMarch, la),
		"and still Mar 31 evening in Los Angeles")

	newYearUTC := time.Date(2027, time.January, 1, 2, 0, 0, 0, time.UTC).Unix()
	assert.Equal(t, ifta.NewPeriod(2027, 1), ifta.PeriodOf(newYearUTC, time.UTC))
	assert.Equal(t, ifta.NewPeriod(2026, 4), ifta.PeriodOf(newYearUTC, ny),
		"02:00 UTC on Jan 1 is still Dec 31 in New York")

	assert.Equal(t, ifta.NewPeriod(2026, 2), ifta.PeriodOf(lateMarch, nil),
		"nil location falls back to UTC")
	assert.Equal(t, ifta.PeriodOf(lateMarch, time.UTC), ifta.PeriodOf(lateMarch, nil))
}

func TestPeriod_DueDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		period    ifta.Period
		wantYear  int
		wantMonth time.Month
		wantDay   int
	}{
		{ifta.NewPeriod(2026, 1), 2026, time.April, 30},
		{ifta.NewPeriod(2026, 2), 2026, time.July, 31},
		{ifta.NewPeriod(2026, 3), 2026, time.October, 31},
		{ifta.NewPeriod(2026, 4), 2027, time.January, 31},
	}

	for _, zone := range []string{"UTC", "America/New_York", "America/Los_Angeles"} {
		for _, tt := range tests {
			t.Run(zone+"/"+tt.period.Key(), func(t *testing.T) {
				t.Parallel()

				loc := mustLocation(t, zone)
				due := time.Unix(tt.period.DueDate(loc), 0).In(loc)

				assert.Equal(t, tt.wantYear, due.Year())
				assert.Equal(t, tt.wantMonth, due.Month())
				assert.Equal(t, tt.wantDay, due.Day())
				assert.Equal(t, 0, due.Hour(), "due date is local midnight at the start of the day")

				_, end := tt.period.Bounds(loc)
				assert.Greater(t, tt.period.DueDate(loc), end, "due after the quarter closes")
			})
		}
	}
}
