package receivabletools_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/receivabletools"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func escalationPolicies() map[string]agentports.ToolPolicy {
	return map[string]agentports.ToolPolicy{
		"raise_exception":        (&receivabletools.RaiseExceptionTool{}).Policy(),
		"flag_for_manual_review": (&receivabletools.FlagManualReviewTool{}).Policy(),
	}
}

func unattendedCall() agentports.ToolExecuteParams {
	return agentports.ToolExecuteParams{
		Actor: &permissionports.RequestActor{
			PrincipalType: permissionports.PrincipalTypeAgent,
			PrincipalID:   pulid.MustNew("agdef_"),
		},
		Params: map[string]any{},
	}
}

func TestEscalation_RunsWhateverTheAgentsCeiling(t *testing.T) {
	t.Parallel()

	tainted := &agent.RunTaint{}
	tainted.Add(agent.TaintMark{
		Source:   agent.TaintSourceInboundMessage,
		ToolName: "get_inbound_message",
		CallID:   "call_1",
		Ref:      &agent.RecordRef{EntityType: "inbound_message", ID: "imsg_1"},
	})

	for name, policy := range escalationPolicies() {
		for _, ceiling := range []agent.AutonomyTier{
			agent.TierPropose,
			agent.TierActWithApproval,
			agent.TierAutoExecute,
		} {
			definition := &agentdefinition.Definition{AutonomyCeiling: ceiling}

			decision := agenttoolpolicy.Decide(t.Context(), agenttoolpolicy.DecideInput{
				Policy:     policy,
				Params:     unattendedCall(),
				Definition: definition,
				Unattended: true,
				Taint:      tainted,
			})

			assert.Equal(t, agent.TierAutoExecute, decision.Tier, "%s under %s", name, ceiling)
			assert.Empty(t, decision.HeldBy, "%s under %s", name, ceiling)
			assert.Equal(t, agent.TierAutoExecute,
				agenttoolpolicy.StaticTier(definition, policy), "%s under %s", name, ceiling)
			assert.False(t, agenttoolpolicy.AlwaysProposes(definition, policy),
				"%s under %s is not listed as a change that waits for a person", name, ceiling)
		}
	}
}

func TestEscalation_KeepsATierAPersonSetOnTheAgent(t *testing.T) {
	t.Parallel()

	for name, policy := range escalationPolicies() {
		definition := &agentdefinition.Definition{
			AutonomyCeiling: agent.TierAutoExecute,
			ToolTiers:       map[string]agent.AutonomyTier{name: agent.TierPropose},
		}

		decision := agenttoolpolicy.Decide(t.Context(), agenttoolpolicy.DecideInput{
			Policy:          policy,
			Params:          unattendedCall(),
			Definition:      definition,
			Unattended:      true,
			TierSetByPerson: true,
		})

		assert.Equal(t, agent.TierPropose, decision.Tier, name)
		assert.Equal(t, agent.TierSourcePersonSetting, decision.Source, name)
		assert.Equal(t, agent.TierPropose, agenttoolpolicy.StaticTier(definition, policy), name)
	}
}
