package agenttoolpolicy_test

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func moneyPolicy(bounds func(context.Context, agentports.ToolExecuteParams) agentports.MoneyBound) agentports.ToolPolicy {
	policy := agentports.ToolPolicy{
		Name:          "post_customer_payment",
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceCustomerPayment,
		Operation:     permission.OpCreate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierAutoExecute,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressMoney},
		Effect:        agent.ToolEffectChange,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     "Records a customer payment.",
	}
	if bounds != nil {
		policy.Condition = &agentports.TierCondition{
			Description: "Runs on its own only within bounds.",
			Bounds:      bounds,
		}
	}

	return policy
}

const functional = "USD"

func boundAt(amountMinor int64) func(context.Context, agentports.ToolExecuteParams) agentports.MoneyBound {
	return boundIn(amountMinor, functional)
}

func boundIn(
	amountMinor int64,
	currency string,
	reread ...agent.TaintSource,
) func(context.Context, agentports.ToolExecuteParams) agentports.MoneyBound {
	return func(context.Context, agentports.ToolExecuteParams) agentports.MoneyBound {
		return agentports.WithinMoneyBound(amountMinor, currency).Rereading(reread...)
	}
}

func outside(held string) func(context.Context, agentports.ToolExecuteParams) agentports.MoneyBound {
	return func(context.Context, agentports.ToolExecuteParams) agentports.MoneyBound {
		return agentports.OutsideMoneyBound(held)
	}
}

func provenWithin(policy agentports.ToolPolicy) agentports.ToolPolicy {
	condition := *policy.Condition
	condition.Bounds = boundAt(1)
	policy.Condition = &condition

	return policy
}

func limits() agent.MoneyLimits {
	return agent.MoneyLimits{
		CallCeilingMinor: 100_000,
		RunCalls:         3,
		RunAmountMinor:   200_000,
		DayCalls:         10,
		DayAmountMinor:   500_000,
	}
}

func allowance(run, day agent.MoneyUse) agenttoolpolicy.MoneyAllowanceReader {
	return allowanceWithin(limits(), run, day)
}

func allowanceWithin(
	bounds agent.MoneyLimits,
	run, day agent.MoneyUse,
) agenttoolpolicy.MoneyAllowanceReader {
	return func(context.Context) (agent.MoneyAllowance, error) {
		return agent.MoneyAllowance{Limits: bounds, Currency: functional, Run: run, Day: day}, nil
	}
}

func moneyInput(
	policy agentports.ToolPolicy,
	reader agenttoolpolicy.MoneyAllowanceReader,
) agenttoolpolicy.DecideInput {
	return agenttoolpolicy.DecideInput{
		Policy:         policy,
		Params:         call(agentPrincipal(), nil),
		Definition:     definition(agent.TierAutoExecute, nil),
		Unattended:     true,
		Taint:          clean(),
		MoneyAllowance: reader,
	}
}

