package agentruntime

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/tooldispatch"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
)

func (t *Turn) MoneySpent() agenttooltrust.MoneyUse {
	spent := tooldispatch.UnrecordedMoney(t.result.Actions)
	for idx := range t.result.Delegations {
		spent = spent.Plus(tooldispatch.UnrecordedMoney(t.result.Delegations[idx].Actions))
	}
	if t.req.Delegation != nil {
		spent = spent.Plus(t.req.Delegation.MoneySpent)
	}

	return spent
}
