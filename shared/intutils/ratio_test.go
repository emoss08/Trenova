package intutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRatioLeadExceedsPoints(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name                         string
		num, den, otherNum, otherDen int
		points                       int
		want                         bool
	}{
		{name: "exactly the allowance", num: 180, den: 200, otherNum: 85, otherDen: 100, points: 5},
		{name: "just past it", num: 180, den: 200, otherNum: 849, otherDen: 1000, points: 5, want: true},
		{name: "behind", num: 85, den: 100, otherNum: 90, otherDen: 100, points: 5},
		{name: "no evidence", num: 1, den: 0, otherNum: 0, otherDen: 10, points: 1},
		{name: "large counts", num: 900_000, den: 1_000_000, otherNum: 800_000, otherDen: 1_000_000, points: 9, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, RatioLeadExceedsPoints(tc.num, tc.den, tc.otherNum, tc.otherDen, tc.points))
		})
	}
}