func TestDecide_MoneyRunsOnItsOwnOnlyWithinBounds(t *testing.T) {
	t.Parallel()

	none := agent.MoneyUse{}
	cases := []struct {
		name   string
		in     agenttoolpolicy.DecideInput
		tier   agent.AutonomyTier
		heldBy []string
	}{
		{
			name:   "a money tool that proves nothing asks first",
			in:     moneyInput(moneyPolicy(nil), allowance(none, none)),
			tier:   agent.TierActWithApproval,
			heldBy: []string{agent.HeldByMoneyUnbounded},
		},
		{
			name: "a call within every bound runs",
			in:   moneyInput(moneyPolicy(boundAt(50_000)), allowance(none, none)),
			tier: agent.TierAutoExecute,
		},
		{
			name: "a refund within the ceiling runs",
			in:   moneyInput(moneyPolicy(boundAt(-50_000)), allowance(none, none)),
			tier: agent.TierAutoExecute,
		},
		{
			name:   "a call its condition does not prove asks first",
			in:     moneyInput(moneyPolicy(outside(agent.HeldByMoneyExactMatch)), allowance(none, none)),
			tier:   agent.TierActWithApproval,
			heldBy: []string{agent.HeldByMoneyExactMatch},
		},
		{
			name:   "a condition naming no known bound is held as unproven",
			in:     moneyInput(moneyPolicy(outside("whatever")), allowance(none, none)),
			tier:   agent.TierActWithApproval,
			heldBy: []string{agent.HeldByMoneyUnproven},
		},
		{
			name:   "limits nobody could read hold the call",
			in:     moneyInput(moneyPolicy(boundAt(1)), nil),
			tier:   agent.TierActWithApproval,
			heldBy: []string{agent.HeldByMoneyLimitsUnread},
		},
		{
			name: "limits that fail to read hold the call",
			in: moneyInput(moneyPolicy(boundAt(1)), func(context.Context) (agent.MoneyAllowance, error) {
				return agent.MoneyAllowance{}, errors.New("database down")
			}),
			tier:   agent.TierActWithApproval,
			heldBy: []string{agent.HeldByMoneyLimitsUnread},
		},
		{
			name:   "a call past the ceiling asks first",
			in:     moneyInput(moneyPolicy(boundAt(100_001)), allowance(none, none)),
			tier:   agent.TierActWithApproval,
			heldBy: []string{agent.HeldByMoneyCallCeiling},
		},
		{
			name:   "a refund past the ceiling asks first",
			in:     moneyInput(moneyPolicy(boundAt(-100_001)), allowance(none, none)),
			tier:   agent.TierActWithApproval,
			heldBy: []string{agent.HeldByMoneyCallCeiling},
		},
		{
			name: "a call past the run's count asks first",
			in: moneyInput(moneyPolicy(boundAt(1)),
				allowance(agent.MoneyUse{Calls: 3, AmountMinor: 3}, none)),
			tier:   agent.TierActWithApproval,
			heldBy: []string{agent.HeldByMoneyRunCalls},
		},
		{
			name: "a call past the run's total asks first",
			in: moneyInput(moneyPolicy(boundAt(50_001)),
				allowance(agent.MoneyUse{Calls: 2, AmountMinor: 150_000}, none)),
			tier:   agent.TierActWithApproval,
			heldBy: []string{agent.HeldByMoneyRunAmount},
		},
		{
			name: "a call past the day's count asks first",
			in: moneyInput(moneyPolicy(boundAt(1)),
				allowance(none, agent.MoneyUse{Calls: 10, AmountMinor: 10})),
			tier:   agent.TierActWithApproval,
			heldBy: []string{agent.HeldByMoneyDayCalls},
		},
		{
			name: "a call past the day's total asks first",
			in: moneyInput(moneyPolicy(boundAt(50_000)),
				allowance(none, agent.MoneyUse{Calls: 9, AmountMinor: 450_001})),
			tier:   agent.TierActWithApproval,
			heldBy: []string{agent.HeldByMoneyDayAmount},
		},
		{
			name:   "a call in another currency asks first",
			in:     moneyInput(moneyPolicy(boundIn(1, "CAD")), allowance(none, none)),
			tier:   agent.TierActWithApproval,
			heldBy: []string{agent.HeldByMoneyCurrency},
		},
		{
			name:   "a call that names no currency asks first",
			in:     moneyInput(moneyPolicy(boundIn(1, "")), allowance(none, none)),
			tier:   agent.TierActWithApproval,
			heldBy: []string{agent.HeldByMoneyCurrency},
		},
		{
			name: "the functional currency is matched whatever its case",
			in:   moneyInput(moneyPolicy(boundIn(1, "usd")), allowance(none, none)),
			tier: agent.TierAutoExecute,
		},
		{
			name: "a ceiling of zero holds even a call that moves nothing",
			in: moneyInput(moneyPolicy(boundAt(0)), allowanceWithin(
				agent.MoneyLimits{
					RunCalls:       3,
					RunAmountMinor: 200_000,
					DayCalls:       10,
					DayAmountMinor: 500_000,
				}, none, none)),
			tier:   agent.TierActWithApproval,
			heldBy: []string{agent.HeldByMoneyCallCeiling},
		},
		{
			name: "a call landing exactly on every cap runs",
			in: moneyInput(moneyPolicy(boundAt(50_000)), allowance(
				agent.MoneyUse{Calls: 2, AmountMinor: 150_000},
				agent.MoneyUse{Calls: 9, AmountMinor: 450_000},
			)),
			tier: agent.TierAutoExecute,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			decision := agenttoolpolicy.Decide(t.Context(), tc.in)
			assert.Equal(t, tc.tier, decision.Tier)
			assert.Equal(t, agent.EgressMoney, decision.Egress)
			if tc.tier == agent.TierAutoExecute {
				require.NotNil(t, decision.MoneyMinor)
				assert.NotEmpty(t, decision.MoneyCurrency)
				assert.Empty(t, decision.HeldBy)
				return
			}
			assert.Nil(t, decision.MoneyMinor)
			assert.Empty(t, decision.MoneyCurrency)
			assert.Equal(t, tc.heldBy, decision.HeldBy)
		})
	}
}

