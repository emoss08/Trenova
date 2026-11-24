package boardcatalog_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPeriodDaysAreWholeDaysOfTheOrganizationsZone(t *testing.T) {
	t.Parallel()

	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	now := time.Date(2026, time.November, 23, 1, 30, 0, 0, chicago).Unix()
	day := func(month time.Month, d int) time.Time {
		return time.Date(2026, month, d, 0, 0, 0, 0, chicago)
	}

	cases := []struct {
		period      boardcatalog.Period
		first, last time.Time
	}{
		{boardcatalog.PeriodToday, day(time.November, 23), day(time.November, 23)},
		{boardcatalog.PeriodYesterday, day(time.November, 22), day(time.November, 22)},
		{boardcatalog.PeriodThisWeek, day(time.November, 23), day(time.November, 23)},
		{boardcatalog.PeriodLastWeek, day(time.November, 16), day(time.November, 22)},
		{boardcatalog.PeriodNextWeek, day(time.November, 30), day(time.December, 6)},
		{boardcatalog.PeriodThisMonth, day(time.November, 1), day(time.November, 23)},
		{boardcatalog.PeriodLastMonth, day(time.October, 1), day(time.October, 31)},
		{boardcatalog.PeriodThisQuarter, day(time.October, 1), day(time.November, 23)},
		{boardcatalog.PeriodLastQuarter, day(time.July, 1), day(time.September, 30)},
	}
	for _, tc := range cases {
		first, last, ok := tc.period.Days(now, chicago)
		require.True(t, ok, tc.period)
		assert.Equal(t, tc.first, first, tc.period)
		assert.Equal(t, tc.last, last, tc.period)
	}

	_, _, ok := boardcatalog.PeriodNext48Hours.Days(now, chicago)
	assert.False(t, ok, "the next 48 hours are not whole days")
}
