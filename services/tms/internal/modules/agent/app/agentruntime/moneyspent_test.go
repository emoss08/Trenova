package agentruntime

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func paymentTurns(count int) []*agentports.ChatCompletionResult {
	turns := make([]*agentports.ChatCompletionResult, 0, count)
	for range count {
		turns = append(turns, toolTurn("apply_payment", map[string]any{"invoiceId": pulid.MustNew("inv_").String()}))
	}

	return turns
}

func tenPaymentLimits() agent.MoneyLimits {
	limits := roomyMoneyLimits()
	limits.RunCalls = 10

	return limits
}

func delegatedPayments(definition *agentdefinition.Definition, spent agent.MoneyUse) *agentports.RunRequest {
	return &agentports.RunRequest{
		Definition: definition,
		Actor:      testActor(),
		Input:      "Apply one more payment.",
		Delegation: &agentports.Delegation{
			ParentAgentID:   pulid.MustNew("agdef_"),
			ParentAgentName: "Billing",
			CallID:          "call_delegate",
			StepScope:       "scope",
			MoneySpent:      spent,
		},
		Taint: &agent.RunTaint{},
	}
}

func executedPayments(count int, amountMinor int64) []agentports.PendingAction {
	actions := make([]agentports.PendingAction, 0, count)
	for range count {
		amount := amountMinor
		actions = append(actions, agentports.PendingAction{
			ToolName:      "apply_payment",
			Tier:          agent.TierAutoExecute,
			Executed:      true,
			Egress:        agent.EgressMoney,
			MoneyMinor:    &amount,
			MoneyCurrency: agenttest.StubFunctionalCurrency,
		})
	}

	return actions
}

func TestRun_TheDelegateIsHandedWhatTheTurnHasAlreadyMoved(t *testing.T) {
	t.Parallel()

	builder := reportBuilder()
	turns := append(paymentTurns(10),
		delegateTurn(builder.ID, "Apply the last payment."),
		textTurn("Done."))
	pay := moneyTool()
	rt := newRuntime(&scriptedCompletion{Turns: turns}, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agentports.AgentTool{pay}}, nil)
	rt.budgets = &moneyBudgets{Limits: tenPaymentLimits()}

	req := delegatingRequest(builder)
	req.Definition = autoDefinition("apply_payment")
	req.Taint = &agent.RunTaint{}
	fx := &delegateRecorder{run: scriptedDelegateRun(builder)}
	driveWith(t, rt, req, fx)

	assert.Equal(t, 10, pay.Calls)
	require.Len(t, fx.calls, 1)
	assert.Equal(t, agent.MoneyUse{Calls: 10, AmountMinor: 100_000}, fx.calls[0].MoneySpent,
		"the delegate's caps start from what the turn moved")
}

func TestRun_ADelegateHandedTenPaymentsIsHeldAtTheEleventh(t *testing.T) {
	t.Parallel()

	pay := moneyTool()
	completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
		toolTurn("apply_payment", map[string]any{"invoiceId": "inv_11"}),
		textTurn("Filed."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agentports.AgentTool{pay}}, nil)
	budgets := &moneyBudgets{Limits: tenPaymentLimits()}
	rt.budgets = budgets

	run := runTainted(t, rt, delegatedPayments(
		autoDefinition("apply_payment"),
		agent.MoneyUse{Calls: 10, AmountMinor: 100_000},
	))

	assert.Zero(t, pay.Calls)
	require.Len(t, run.result.Actions, 1)
	assert.Equal(t, []string{agent.HeldByMoneyRunCalls}, run.result.Actions[0].HeldBy)
	require.Len(t, budgets.Asked(), 1)
	assert.Equal(t, agent.MoneyUse{Calls: 10, AmountMinor: 100_000}, budgets.Asked()[0].Unrecorded)
}

func TestRun_ADelegateIsHeldWhereTheTurnAndItTogetherPassTheRunTotal(t *testing.T) {
	t.Parallel()

	pay := moneyTool()
	completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
		toolTurn("apply_payment", map[string]any{"invoiceId": "inv_1"}),
		textTurn("Filed."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agentports.AgentTool{pay}}, nil)
	limits := roomyMoneyLimits()
	limits.RunAmountMinor = 50_000
	rt.budgets = &moneyBudgets{Limits: limits}

	run := runTainted(t, rt, delegatedPayments(
		autoDefinition("apply_payment"),
		agent.MoneyUse{Calls: 1, AmountMinor: 45_000},
	))

	assert.Zero(t, pay.Calls, "10,000 more would take the two past 50,000")
	require.Len(t, run.result.Actions, 1)
	assert.Equal(t, []string{agent.HeldByMoneyRunAmount}, run.result.Actions[0].HeldBy)
}

