package conform

import (
	"testing"

	"github.com/emoss08/trenova/edi-partner-sim/internal/acknowledge"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
)

func TestDecideAgreesWithTheRejectRules(t *testing.T) {
	t.Parallel()

	rules := []partner.RejectRules{
		{When: partner.RejectNever},
		{When: partner.RejectErrors},
		{When: partner.RejectAny},
		{When: partner.RejectErrors, Sets: []string{"214"}},
	}
	for _, rule := range rules {
		a := partner.Acknowledgment{Reject: rule}
		for _, set := range []string{"990", "214"} {
			for errs := range 3 {
				for warns := range 3 {
					got := acknowledge.Code(&a, set, errs, warns)
					if want := expectedAck(rule, set, errs, warns); got != want {
						t.Errorf("%+v %s %d/%d: Decide %s, rules %s",
							rule, set, errs, warns, got, want)
					}
				}
			}
		}
	}
}
