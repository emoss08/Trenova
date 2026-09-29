package timeutils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRelativeDays(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "today", RelativeDays(0))
	assert.Equal(t, "tomorrow", RelativeDays(1))
	assert.Equal(t, "yesterday", RelativeDays(-1))
	assert.Equal(t, "in 7 days", RelativeDays(7))
	assert.Equal(t, "3 days ago", RelativeDays(-3))
}

func TestCalendarDaysBetweenCountsWholeDaysInTheZone(t *testing.T) {
	t.Parallel()

	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	now := time.Date(2026, time.September, 29, 23, 30, 0, 0, chicago)
	assert.Equal(t, int64(1),
		CalendarDaysBetween(now, time.Date(2026, time.September, 30, 0, 30, 0, 0, chicago)))
	assert.Equal(t, int64(7),
		CalendarDaysBetween(now, time.Date(2026, time.October, 6, 8, 0, 0, 0, chicago)))
	assert.Equal(t, int64(-29),
		CalendarDaysBetween(now, time.Date(2026, time.August, 31, 8, 0, 0, 0, chicago)))
	assert.Equal(t, int64(0),
		CalendarDaysBetween(now, time.Date(2026, time.September, 30, 4, 0, 0, 0, time.UTC)))
}

func TestCalendarDayOnPicksTheYearNearestNow(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.December, 28, 9, 0, 0, 0, time.UTC)

	assert.Equal(t, time.Date(2027, time.January, 3, 0, 0, 0, 0, time.UTC),
		CalendarDay{Month: time.January, Day: 3}.On(now))
	assert.Equal(t, time.Date(2026, time.December, 20, 0, 0, 0, 0, time.UTC),
		CalendarDay{Month: time.December, Day: 20}.On(now))
	assert.Equal(t, time.Date(2025, time.March, 1, 0, 0, 0, 0, time.UTC),
		CalendarDay{Year: 2025, Month: time.March, Day: 1}.On(now))
}

func TestFindWrittenDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  WrittenDate
		ok    bool
	}{
		{
			name:  "iso day",
			input: "2026-10-06",
			want: WrittenDate{
				Day: CalendarDay{2026, time.October, 6}, Start: 0, End: 10,
			},
			ok: true,
		},
		{
			name:  "iso with clock",
			input: "Pick up 2026-10-06T14:30",
			want: WrittenDate{
				Day: CalendarDay{2026, time.October, 6}, Start: 8, End: 24, Clock: "14:30",
			},
			ok: true,
		},
		{
			name:  "iso with spaced clock",
			input: "2026-10-06 08:00",
			want: WrittenDate{
				Day: CalendarDay{2026, time.October, 6}, Start: 0, End: 16, Clock: "08:00",
			},
			ok: true,
		},
		{
			name:  "named without year",
			input: "Oct 6",
			want:  WrittenDate{Day: CalendarDay{0, time.October, 6}, Start: 0, End: 5},
			ok:    true,
		},
		{
			name:  "named with year",
			input: "October 6th, 2026 (morning)",
			want:  WrittenDate{Day: CalendarDay{2026, time.October, 6}, Start: 0, End: 17},
			ok:    true,
		},
		{
			name:  "numeric with a four digit year",
			input: "10/06/2026",
			want:  WrittenDate{Day: CalendarDay{2026, time.October, 6}, Start: 0, End: 10},
			ok:    true,
		},
		{
			name:  "day first",
			input: "10 October 2026",
			want:  WrittenDate{Day: CalendarDay{2026, time.October, 10}, Start: 0, End: 15},
			ok:    true,
		},
		{name: "a range of days is not a date", input: "1-2 days", ok: false},
		{name: "a fraction is not a date", input: "3/4 truckload", ok: false},
		{name: "no date", input: "30 days", ok: false},
		{name: "impossible day", input: "2026-02-30", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := FindWrittenDate(tt.input)
			assert.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestFindWrittenDateReadsOnlyWholeMonthNames(t *testing.T) {
	t.Parallel()

	_, ok := FindWrittenDate("Decline 3 loads")
	assert.False(t, ok)

	got, ok := FindWrittenDate("Tue Sept 8")
	require.True(t, ok)
	assert.Equal(t, CalendarDay{0, time.September, 8}, got.Day)
	assert.Equal(t, 4, got.Start)
}

func TestDescribeUnixDateInCountsADayShortenedByTheClockChange(t *testing.T) {
	t.Parallel()

	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	now := time.Date(2026, time.March, 7, 12, 0, 0, 0, chicago).Unix()
	next := time.Date(2026, time.March, 8, 9, 0, 0, 0, chicago).Unix()

	assert.Equal(t, "2026-03-08 (tomorrow)", DescribeUnixDateIn(next, now, "America/Chicago"))
}
