package agenttoolpolicy_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolpolicy"
	"github.com/stretchr/testify/assert"
)

func writePolicy(egress agent.EgressClass) serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          "assign_move",
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceShipmentMove,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{egress},
		Effect:        agent.ToolEffectChange,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     "A move assignment.",
	}
}

func TestDecide_SaysWhatSetTheTier(t *testing.T) {
	t.Parallel()

	internal := writePolicy(agent.EgressInternal)
	money := writePolicy(agent.EgressMoney)
	createReport := registeredPolicy(t, "create_report")
	private := map[string]any{"name": "Late loads", "visibility": string(report.VisibilityPrivate)}

	cases := []struct {
		name   string
		in     agenttoolpolicy.DecideInput
		tier   agent.AutonomyTier
		source agent.TierSource
	}{
		{
			name: "the tool's own default",
			in: agenttoolpolicy.DecideInput{
				Policy:     internal,
				Params:     call(person(), nil),
				Definition: definition(agent.TierAutoExecute, nil),
				Taint:      clean(),
			},
			tier:   agent.TierActWithApproval,
			source: agent.TierSourcePolicyDefault,
		},
		{
			name: "a tier a person chose on the agent",
			in: agenttoolpolicy.DecideInput{
				Policy: internal,
				Params: call(person(), nil),
				Definition: definition(agent.TierAutoExecute, map[string]agent.AutonomyTier{
					"assign_move": agent.TierAutoExecute,
				}),
				TierSetByPerson: true,
				Taint:           clean(),
			},
			tier:   agent.TierAutoExecute,
			source: agent.TierSourcePersonSetting,
		},
		{
			name: "a tier the tool earned through clean approvals",
			in: agenttoolpolicy.DecideInput{
				Policy: internal,
				Params: call(person(), nil),
				Definition: definition(agent.TierAutoExecute, map[string]agent.AutonomyTier{
					"assign_move": agent.TierAutoExecute,
				}),
				TierSetByPerson: false,
				Taint:           clean(),
			},
			tier:   agent.TierAutoExecute,
			source: agent.TierSourceTrustEarned,
		},
		{
			name: "a person's tier that the agent's ceiling lowered is the ceiling's",
			in: agenttoolpolicy.DecideInput{
				Policy: internal,
				Params: call(person(), nil),
				Definition: definition(agent.TierPropose, map[string]agent.AutonomyTier{
					"assign_move": agent.TierAutoExecute,
				}),
				TierSetByPerson: true,
				Taint:           clean(),
			},
			tier:   agent.TierPropose,
			source: agent.TierSourcePolicyDefault,
		},
		{
			name: "a private write for the person in the conversation",
			in: agenttoolpolicy.DecideInput{
				Policy:     createReport,
				Params:     call(person(), private),
				Definition: definition(agent.TierPropose, nil),
				Taint:      clean(),
			},
			tier:   agent.TierAutoExecute,
			source: agent.TierSourcePersonalExemption,
		},
		{
			name: "an earned tier that taint held back is the policy's",
			in: agenttoolpolicy.DecideInput{
				Policy: money,
				Params: call(person(), nil),
				Definition: definition(agent.TierAutoExecute, map[string]agent.AutonomyTier{
					"assign_move": agent.TierAutoExecute,
				}),
				Taint: tainted(),
			},
			tier:   agent.TierActWithApproval,
			source: agent.TierSourcePolicyDefault,
		},
		{
			name: "no agent at all",
			in: agenttoolpolicy.DecideInput{
				Policy: internal,
				Params: call(person(), nil),
				Taint:  clean(),
			},
			tier:   agent.TierActWithApproval,
			source: agent.TierSourcePolicyDefault,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			decision := agenttoolpolicy.Decide(t.Context(), tc.in)
			assert.Equal(t, tc.tier, decision.Tier)
			assert.Equal(t, tc.source, decision.Source)
			assert.True(t, decision.Source.IsValid())
		})
	}
}
