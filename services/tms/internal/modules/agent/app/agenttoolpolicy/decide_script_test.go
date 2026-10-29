package agenttoolpolicy_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/stretchr/testify/assert"
)

/*
A write a script prepared is an intent: it waits for a person to approve the
script's batch whatever would let the same call run on its own, and every
other limit still lowers it and is still named.
*/
func TestDecide_ScriptIntentsNeverRunOnTheirOwn(t *testing.T) {
	t.Parallel()

	money := registeredPolicy(t, "receivable_post_customer_payment")
	internal := registeredPolicy(t, "dispatch_assign_move")
	actsAlone := func(name string) *agentdefinition.Definition {
		return definition(agent.TierAutoExecute, map[string]agent.AutonomyTier{
			name: agent.TierAutoExecute,
		})
	}

	runDecideCases(t, []decideCase{
		{
			name: "an internal write the agent may run alone is held for the batch",
			in: agenttoolpolicy.DecideInput{
				Policy:     internal,
				Params:     call(agentPrincipal(), nil),
				Definition: actsAlone("dispatch_assign_move"),
				Taint:      clean(),
				FromScript: true,
			},
			tier:   agent.TierActWithApproval,
			heldBy: []string{agenttoolpolicy.HeldByScriptIntent},
		},
		{
			name: "a payment within its money bounds is still held for the batch",
			in: agenttoolpolicy.DecideInput{
				Policy:         provenWithin(money),
				Params:         call(agentPrincipal(), nil),
				Definition:     actsAlone("receivable_post_customer_payment"),
				Taint:          clean(),
				MoneyAllowance: allowance(agenttooltrust.MoneyUse{}, agenttooltrust.MoneyUse{}),
				FromScript:     true,
			},
			tier:   agent.TierActWithApproval,
			heldBy: []string{agenttoolpolicy.HeldByScriptIntent},
		},
		{
			name: "a payment past its money bounds names the bound and the script",
			in: agenttoolpolicy.DecideInput{
				Policy:         money,
				Params:         call(agentPrincipal(), nil),
				Definition:     actsAlone("receivable_post_customer_payment"),
				Taint:          clean(),
				MoneyAllowance: allowance(agenttooltrust.MoneyUse{}, agenttooltrust.MoneyUse{}),
				FromScript:     true,
			},
			tier: agent.TierActWithApproval,
			heldBy: []string{
				agenttooltrust.HeldByMoneyUnproven,
				agenttoolpolicy.HeldByScriptIntent,
			},
		},
		{
			name: "a tainted script's payment names the taint and the script",
			in: agenttoolpolicy.DecideInput{
				Policy:     money,
				Params:     call(agentPrincipal(), nil),
				Definition: actsAlone("receivable_post_customer_payment"),
				Taint:      tainted(),
				FromScript: true,
			},
			tier: agent.TierActWithApproval,
			heldBy: []string{
				agenttoolpolicy.HeldByTainted,
				agenttoolpolicy.HeldByScriptIntent,
			},
		},
		{
			name: "the agent's ceiling still lowers an intent to Propose",
			in: agenttoolpolicy.DecideInput{
				Policy: internal,
				Params: call(person(), nil),
				Definition: definition(agent.TierPropose, map[string]agent.AutonomyTier{
					"dispatch_assign_move": agent.TierAutoExecute,
				}),
				TierSetByPerson: true,
				Taint:           clean(),
				FromScript:      true,
			},
			tier: agent.TierPropose,
			heldBy: []string{
				agenttoolpolicy.HeldByAgentCeiling,
				agenttoolpolicy.HeldByScriptIntent,
			},
		},
		{
			name: "a tool that escalates on its own does not escalate from a script",
			in: agenttoolpolicy.DecideInput{
				Policy:     registeredPolicy(t, "agent_raise_exception"),
				Params:     call(agentPrincipal(), nil),
				Definition: definition(agent.TierAutoExecute, nil),
				Taint:      clean(),
				FromScript: true,
			},
			tier: agent.TierActWithApproval,
		},
	})
}

func TestDecide_ScriptIntentHoldsNoMoneyAllowance(t *testing.T) {
	t.Parallel()

	decision := agenttoolpolicy.Decide(t.Context(), agenttoolpolicy.DecideInput{
		Policy: provenWithin(registeredPolicy(t, "receivable_post_customer_payment")),
		Params: call(agentPrincipal(), nil),
		Definition: definition(agent.TierAutoExecute, map[string]agent.AutonomyTier{
			"receivable_post_customer_payment": agent.TierAutoExecute,
		}),
		Taint:          clean(),
		MoneyAllowance: allowance(agenttooltrust.MoneyUse{}, agenttooltrust.MoneyUse{}),
		FromScript:     true,
	})

	assert.Nil(
		t,
		decision.MoneyMinor,
		"a held write spends nothing of the run's or the day's allowance",
	)
}