func TestDecide_MoneyBoundsAreNotReadForACallAlreadyHeld(t *testing.T) {
	t.Parallel()

	unread := func(context.Context, agentports.ToolExecuteParams) agentports.MoneyBound {
		t.Error("bounds were read for a call that could not run on its own")
		return agentports.MoneyBound{}
	}
	noAllowance := func(context.Context) (agent.MoneyAllowance, error) {
		t.Error("the allowance was read for a call that could not run on its own")
		return agent.MoneyAllowance{}, nil
	}

	taintedRun := moneyInput(moneyPolicy(unread), noAllowance)
	taintedRun.Taint = tainted()
	decision := agenttoolpolicy.Decide(t.Context(), taintedRun)
	assert.Equal(t, agent.TierActWithApproval, decision.Tier)
	assert.Contains(t, decision.HeldBy, agenttoolpolicy.HeldByTainted)
	assert.Nil(t, decision.MoneyMinor)

	unknownTaint := moneyInput(moneyPolicy(unread), noAllowance)
	unknownTaint.Taint = nil
	decision = agenttoolpolicy.Decide(t.Context(), unknownTaint)
	assert.Equal(t, agent.TierActWithApproval, decision.Tier)
	assert.Contains(t, decision.HeldBy, agenttoolpolicy.HeldByTainted)

	proposing := moneyInput(moneyPolicy(unread), noAllowance)
	proposing.Definition = definition(agent.TierAutoExecute, map[string]agent.AutonomyTier{
		"post_customer_payment": agent.TierPropose,
	})
	decision = agenttoolpolicy.Decide(t.Context(), proposing)
	assert.Equal(t, agent.TierPropose, decision.Tier)
	assert.NotContains(t, decision.HeldBy, agent.HeldByMoneyUnproven)
}

func TestDecide_ABoundedMoneyCallHeldByItsOwnLimitRecordsNoAmount(t *testing.T) {
	t.Parallel()

	policy := moneyPolicy(boundAt(10))
	policy.Condition.Limit = func(context.Context, agentports.ToolExecuteParams) agent.AutonomyTier {
		return agent.TierPropose
	}

	decision := agenttoolpolicy.Decide(t.Context(), moneyInput(policy, allowance(agent.MoneyUse{}, agent.MoneyUse{})))
	assert.Equal(t, agent.TierPropose, decision.Tier)
	assert.Nil(t, decision.MoneyMinor)
	assert.Equal(t, []string{agenttoolpolicy.HeldByCondition}, decision.HeldBy)
}

func TestPromotable_ABoundedMoneyToolMayEarnAutomatic(t *testing.T) {
	t.Parallel()

	assert.Equal(t, agent.TierAutoExecute, agenttoolpolicy.Promotable(moneyPolicy(boundAt(1))))
	assert.Equal(t, agent.TierActWithApproval, agenttoolpolicy.Promotable(moneyPolicy(nil)))
}

func receiptRead() *agent.RunTaint {
	taint := &agent.RunTaint{}
	taint.Add(agent.TaintMark{
		Source:   agent.TaintSourceBankReceipt,
		ToolName: "get_bank_receipt",
		CallID:   "call_receipt",
	})

	return taint
}

