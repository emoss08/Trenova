package timeutils_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
)

func TestBusinessDaysBetween(t *testing.T) {
	t.Parallel()

	chicago, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Skip("tzdata unavailable")
	}
	at := func(loc *time.Location, year int, month time.Month, day, hour int) int64 {
		return time.Date(year, month, day, hour, 0, 0, 0, loc).Unix()
	}

	tests := []struct {
		name     string
		from, to int64
		loc      *time.Location
		want     int
	}{
		{
			name: "the next weekday is one",
			from: at(time.UTC, 2026, time.November, 5, 10),
			to:   at(time.UTC, 2026, time.November, 6, 9),
			want: 1,
		},
		{
			name: "friday to monday skips the weekend",
			from: at(time.UTC, 2026, time.November, 6, 10),
			to:   at(time.UTC, 2026, time.November, 9, 15),
			want: 1,
		},
		{
			name: "two whole weeks are ten",
			from: at(time.UTC, 2026, time.November, 2, 9),
			to:   at(time.UTC, 2026, time.November, 16, 9),
			want: 10,
		},
		{
			name: "the same day is none",
			from: at(time.UTC, 2026, time.November, 5, 8),
			to:   at(time.UTC, 2026, time.November, 5, 17),
			want: 0,
		},
		{
			name: "an end before the start is none",
			from: at(time.UTC, 2026, time.November, 9, 8),
			to:   at(time.UTC, 2026, time.November, 5, 8),
			want: 0,
		},
		{
			name: "days are counted in the organization's zone, across a clock change",
			from: at(chicago, 2026, time.March, 6, 22),
			to:   at(chicago, 2026, time.March, 9, 1),
			loc:  chicago,
			want: 1,
		},
		{
			name: "an evening in the organization's zone is still its own day",
			from: at(chicago, 2026, time.March, 5, 21),
			to:   at(chicago, 2026, time.March, 6, 20),
			loc:  chicago,
			want: 1,
		},
		{
			name: "a nil zone counts in UTC",
			from: at(time.UTC, 2026, time.November, 5, 23),
			to:   at(time.UTC, 2026, time.November, 6, 1),
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, timeutils.BusinessDaysBetween(tt.from, tt.to, tt.loc))
		})
	}
}
