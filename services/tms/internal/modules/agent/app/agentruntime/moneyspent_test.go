package agentruntime

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func paymentTurns(count int) []*llmports.ChatCompletionResult {
	turns := make([]*llmports.ChatCompletionResult, 0, count)
	for range count {
		turns = append(
			turns,
			runtimetest.ToolTurn(
				"apply_payment",
				map[string]any{"invoiceId": pulid.MustNew("inv_").String()},
			),
		)
	}

	return turns
}

func tenPaymentLimits() agenttooltrust.MoneyLimits {
	limits := roomyMoneyLimits()
	limits.RunCalls = 10

	return limits
}

func delegatedPayments(
	definition *agentdefinition.Definition,
	spent agenttooltrust.MoneyUse,
) *agentruntimeports.RunRequest {
	return &agentruntimeports.RunRequest{
		Definition: definition,
		Actor:      runtimetest.Actor(),
		Input:      "Apply one more payment.",
		Delegation: &agentruntimeports.Delegation{
			ParentAgentID:   pulid.MustNew("agdef_"),
			ParentAgentName: "Billing",
			CallID:          "call_delegate",
			StepScope:       "scope",
			MoneySpent:      spent,
		},
		Taint: &agent.RunTaint{},
	}
}

func executedPayments(count int, amountMinor int64) []agentruntimeports.PendingAction {
	actions := make([]agentruntimeports.PendingAction, 0, count)
	for range count {
		amount := amountMinor
		actions = append(actions, agentruntimeports.PendingAction{
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

	builder := runtimetest.ReportBuilder()
	turns := append(paymentTurns(10),
		delegateTurn(builder.ID, "Apply the last payment."),
		runtimetest.TextTurn("Done."))
	pay := runtimetest.MoneyTool()
	rt := newRuntime(&scriptedCompletion{Turns: turns}, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agenttoolports.AgentTool{pay}}, nil)
	rewire(rt, func(p *Params) { p.Budgets = &moneyBudgets{Limits: tenPaymentLimits()} })

	req := delegatingRequest(builder)
	req.Definition = runtimetest.AutoDefinition("apply_payment")
	req.Taint = &agent.RunTaint{}
	fx := &delegateRecorder{run: scriptedDelegateRun(builder)}
	driveWith(t, rt, req, fx)

	assert.Equal(t, 10, pay.Calls)
	require.Len(t, fx.calls, 1)
	assert.Equal(
		t,
		agenttooltrust.MoneyUse{Calls: 10, AmountMinor: 100_000},
		fx.calls[0].MoneySpent,
		"the delegate's caps start from what the turn moved",
	)
}

func TestRun_ADelegateHandedTenPaymentsIsHeldAtTheEleventh(t *testing.T) {
	t.Parallel()

	pay := runtimetest.MoneyTool()
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("apply_payment", map[string]any{"invoiceId": "inv_11"}),
		runtimetest.TextTurn("Filed."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agenttoolports.AgentTool{pay}}, nil)
	budgets := &moneyBudgets{Limits: tenPaymentLimits()}
	rewire(rt, func(p *Params) { p.Budgets = budgets })

	run := runTainted(t, rt, delegatedPayments(
		runtimetest.AutoDefinition("apply_payment"),
		agenttooltrust.MoneyUse{Calls: 10, AmountMinor: 100_000},
	))

	assert.Zero(t, pay.Calls)
	require.Len(t, run.result.Actions, 1)
	assert.Equal(t, []string{agenttooltrust.HeldByMoneyRunCalls}, run.result.Actions[0].HeldBy)
	require.Len(t, budgets.Asked(), 1)
	assert.Equal(
		t,
		agenttooltrust.MoneyUse{Calls: 10, AmountMinor: 100_000},
		budgets.Asked()[0].Unrecorded,
	)
}

func TestRun_ADelegateIsHeldWhereTheTurnAndItTogetherPassTheRunTotal(t *testing.T) {
	t.Parallel()

	pay := runtimetest.MoneyTool()
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("apply_payment", map[string]any{"invoiceId": "inv_1"}),
		runtimetest.TextTurn("Filed."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agenttoolports.AgentTool{pay}}, nil)
	limits := roomyMoneyLimits()
	limits.RunAmountMinor = 50_000
	rewire(rt, func(p *Params) { p.Budgets = &moneyBudgets{Limits: limits} })

	run := runTainted(t, rt, delegatedPayments(
		runtimetest.AutoDefinition("apply_payment"),
		agenttooltrust.MoneyUse{Calls: 1, AmountMinor: 45_000},
	))

	assert.Zero(t, pay.Calls, "10,000 more would take the two past 50,000")
	require.Len(t, run.result.Actions, 1)
	assert.Equal(t, []string{agenttooltrust.HeldByMoneyRunAmount}, run.result.Actions[0].HeldBy)
}

