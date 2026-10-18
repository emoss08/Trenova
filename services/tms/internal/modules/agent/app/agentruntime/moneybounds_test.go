package agentruntime

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type moneyBudgets struct {
	agentports.AgentBudgetService

	limits agent.MoneyLimits
	used   agent.MoneyUse
	err    error
	asked  []agentports.MoneyAllowanceRequest
}

func (b *moneyBudgets) MoneyAllowance(
	_ context.Context,
	req agentports.MoneyAllowanceRequest,
) (agent.MoneyAllowance, error) {
	b.asked = append(b.asked, req)
	if b.err != nil {
		return agent.MoneyAllowance{}, b.err
	}

	return agent.MoneyAllowance{
		Limits: b.limits,
		Run:    req.Unrecorded,
		Day:    b.used.Plus(req.Unrecorded),
	}, nil
}

func (b *moneyBudgets) CheckTool(
	context.Context,
	agentports.CheckToolBudgetRequest,
) (agentports.BudgetRefusal, error) {
	return agentports.BudgetRefusal{}, nil
}

func (b *moneyBudgets) WithinBusinessHours(context.Context, *agentdefinition.Definition) bool {
	return true
}

func roomyMoneyLimits() agent.MoneyLimits {
	return agent.MoneyLimits{
		CallCeilingMinor: 100_000,
		RunCalls:         10,
		RunAmountMinor:   1_000_000,
		DayCalls:         100,
		DayAmountMinor:   10_000_000,
	}
}

func TestRun_AMoneyWritePastTheCeilingIsFiledNamingTheBound(t *testing.T) {
	t.Parallel()

	pay := moneyTool()
	completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
		toolTurn("apply_payment", map[string]any{"invoiceId": "inv_1"}),
		textTurn("Filed."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agentports.AgentTool{pay}}, nil)
	limits := roomyMoneyLimits()
	limits.CallCeilingMinor = 9_999
	rt.budgets = &moneyBudgets{limits: limits}

	run := runTainted(t, rt, &agentports.RunRequest{
		Definition: autoDefinition("apply_payment"),
		Actor:      testActor(),
		Input:      "Apply the payment.",
	})

	assert.Zero(t, pay.Calls, "a write past its bounds does not run")
	require.Len(t, run.result.Actions, 1)
	action := run.result.Actions[0]
	assert.False(t, action.Executed)
	assert.Equal(t, agent.TierActWithApproval, action.Tier)
	assert.Equal(t, []string{agent.HeldByMoneyCallCeiling}, action.HeldBy)
	assert.Nil(t, action.MoneyMinor)
}

func TestRun_WritesEarlierInTheTurnCountAgainstTheRunCap(t *testing.T) {
	t.Parallel()

	pay := moneyTool()
	completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
		toolTurn("apply_payment", map[string]any{"invoiceId": "inv_1"}),
		toolTurn("apply_payment", map[string]any{"invoiceId": "inv_2"}),
		textTurn("Applied one."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agentports.AgentTool{pay}}, nil)
	limits := roomyMoneyLimits()
	limits.RunCalls = 1
	budgets := &moneyBudgets{limits: limits}
	rt.budgets = budgets

	run := runTainted(t, rt, &agentports.RunRequest{
		Definition: autoDefinition("apply_payment"),
		Actor:      testActor(),
		Input:      "Apply both payments.",
	})

	assert.Equal(t, 1, pay.Calls)
	require.Len(t, run.result.Actions, 2)
	assert.True(t, run.result.Actions[0].Executed)
	assert.False(t, run.result.Actions[1].Executed)
	assert.Equal(t, []string{agent.HeldByMoneyRunCalls}, run.result.Actions[1].HeldBy)
	require.Len(t, budgets.asked, 2)
	assert.Equal(t, agent.MoneyUse{Calls: 1, AmountMinor: 10_000}, budgets.asked[1].Unrecorded)
}

func TestRun_MoneyLimitsThatCannotBeReadHoldTheWrite(t *testing.T) {
	t.Parallel()

	for name, budgets := range map[string]agentports.AgentBudgetService{
		"unreadable": &moneyBudgets{err: errors.New("database down")},
		"absent":     nil,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			pay := moneyTool()
			completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
				toolTurn("apply_payment", map[string]any{"invoiceId": "inv_1"}),
				textTurn("Filed."),
			}}
			rt := newRuntime(completion, &stubQueryRegistry{},
				&stubActionRegistry{Tools: []agentports.AgentTool{pay}}, nil)
			rt.budgets = budgets

			run := runTainted(t, rt, &agentports.RunRequest{
				Definition: autoDefinition("apply_payment"),
				Actor:      testActor(),
				Input:      "Apply the payment.",
			})

			assert.Zero(t, pay.Calls)
			require.Len(t, run.result.Actions, 1)
			assert.Equal(t, []string{agent.HeldByMoneyLimitsUnread}, run.result.Actions[0].HeldBy)
		})
	}
}

func TestRun_AMoneyToolThatProvesNothingAsksFirst(t *testing.T) {
	t.Parallel()

	pay := moneyTool()
	pay.MoneyMinor = nil
	completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
		toolTurn("apply_payment", map[string]any{"invoiceId": "inv_1"}),
		textTurn("Filed."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agentports.AgentTool{pay}}, nil)
	rt.budgets = &moneyBudgets{limits: roomyMoneyLimits()}

	run := runTainted(t, rt, &agentports.RunRequest{
		Definition: autoDefinition("apply_payment"),
		Actor:      testActor(),
		Input:      "Apply the payment.",
	})

	assert.Zero(t, pay.Calls)
	require.Len(t, run.result.Actions, 1)
	assert.Equal(t, agent.TierActWithApproval, run.result.Actions[0].Tier)
	assert.Equal(t, []string{agent.HeldByMoneyUnbounded}, run.result.Actions[0].HeldBy)
}

func TestHeldReason_ExplainsEveryMoneyBound(t *testing.T) {
	t.Parallel()

	for _, key := range agent.MoneyHeldByKeys() {
		reason := heldReason([]string{key}, agent.EgressMoney)
		assert.NotEmpty(t, reason, key)
		assert.Contains(t, reason, moneyHeldReasons[key], key)
	}
}