func TestRun_WhatADelegateMovedCountsAgainstTheTurnAfterIt(t *testing.T) {
	t.Parallel()

	builder := reportBuilder()
	pay := moneyTool()
	rt := newRuntime(&scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
		delegateTurn(builder.ID, "Apply ten payments."),
		toolTurn("apply_payment", map[string]any{"invoiceId": "inv_11"}),
		textTurn("Filed the last one."),
	}}, &stubQueryRegistry{}, &stubActionRegistry{Tools: []agentports.AgentTool{pay}}, nil)
	budgets := &moneyBudgets{Limits: tenPaymentLimits()}
	rt.budgets = budgets

	delegated := scriptedDelegateRun(builder)
	delegated.Result.Actions = executedPayments(10, 10_000)
	delegated.Result.Taint = &agent.RunTaint{}
	req := delegatingRequest(builder)
	req.Definition = autoDefinition("apply_payment")
	req.Taint = &agent.RunTaint{}
	result := driveWith(t, rt, req, &delegateRecorder{run: delegated})

	assert.Zero(t, pay.Calls)
	require.Len(t, result.Actions, 1)
	assert.Equal(t, []string{agent.HeldByMoneyRunCalls}, result.Actions[0].HeldBy)
	require.Len(t, budgets.Asked(), 1)
	assert.Equal(t, agent.MoneyUse{Calls: 10, AmountMinor: 100_000}, budgets.Asked()[0].Unrecorded)
}

func TestRun_AFailedMoneyWriteMovedNothingAndCountsAgainstNothing(t *testing.T) {
	t.Parallel()

	failing := moneyTool()
	failing.Err = errors.New("the invoice is closed")
	completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
		toolTurn("apply_payment", map[string]any{"invoiceId": "inv_1"}),
		toolTurn("apply_payment", map[string]any{"invoiceId": "inv_2"}),
		textTurn("Tried twice."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agentports.AgentTool{failing}}, nil)
	limits := roomyMoneyLimits()
	limits.RunCalls = 1
	budgets := &moneyBudgets{Limits: limits}
	rt.budgets = budgets

	run := runTainted(t, rt, &agentports.RunRequest{
		Definition: autoDefinition("apply_payment"),
		Actor:      testActor(),
		Input:      "Apply both payments.",
	})

	assert.Equal(t, 2, failing.Calls, "the failed write left the run's one change unspent")
	require.Len(t, budgets.Asked(), 2)
	assert.Equal(t, agent.MoneyUse{}, budgets.Asked()[1].Unrecorded)
	require.Len(t, run.result.Actions, 2)
	assert.NotEmpty(t, run.result.Actions[0].ExecutionError)
}

func TestRun_ASimulatedRunIsHeldWhereTheRealRunWouldBe(t *testing.T) {
	t.Parallel()

	pay := moneyTool()
	completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
		toolTurn("apply_payment", map[string]any{"invoiceId": "inv_1"}),
		toolTurn("apply_payment", map[string]any{"invoiceId": "inv_2"}),
		textTurn("Simulated."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agentports.AgentTool{pay}}, nil)
	limits := roomyMoneyLimits()
	limits.RunCalls = 1
	rt.budgets = &moneyBudgets{Limits: limits}

	definition := autoDefinition("apply_payment")
	definition.SimulationMode = true
	run := runTainted(t, rt, &agentports.RunRequest{
		Definition: definition,
		Actor:      testActor(),
		Input:      "Apply both payments.",
	})

	assert.Zero(t, pay.Calls, "a simulation changes nothing")
	require.Len(t, run.result.Actions, 2)
	assert.True(t, run.result.Actions[0].Simulated)
	assert.False(t, run.result.Actions[1].Simulated)
	assert.Equal(t, []string{agent.HeldByMoneyRunCalls}, run.result.Actions[1].HeldBy)
}
