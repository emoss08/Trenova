package tenant

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
)

func TestAgentControl_ValidatePersonMonthlyMessages(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		value   int
		invalid bool
	}{
		{name: "unlimited", value: 0},
		{name: "a set allowance", value: 250},
		{name: "negative", value: -1, invalid: true},
		{name: "past the ceiling", value: maxPersonMonthlyMessages + 1, invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			control := &AgentControl{
				PromotionThreshold:    DefaultPromotionThreshold,
				BriefingHourLocal:     DefaultBriefingHourLocal,
				PersonMonthlyMessages: tc.value,
			}
			me := errortypes.NewMultiError()
			control.Validate(me)
			assert.Equal(t, tc.invalid, me.HasErrors())
		})
	}
}