func TestRun_WhatADelegateMovedCountsAgainstTheTurnAfterIt(t *testing.T) {
	t.Parallel()

	builder := runtimetest.ReportBuilder()
	pay := runtimetest.MoneyTool()
	rt := newRuntime(&scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		delegateTurn(builder.ID, "Apply ten payments."),
		runtimetest.ToolTurn("apply_payment", map[string]any{"invoiceId": "inv_11"}),
		runtimetest.TextTurn("Filed the last one."),
	}}, &stubQueryRegistry{}, &stubActionRegistry{Tools: []agenttoolports.AgentTool{pay}}, nil)
	budgets := &moneyBudgets{Limits: tenPaymentLimits()}
	rewire(rt, func(p *Params) { p.Budgets = budgets })

	delegated := scriptedDelegateRun(builder)
	delegated.Result.Actions = executedPayments(10, 10_000)
	delegated.Result.Taint = &agent.RunTaint{}
	req := delegatingRequest(builder)
	req.Definition = runtimetest.AutoDefinition("apply_payment")
	req.Taint = &agent.RunTaint{}
	result := driveWith(t, rt, req, &delegateRecorder{run: delegated})

	assert.Zero(t, pay.Calls)
	require.Len(t, result.Actions, 1)
	assert.Equal(t, []string{agenttooltrust.HeldByMoneyRunCalls}, result.Actions[0].HeldBy)
	require.Len(t, budgets.Asked(), 1)
	assert.Equal(
		t,
		agenttooltrust.MoneyUse{Calls: 10, AmountMinor: 100_000},
		budgets.Asked()[0].Unrecorded,
	)
}

func TestRun_AFailedMoneyWriteMovedNothingAndCountsAgainstNothing(t *testing.T) {
	t.Parallel()

	failing := runtimetest.MoneyTool()
	failing.Err = errors.New("the invoice is closed")
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("apply_payment", map[string]any{"invoiceId": "inv_1"}),
		runtimetest.ToolTurn("apply_payment", map[string]any{"invoiceId": "inv_2"}),
		runtimetest.TextTurn("Tried twice."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agenttoolports.AgentTool{failing}}, nil)
	limits := roomyMoneyLimits()
	limits.RunCalls = 1
	budgets := &moneyBudgets{Limits: limits}
	rewire(rt, func(p *Params) { p.Budgets = budgets })

	run := runTainted(t, rt, &agentruntimeports.RunRequest{
		Definition: runtimetest.AutoDefinition("apply_payment"),
		Actor:      runtimetest.Actor(),
		Input:      "Apply both payments.",
	})

	assert.Equal(t, 2, failing.Calls, "the failed write left the run's one change unspent")
	require.Len(t, budgets.Asked(), 2)
	assert.Equal(t, agenttooltrust.MoneyUse{}, budgets.Asked()[1].Unrecorded)
	require.Len(t, run.result.Actions, 2)
	assert.NotEmpty(t, run.result.Actions[0].ExecutionError)
}

func TestRun_ASimulatedRunIsHeldWhereTheRealRunWouldBe(t *testing.T) {
	t.Parallel()

	pay := runtimetest.MoneyTool()
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("apply_payment", map[string]any{"invoiceId": "inv_1"}),
		runtimetest.ToolTurn("apply_payment", map[string]any{"invoiceId": "inv_2"}),
		runtimetest.TextTurn("Simulated."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agenttoolports.AgentTool{pay}}, nil)
	limits := roomyMoneyLimits()
	limits.RunCalls = 1
	rewire(rt, func(p *Params) { p.Budgets = &moneyBudgets{Limits: limits} })

	definition := runtimetest.AutoDefinition("apply_payment")
	definition.SimulationMode = true
	run := runTainted(t, rt, &agentruntimeports.RunRequest{
		Definition: definition,
		Actor:      runtimetest.Actor(),
		Input:      "Apply both payments.",
	})

	assert.Zero(t, pay.Calls, "a simulation changes nothing")
	require.Len(t, run.result.Actions, 2)
	assert.True(t, run.result.Actions[0].Simulated)
	assert.False(t, run.result.Actions[1].Simulated)
	assert.Equal(t, []string{agenttooltrust.HeldByMoneyRunCalls}, run.result.Actions[1].HeldBy)
}
