package spoken

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
)

func TestVariationsAndAmountsReadAsPeopleSayThem(t *testing.T) {
	t.Parallel()

	for _, c := range []struct {
		got, want string
	}{
		{Variation(partner.Variation{Kind: partner.KindOriginal}), "Original tender"},
		{Variation(partner.Variation{Kind: partner.KindDuplicate}), "Exact resend"},
		{Variation(partner.Variation{Kind: partner.KindChange, Delta: partner.Delta{AddStop: true}}),
			"Change: a stop added"},
		{Variation(partner.Variation{
			Kind:  partner.KindChange,
			Delta: partner.Delta{Move: &partner.Move{By: 13 * time.Hour}},
		}), "Change: the last stop moved 13 h later"},
		{Variation(partner.Variation{Kind: partner.KindReplace, Delta: partner.Delta{RatePercent: -8,
			Move: &partner.Move{Stop: 1, By: -150 * time.Minute}}}),
			"Replacement: total -8%, stop 1 moved 2 h 30 min earlier"},
		{Dollars(-15000), "$150.00"},
		{Dollars(123456789), "$1,234,567.89"},
		{Duration(45 * time.Minute), "45 min"},
	} {
		if c.got != c.want {
			t.Errorf("got %q, want %q", c.got, c.want)
		}
	}
}
