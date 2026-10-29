package agenttest

import (
	"context"
	"sync"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentbudgetports"
)

type StubMoneyBudgets struct {
	agentbudgetports.AgentBudgetService

	Limits agenttooltrust.MoneyLimits
	Used   agenttooltrust.MoneyUse
	Err    error

	mu    sync.Mutex
	asked []agentbudgetports.MoneyAllowanceRequest
}

func RoomyMoneyLimits() agenttooltrust.MoneyLimits {
	return agenttooltrust.MoneyLimits{
		CallCeilingMinor: 100_000,
		RunCalls:         10,
		RunAmountMinor:   1_000_000,
		DayCalls:         100,
		DayAmountMinor:   10_000_000,
	}
}

func (b *StubMoneyBudgets) MoneyAllowance(
	_ context.Context,
	req agentbudgetports.MoneyAllowanceRequest,
) (agenttooltrust.MoneyAllowance, error) {
	b.mu.Lock()
	b.asked = append(b.asked, req)
	b.mu.Unlock()
	if b.Err != nil {
		return agenttooltrust.MoneyAllowance{}, b.Err
	}

	return agenttooltrust.MoneyAllowance{
		Limits:   b.Limits,
		Currency: StubFunctionalCurrency,
		Run:      req.Unrecorded,
		Day:      b.Used.Plus(req.Unrecorded),
	}, nil
}

func (b *StubMoneyBudgets) Asked() []agentbudgetports.MoneyAllowanceRequest {
	b.mu.Lock()
	defer b.mu.Unlock()

	return append([]agentbudgetports.MoneyAllowanceRequest(nil), b.asked...)
}

func (b *StubMoneyBudgets) CheckTool(
	context.Context,
	agentbudgetports.CheckToolBudgetRequest,
) (agentbudgetports.BudgetRefusal, error) {
	return agentbudgetports.BudgetRefusal{}, nil
}

func (b *StubMoneyBudgets) WithinBusinessHours(context.Context, *agentdefinition.Definition) bool {
	return true
}
