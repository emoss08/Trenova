package agenttest

import (
	"context"
	"sync"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

type StubMoneyBudgets struct {
	agentports.AgentBudgetService

	Limits agenttooltrust.MoneyLimits
	Used   agenttooltrust.MoneyUse
	Err    error

	mu    sync.Mutex
	asked []agentports.MoneyAllowanceRequest
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
	req agentports.MoneyAllowanceRequest,
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

func (b *StubMoneyBudgets) Asked() []agentports.MoneyAllowanceRequest {
	b.mu.Lock()
	defer b.mu.Unlock()

	return append([]agentports.MoneyAllowanceRequest(nil), b.asked...)
}

func (b *StubMoneyBudgets) CheckTool(
	context.Context,
	agentports.CheckToolBudgetRequest,
) (agentports.BudgetRefusal, error) {
	return agentports.BudgetRefusal{}, nil
}

func (b *StubMoneyBudgets) WithinBusinessHours(context.Context, *agentdefinition.Definition) bool {
	return true
}
