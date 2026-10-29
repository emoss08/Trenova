package tooldispatch

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/stretchr/testify/assert"
)

func TestHeldReason_ExplainsEveryMoneyBound(t *testing.T) {
	t.Parallel()

	for _, key := range agenttooltrust.MoneyHeldByKeys() {
		reason := heldReason([]string{key}, agent.EgressMoney)
		assert.NotEmpty(t, reason, key)
		assert.Contains(t, reason, moneyHeldReasons[key], key)
	}
}
