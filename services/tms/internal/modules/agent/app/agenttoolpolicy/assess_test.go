package agenttoolpolicy_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/agentcontrol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func actionPolicy(name string, egress ...agent.EgressClass) agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:        name,
		Kind:        agenttooltrust.ToolKindAction,
		Scope:       agenttooltrust.ToolScopeTenant,
		DefaultTier: agent.TierAutoExecute,
		MaxTier:     agent.TierAutoExecute,
		Egress:      egress,
		Effect:      agenttooltrust.ToolEffectChange,
	}
}

func chatAgent(
	ceiling agent.AutonomyTier,
	tiers map[string]agent.AutonomyTier,
) *agentdefinition.Definition {
	return &agentdefinition.Definition{
		AutonomyCeiling: ceiling,
		ToolTiers:       tiers,
		TriggerMode:     agentdefinition.TriggerChat,
	}
}

func scheduledAgent(
	ceiling agent.AutonomyTier,
	tiers map[string]agent.AutonomyTier,
) *agentdefinition.Definition {
	definition := chatAgent(ceiling, tiers)
	definition.TriggerMode = agentdefinition.TriggerScheduled

	return definition
}

func assessBoth(
	t *testing.T,
	in agenttoolpolicy.AssessInput,
) (agenttoolports.ToolAutonomy, agenttoolports.ToolAutonomy) {
	t.Helper()

	in.Tainted = false
	clean := agenttoolpolicy.Assess(t.Context(), &in)
	in.Tainted = true
	tainted := agenttoolpolicy.Assess(t.Context(), &in)

	return clean, tainted
}

/*
A payment an agent may post on its own still waits for a person once the run
has read an email: money is not held by its class, only by what the run read.
*/
func TestAssess_BoundedMoneyRunsOnItsOwnWithinBoundsUntilTheRunIsTainted(t *testing.T) {
	t.Parallel()

	policy := actionPolicy("receivable_post_customer_payment", agent.EgressMoney)
	policy.Condition = &agenttoolports.TierCondition{
		Description: "Runs on its own only within bounds.",
		Bounds: func(context.Context, agenttoolports.ToolExecuteParams) agenttoolports.MoneyBound {
			return agenttoolports.WithinMoneyBound(1, "USD")
		},
	}
	clean, tainted := assessBoth(t, agenttoolpolicy.AssessInput{
		Policy:     policy,
		Definition: chatAgent(agent.TierAutoExecute, nil),
	})

	assert.Equal(t, agenttooltrust.AutonomyConditional, clean.Answer)
	assert.Equal(t, agent.TierAutoExecute, clean.Tier)
	assert.Equal(t, []string{agenttooltrust.HeldByMoneyUnproven}, clean.HeldBy)

	assert.Equal(t, agenttooltrust.AutonomyNeedsApproval, tainted.Answer)
	assert.Equal(t, agent.TierActWithApproval, tainted.Tier)
	assert.Equal(t, []string{agenttoolpolicy.HeldByTainted}, tainted.HeldBy)
}

func TestAssess_UnboundedMoneyAsksFirst(t *testing.T) {
	t.Parallel()

	clean, tainted := assessBoth(t, agenttoolpolicy.AssessInput{
		Policy:     actionPolicy("receivable_post_customer_payment", agent.EgressMoney),
		Definition: chatAgent(agent.TierAutoExecute, nil),
	})

	for _, autonomy := range []agenttoolports.ToolAutonomy{clean, tainted} {
		assert.Equal(t, agenttooltrust.AutonomyNeedsApproval, autonomy.Answer)
		assert.Equal(t, agent.TierActWithApproval, autonomy.Tier)
		assert.Contains(t, autonomy.HeldBy, agenttooltrust.HeldByMoneyUnbounded)
	}
	assert.Contains(t, tainted.HeldBy, agenttoolpolicy.HeldByTainted)
}

func TestAssess_WorkThatLeavesNeverRunsPastApproval(t *testing.T) {
	t.Parallel()

	name := "customer_send_email"
	clean, tainted := assessBoth(t, agenttoolpolicy.AssessInput{
		Policy: actionPolicy(name, agent.EgressExternalRecipient),
		Definition: chatAgent(agent.TierAutoExecute, map[string]agent.AutonomyTier{
			name: agent.TierAutoExecute,
		}),
	})

	for _, autonomy := range []agenttoolports.ToolAutonomy{clean, tainted} {
		assert.Equal(t, agenttooltrust.AutonomyNeedsApproval, autonomy.Answer)
		assert.Equal(t, agent.TierActWithApproval, autonomy.Tier)
		assert.Contains(t, autonomy.HeldBy, agenttoolpolicy.HeldByEgressClass)
	}
}

