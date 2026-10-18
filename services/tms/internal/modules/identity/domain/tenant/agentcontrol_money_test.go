package tenant

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
)

func moneyControl() *AgentControl {
	control := &AgentControl{PromotionThreshold: DefaultPromotionThreshold}
	control.ApplyAgentMoneyDefaults()

	return control
}

func TestAgentControl_MoneyDefaultsAreValid(t *testing.T) {
	t.Parallel()

	multiErr := errortypes.NewMultiError()
	moneyControl().Validate(multiErr)
	assert.False(t, multiErr.HasErrors())
}

func TestAgentControl_ZeroMoneyLimitsTurnAutomaticMoneyOff(t *testing.T) {
	t.Parallel()

	control := moneyControl()
	control.AgentMoneyCallCeilingMinor = 0
	control.AgentMoneyRunCalls = 0
	control.AgentMoneyRunAmountMinor = 0
	control.AgentMoneyDayCalls = 0
	control.AgentMoneyDayAmountMinor = 0

	multiErr := errortypes.NewMultiError()
	control.Validate(multiErr)
	assert.False(t, multiErr.HasErrors())
}

func TestAgentControl_RefusesMoneyLimitsThatDoNotNest(t *testing.T) {
	t.Parallel()

	cases := map[string]func(*AgentControl){
		"agentMoneyCallCeiling": func(c *AgentControl) { c.AgentMoneyCallCeilingMinor = -1 },
		"agentMoneyDayAmount": func(c *AgentControl) {
			c.AgentMoneyDayAmountMinor = maxAgentMoneyAmountMinor + 1
		},
		"agentMoneyDayCalls": func(c *AgentControl) { c.AgentMoneyDayCalls = maxAgentMoneyCalls + 1 },
		"agentMoneyRunAmount": func(c *AgentControl) {
			c.AgentMoneyRunAmountMinor = c.AgentMoneyDayAmountMinor + 1
		},
		"agentMoneyRunCalls": func(c *AgentControl) { c.AgentMoneyRunCalls = c.AgentMoneyDayCalls + 1 },
	}
	for field, breaks := range cases {
		t.Run(field, func(t *testing.T) {
			t.Parallel()

			control := moneyControl()
			breaks(control)
			multiErr := errortypes.NewMultiError()
			control.Validate(multiErr)
			assert.True(t, multiErr.HasErrors())
			assert.Contains(t, multiErr.Error(), field)
		})
	}

	ceiling := moneyControl()
	ceiling.AgentMoneyCallCeilingMinor = ceiling.AgentMoneyRunAmountMinor + 1
	multiErr := errortypes.NewMultiError()
	ceiling.Validate(multiErr)
	assert.Contains(t, multiErr.Error(), "agentMoneyCallCeiling")
}