func receiptAndWebRead() *agent.RunTaint {
	taint := receiptRead()
	taint.Add(agent.TaintMark{Source: agent.TaintSourceWeb, ToolName: "web_search", CallID: "call_web"})

	return taint
}

func vouchingPolicy(
	bounds func(context.Context, agentports.ToolExecuteParams) agentports.MoneyBound,
) agentports.ToolPolicy {
	policy := moneyPolicy(bounds)
	policy.Condition.Vouches = []agent.TaintSource{agent.TaintSourceBankReceipt}

	return policy
}

func TestDecide_ABoundThatRereadsTheReceiptVouchesForHavingReadIt(t *testing.T) {
	t.Parallel()

	none := agent.MoneyUse{}
	rereads := boundIn(40_000, functional, agent.TaintSourceBankReceipt)
	cases := []struct {
		name   string
		policy agentports.ToolPolicy
		taint  *agent.RunTaint
		use    agent.MoneyUse
		tier   agent.AutonomyTier
		heldBy []string
	}{
		{
			name:   "reading the receipt and then posting runs within the limits",
			policy: vouchingPolicy(rereads),
			taint:  receiptRead(),
			tier:   agent.TierAutoExecute,
		},
		{
			name:   "reading the receipt and the web holds the post",
			policy: vouchingPolicy(rereads),
			taint:  receiptAndWebRead(),
			tier:   agent.TierActWithApproval,
			heldBy: []string{agenttoolpolicy.HeldByTainted},
		},
		{
			name:   "a tool that does not vouch for receipts stays held",
			policy: moneyPolicy(rereads),
			taint:  receiptRead(),
			tier:   agent.TierActWithApproval,
			heldBy: []string{agenttoolpolicy.HeldByTainted},
		},
		{
			name:   "a bound that did not re-read the receipt does not vouch for it",
			policy: vouchingPolicy(boundAt(40_000)),
			taint:  receiptRead(),
			tier:   agent.TierActWithApproval,
			heldBy: []string{agenttoolpolicy.HeldByTainted},
		},
		{
			name:   "a run whose taint is unknown is never vouched for",
			policy: vouchingPolicy(rereads),
			taint:  nil,
			tier:   agent.TierActWithApproval,
			heldBy: []string{agenttoolpolicy.HeldByTainted},
		},
		{
			name:   "a vouched call still keeps to the run's caps",
			policy: vouchingPolicy(rereads),
			taint:  receiptRead(),
			use:    agent.MoneyUse{Calls: 3, AmountMinor: 3},
			tier:   agent.TierActWithApproval,
			heldBy: []string{agent.HeldByMoneyRunCalls, agenttoolpolicy.HeldByTainted},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			in := moneyInput(tc.policy, allowance(tc.use, none))
			in.Taint = tc.taint
			decision := agenttoolpolicy.Decide(t.Context(), in)

			assert.Equal(t, tc.tier, decision.Tier)
			if tc.tier == agent.TierAutoExecute {
				require.NotNil(t, decision.MoneyMinor)
				assert.Equal(t, int64(40_000), *decision.MoneyMinor)
				assert.Empty(t, decision.HeldBy)
				return
			}
			assert.Nil(t, decision.MoneyMinor)
			assert.Equal(t, tc.heldBy, decision.HeldBy)
		})
	}
}

func TestDecide_ABoundIsNotReadWhenTheRunHoldsContentItCannotVouchFor(t *testing.T) {
	t.Parallel()

	unread := func(context.Context, agentports.ToolExecuteParams) agentports.MoneyBound {
		t.Error("bounds were read for a call held by content they cannot re-read")
		return agentports.MoneyBound{}
	}

	in := moneyInput(vouchingPolicy(unread), allowance(agent.MoneyUse{}, agent.MoneyUse{}))
	in.Taint = receiptAndWebRead()
	decision := agenttoolpolicy.Decide(t.Context(), in)
	assert.Equal(t, agent.TierActWithApproval, decision.Tier)
	assert.Equal(t, []string{agenttoolpolicy.HeldByTainted}, decision.HeldBy)
}
