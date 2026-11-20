package timeutils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAddMonthsUTC(t *testing.T) {
	t.Parallel()

	jan31 := time.Date(2026, time.January, 31, 9, 30, 0, 0, time.UTC).Unix()
	assert.Equal(t,
		time.Date(2026, time.February, 28, 9, 30, 0, 0, time.UTC).Unix(),
		AddMonthsUTC(jan31, 1), "clamps to the last day of a short month")
	assert.Equal(t,
		time.Date(2026, time.March, 31, 9, 30, 0, 0, time.UTC).Unix(),
		AddMonthsUTC(jan31, 2), "a long target month keeps the day")
	assert.Equal(t,
		time.Date(2028, time.January, 31, 9, 30, 0, 0, time.UTC).Unix(),
		AddMonthsUTC(jan31, 24), "rolls across years")
	assert.Equal(t, jan31, AddMonthsUTC(jan31, 0))

	feb29 := time.Date(2024, time.February, 29, 0, 0, 0, 0, time.UTC).Unix()
	assert.Equal(t,
		time.Date(2025, time.February, 28, 0, 0, 0, 0, time.UTC).Unix(),
		AddMonthsUTC(feb29, 12))
}

func TestAddMonthsClampsBackwardStepsInTheGivenZone(t *testing.T) {
	t.Parallel()

	chicago, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Skip("tzdata unavailable")
	}

	tests := []struct {
		name string
		from time.Time
		want time.Time
	}{
		{
			name: "31 March to 28 February",
			from: time.Date(2026, time.March, 31, 0, 0, 0, 0, chicago),
			want: time.Date(2026, time.February, 28, 0, 0, 0, 0, chicago),
		},
		{
			name: "31 March to 29 February in a leap year",
			from: time.Date(2024, time.March, 31, 0, 0, 0, 0, chicago),
			want: time.Date(2024, time.February, 29, 0, 0, 0, 0, chicago),
		},
		{
			name: "31 January to 31 December",
			from: time.Date(2026, time.January, 31, 0, 0, 0, 0, chicago),
			want: time.Date(2025, time.December, 31, 0, 0, 0, 0, chicago),
		},
		{
			name: "local midnight survives the spring daylight change",
			from: time.Date(2026, time.April, 1, 0, 0, 0, 0, chicago),
			want: time.Date(2026, time.March, 1, 0, 0, 0, 0, chicago),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.True(t, tt.want.Equal(AddMonths(tt.from, -1)), AddMonths(tt.from, -1).String())
		})
	}
}