func TestAssess_TheAgentsCeilingHoldsEveryTool(t *testing.T) {
	t.Parallel()

	clean, tainted := assessBoth(t, agenttoolpolicy.AssessInput{
		Policy:     actionPolicy("dispatch_assign_move", agent.EgressInternal),
		Definition: chatAgent(agent.TierPropose, nil),
	})

	for _, autonomy := range []agenttoolports.ToolAutonomy{clean, tainted} {
		assert.Equal(t, agenttooltrust.AutonomyProposeOnly, autonomy.Answer)
		assert.Equal(t, agent.TierPropose, autonomy.Tier)
		assert.Contains(t, autonomy.HeldBy, agenttoolpolicy.HeldByAgentCeiling)
	}
}

/*
A report is private or shared by what each call asks for. A private one runs
while its owner is in the conversation; a shared one waits at the tool's tier.
Unattended, the exemption never applies.
*/
func TestAssess_AClassifiedToolWithTheExemptionIsConditionalInChat(t *testing.T) {
	t.Parallel()

	policy := actionPolicy("report_create", agent.EgressPersonal, agent.EgressInternal)
	policy.DefaultTier = agent.TierPropose
	policy.PersonalRunsUnasked = true
	policy.Classify = func(agenttoolports.ToolExecuteParams) agenttoolports.CallPolicy {
		return agenttoolports.CallPolicy{Egress: agent.EgressInternal}
	}

	clean, _ := assessBoth(t, agenttoolpolicy.AssessInput{
		Policy:     policy,
		Definition: chatAgent(agent.TierAutoExecute, nil),
	})
	assert.Equal(t, agenttooltrust.AutonomyConditional, clean.Answer)
	assert.Equal(t, agent.TierAutoExecute, clean.Tier)
	assert.Contains(t, clean.HeldBy, agenttoolpolicy.HeldByPersonalExemption)
	assert.Contains(t, clean.HeldBy, agenttoolpolicy.HeldByToolTier)

	unattended, _ := assessBoth(t, agenttoolpolicy.AssessInput{
		Policy:     policy,
		Definition: scheduledAgent(agent.TierAutoExecute, nil),
	})
	assert.Equal(t, agenttooltrust.AutonomyProposeOnly, unattended.Answer)
	assert.NotContains(t, unattended.HeldBy, agenttoolpolicy.HeldByPersonalExemption)
}

func TestAssess_AConditionMakesAnAutomaticToolConditional(t *testing.T) {
	t.Parallel()

	policy := actionPolicy("inbox_mark_inbound_message", agent.EgressInternal)
	policy.Condition = &agenttoolports.TierCondition{
		Description: "Runs only as far as its mailbox allows.",
		Limit: func(context.Context, agenttoolports.ToolExecuteParams) agent.AutonomyTier {
			panic("a representative call never reads a record")
		},
	}

	clean, tainted := assessBoth(t, agenttoolpolicy.AssessInput{
		Policy:     policy,
		Definition: chatAgent(agent.TierAutoExecute, nil),
	})

	for _, autonomy := range []agenttoolports.ToolAutonomy{clean, tainted} {
		assert.Equal(t, agenttooltrust.AutonomyConditional, autonomy.Answer)
		assert.Equal(t, agent.TierAutoExecute, autonomy.Tier)
		assert.Contains(t, autonomy.HeldBy, agenttoolpolicy.HeldByCondition)
	}
}

func TestAssess_SimulationAndShadowPreviewEveryWrite(t *testing.T) {
	t.Parallel()

	policy := actionPolicy("dispatch_assign_move", agent.EgressInternal)

	simulated := chatAgent(agent.TierAutoExecute, nil)
	simulated.SimulationMode = true
	clean, _ := assessBoth(t, agenttoolpolicy.AssessInput{
		Policy:     policy,
		Definition: simulated,
		Control:    &agentcontrol.AgentControl{EarnedAutonomy: true, PromotionThreshold: 5},
	})
	assert.Equal(t, agenttooltrust.AutonomySimulated, clean.Answer)
	assert.Equal(t, agent.TierAutoExecute, clean.Tier)
	assert.Contains(t, clean.HeldBy, agenttoolpolicy.HeldBySimulationMode)

	paused, _ := assessBoth(t, agenttoolpolicy.AssessInput{
		Policy:     policy,
		Definition: chatAgent(agent.TierActWithApproval, nil),
		Control: &agentcontrol.AgentControl{
			ShadowMode:         true,
			EarnedAutonomy:     true,
			PromotionThreshold: 5,
		},
	})
	assert.Equal(t, agenttooltrust.AutonomySimulated, paused.Answer)
	assert.Contains(t, paused.HeldBy, agenttoolpolicy.HeldByShadowMode)
	assert.Nil(t, paused.ApprovalsToNext, "a run in shadow proposes nothing a person decides")
}

