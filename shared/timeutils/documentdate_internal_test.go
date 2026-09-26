package timeutils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFindDocumentDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  CalendarDay
		ok    bool
	}{
		{name: "us numeric with year", input: "03/15/2026 08:00-10:00", want: CalendarDay{2026, time.March, 15}, ok: true},
		{name: "two digit year", input: "Pickup 3-5-26", want: CalendarDay{2026, time.March, 5}, ok: true},
		{name: "no year", input: "Deliver 12/01", want: CalendarDay{0, time.December, 1}, ok: true},
		{name: "iso", input: "2026-09-25T10:00", want: CalendarDay{2026, time.September, 25}, ok: true},
		{name: "named with year", input: "Sept. 4th, 2026 appt", want: CalendarDay{2026, time.September, 4}, ok: true},
		{name: "named without year", input: "Jan 31", want: CalendarDay{0, time.January, 31}, ok: true},
		{name: "impossible day", input: "02/30/2026", ok: false},
		{name: "leap day without year", input: "02/29", want: CalendarDay{0, time.February, 29}, ok: true},
		{name: "no date", input: "FCFS 08:00-16:00", ok: false},
		{name: "empty", input: "  ", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := FindDocumentDate(tt.input)
			assert.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestCalendarDayMatches(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, time.March, 15, 9, 0, 0, 0, time.UTC)

	assert.True(t, CalendarDay{2026, time.March, 15}.Matches(at))
	assert.True(t, CalendarDay{0, time.March, 15}.Matches(at))
	assert.False(t, CalendarDay{2025, time.March, 15}.Matches(at))
	assert.False(t, CalendarDay{2026, time.March, 16}.Matches(at))
	assert.Equal(t, "2026-03-15", CalendarDay{2026, time.March, 15}.String())
	assert.Equal(t, "03-15", CalendarDay{0, time.March, 15}.String())
}
