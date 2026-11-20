package customerledger

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func utc(year int, month time.Month, day, hour int) int64 {
	return time.Date(year, month, day, hour, 0, 0, 0, time.UTC).Unix()
}

func TestMonthStartIsTheUTCCalendarMonth(t *testing.T) {
	t.Parallel()

	jan := utc(2026, time.January, 1, 0)
	central := time.FixedZone("CST", -6*3600)
	cases := map[string]struct {
		at   int64
		want int64
	}{
		"the first second": {at: jan, want: jan},
		"mid month":        {at: utc(2026, time.January, 17, 9), want: jan},
		"the last second":  {at: utc(2026, time.February, 1, 0) - 1, want: jan},
		"a leap day": {
			at:   utc(2028, time.February, 29, 23),
			want: utc(2028, time.February, 1, 0),
		},
		"late on new year's eve in a US zone": {
			at:   time.Date(2025, time.December, 31, 22, 0, 0, 0, central).Unix(),
			want: jan,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, MonthStart(tc.at))
		})
	}
}