func TestAssess_AQueryIsNeverSimulated(t *testing.T) {
	t.Parallel()

	policy := agenttoolports.ToolPolicy{
		Name:          "shipment_get",
		Kind:          agenttooltrust.ToolKindQuery,
		DefaultTier:   agent.TierAutoExecute,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressNone},
		ReadsExternal: agenttooltrust.ExternalReadNever,
	}
	definition := chatAgent(agent.TierAutoExecute, nil)
	definition.SimulationMode = true

	clean, tainted := assessBoth(t, agenttoolpolicy.AssessInput{
		Policy:     policy,
		Definition: definition,
		Control:    &agentcontrol.AgentControl{ShadowMode: true},
	})

	assert.Equal(t, agenttooltrust.AutonomyRunsOnItsOwn, clean.Answer)
	assert.Equal(t, agenttooltrust.AutonomyRunsOnItsOwn, tainted.Answer)
	assert.Nil(t, clean.ApprovalsToNext)
}

func TestAssess_EarnedTiersAndTheStreakToTheNext(t *testing.T) {
	t.Parallel()

	name := "dispatch_assign_move"
	definition := chatAgent(agent.TierAutoExecute, map[string]agent.AutonomyTier{
		name: agent.TierActWithApproval,
	})
	trust := &agenttooltrust.ToolTrust{
		ToolName:   name,
		Streak:     3,
		EarnedTier: agent.TierActWithApproval,
	}
	control := &agentcontrol.AgentControl{EarnedAutonomy: true, PromotionThreshold: 10}

	clean, tainted := assessBoth(t, agenttoolpolicy.AssessInput{
		Policy:     actionPolicy(name, agent.EgressInternal),
		Definition: definition,
		Trust:      trust,
		Control:    control,
	})

	for _, autonomy := range []agenttoolports.ToolAutonomy{clean, tainted} {
		assert.Equal(t, agenttooltrust.AutonomyNeedsApproval, autonomy.Answer)
		assert.True(t, autonomy.Earned)
		require.NotNil(t, autonomy.ApprovalsToNext)
		assert.Equal(t, 7, *autonomy.ApprovalsToNext)
	}

	trust.Streak = 12
	past, _ := assessBoth(t, agenttoolpolicy.AssessInput{
		Policy:     actionPolicy(name, agent.EgressInternal),
		Definition: definition,
		Trust:      trust,
		Control:    control,
	})
	require.NotNil(t, past.ApprovalsToNext)
	assert.Equal(t, 1, *past.ApprovalsToNext)
}

func TestAssess_NoStreakCanMoveWhatALimitHolds(t *testing.T) {
	t.Parallel()

	control := &agentcontrol.AgentControl{EarnedAutonomy: true, PromotionThreshold: 10}
	name := "customer_send_email"

	leaves, _ := assessBoth(t, agenttoolpolicy.AssessInput{
		Policy: actionPolicy(name, agent.EgressExternalRecipient),
		Definition: chatAgent(agent.TierAutoExecute, map[string]agent.AutonomyTier{
			name: agent.TierActWithApproval,
		}),
		Control: control,
	})
	assert.Nil(t, leaves.ApprovalsToNext, "approval is the most a message out may earn")

	capped, _ := assessBoth(t, agenttoolpolicy.AssessInput{
		Policy:     actionPolicy("dispatch_assign_move", agent.EgressInternal),
		Definition: chatAgent(agent.TierPropose, nil),
		Control:    control,
	})
	assert.Nil(t, capped.ApprovalsToNext, "the agent's ceiling stops a promotion")

	off, _ := assessBoth(t, agenttoolpolicy.AssessInput{
		Policy: actionPolicy("dispatch_assign_move", agent.EgressInternal),
		Definition: chatAgent(agent.TierAutoExecute, map[string]agent.AutonomyTier{
			"dispatch_assign_move": agent.TierPropose,
		}),
		Control: &agentcontrol.AgentControl{PromotionThreshold: 10},
	})
	assert.Nil(t, off.ApprovalsToNext, "nothing is earned while earned autonomy is off")
	assert.False(t, off.Earned)
}

func TestAssess_ARegisteredToolAnswersLikeItsPolicy(t *testing.T) {
	t.Parallel()

	policy := registeredPolicy(t, "customer_send_email")
	clean, tainted := assessBoth(t, agenttoolpolicy.AssessInput{
		Policy: policy,
		Definition: chatAgent(agent.TierAutoExecute, map[string]agent.AutonomyTier{
			policy.Name: agent.TierAutoExecute,
		}),
	})

	assert.False(t, clean.Answer.WithoutAPerson())
	assert.False(t, tainted.Answer.WithoutAPerson())
	assert.False(t, clean.Tier.Above(agenttoolpolicy.Promotable(&policy)))
}
