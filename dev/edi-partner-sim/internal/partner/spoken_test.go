package partner

import (
	"testing"
	"time"
)

func TestVariationsAndAmountsReadAsPeopleSayThem(t *testing.T) {
	t.Parallel()

	for _, c := range []struct {
		got, want string
	}{
		{Variation{Kind: KindOriginal}.Describe(), "Original tender"},
		{Variation{Kind: KindDuplicate}.Describe(), "Exact resend"},
		{Variation{Kind: KindChange, Delta: Delta{AddStop: true}}.Describe(),
			"Change: a stop added"},
		{Variation{Kind: KindChange, Delta: Delta{Move: &Move{By: 13 * time.Hour}}}.Describe(),
			"Change: the last stop moved 13 h later"},
		{Variation{Kind: KindReplace, Delta: Delta{RatePercent: -8,
			Move: &Move{Stop: 1, By: -150 * time.Minute}}}.Describe(),
			"Replacement: total -8%, stop 1 moved 2 h 30 min earlier"},
		{Dollars(-15000), "$150.00"},
		{Dollars(123456789), "$1,234,567.89"},
		{SpokenDuration(45 * time.Minute), "45 min"},
	} {
		if c.got != c.want {
			t.Errorf("got %q, want %q", c.got, c.want)
		}
	}
}
