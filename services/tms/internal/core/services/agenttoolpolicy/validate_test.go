package agenttoolpolicy

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validAction() Spec {
	return Spec{
		ToolName: "assign_move",
		Policy: serviceports.ToolPolicy{
			Name:          "assign_move",
			Kind:          agent.ToolKindAction,
			Resource:      permission.ResourceShipmentMove,
			Operation:     permission.OpUpdate,
			Scope:         agent.ToolScopeTenant,
			DefaultTier:   agent.TierActWithApproval,
			MaxTier:       agent.TierAutoExecute,
			Egress:        []agent.EgressClass{agent.EgressInternal},
			Effect:        agent.ToolEffectChange,
			ReadsExternal: agent.ExternalReadNever,
			Rationale:     "Assigns a driver and tractor inside Trenova.",
		},
	}
}

func validQuery() Spec {
	return Spec{
		ToolName: "get_shipment",
		Policy: serviceports.ToolPolicy{
			Name:          "get_shipment",
			Kind:          agent.ToolKindQuery,
			Resource:      permission.ResourceShipment,
			Operation:     permission.OpRead,
			Scope:         agent.ToolScopeTenant,
			DefaultTier:   agent.TierAutoExecute,
			MaxTier:       agent.TierAutoExecute,
			Egress:        []agent.EgressClass{agent.EgressNone},
			Effect:        agent.ToolEffectLookup,
			ReadsExternal: agent.ExternalReadNever,
			Rationale:     "Reads a shipment.",
		},
	}
}

func holdToPropose(context.Context, serviceports.ToolExecuteParams) agent.AutonomyTier {
	return agent.TierPropose
}

func classifyAsInternal(serviceports.ToolExecuteParams) serviceports.CallPolicy {
	return serviceports.CallPolicy{Egress: agent.EgressInternal}
}

func TestValidate_AcceptsWellFormedSpecs(t *testing.T) {
	t.Parallel()

	require.NoError(t, Validate([]Spec{validAction(), validQuery()}))
}

/*
Each rule has one fixture that breaks it and nothing else, so a rule that stops
being enforced fails here by name rather than passing silently at boot.
*/
func TestValidate_RefusesEachBrokenRule(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		specs  func() []Spec
		reason string
	}{
		{
			name:   "names are unique",
			specs:  func() []Spec { return []Spec{validAction(), validAction()} },
			reason: "declared twice",
		},
		{
			name: "the policy is named for its tool",
			specs: func() []Spec {
				spec := validAction()
				spec.ToolName = "reassign_move"
				return []Spec{spec}
			},
			reason: "but the tool is",
		},
		{
			name: "every egress class is valid",
			specs: func() []Spec {
				spec := validAction()
				spec.Policy.Egress = []agent.EgressClass{"broadcast"}
				return []Spec{spec}
			},
			reason: "egress class \"broadcast\" is not valid",
		},
		{
			name: "a policy declares an egress class",
			specs: func() []Spec {
				spec := validAction()
				spec.Policy.Egress = nil
				return []Spec{spec}
			},
			reason: "declares no egress class",
		},
		{
			name: "every tier is valid",
			specs: func() []Spec {
				spec := validAction()
				spec.Policy.MaxTier = "Sometimes"
				return []Spec{spec}
			},
			reason: "max tier \"Sometimes\" is not valid",
		},
		{
			name: "the default tier is valid",
			specs: func() []Spec {
				spec := validAction()
				spec.Policy.DefaultTier = ""
				return []Spec{spec}
			},
			reason: "default tier \"\" is not valid",
		},
		{
			name: "the effect is valid",
			specs: func() []Spec {
				spec := validAction()
				spec.Policy.Effect = "explode"
				return []Spec{spec}
			},
			reason: "effect \"explode\" is not valid",
		},
		{
			name: "the kind is valid",
			specs: func() []Spec {
				spec := validAction()
				spec.Policy.Kind = "helper"
				return []Spec{spec}
			},
			reason: "kind \"helper\" is not valid",
		},
		{
			name: "the scope is valid",
			specs: func() []Spec {
				spec := validAction()
				spec.Policy.Scope = "planet"
				return []Spec{spec}
			},
			reason: "scope \"planet\" is not valid",
		},
		{
			name: "the resource is registered",
			specs: func() []Spec {
				spec := validAction()
				spec.Policy.Resource = "flux_capacitor"
				return []Spec{spec}
			},
			reason: "not a registered permission resource",
		},
		{
			name: "the operation is valid",
			specs: func() []Spec {
				spec := validAction()
				spec.Policy.Operation = "teleport"
				return []Spec{spec}
			},
			reason: "operation \"teleport\" is not valid",
		},
		{
			name: "a query's only class is none",
			specs: func() []Spec {
				spec := validQuery()
				spec.Policy.Egress = []agent.EgressClass{agent.EgressInternal}
				return []Spec{spec}
			},
			reason: "a query tool's only egress class is none",
		},
		{
			name: "a query reads",
			specs: func() []Spec {
				spec := validQuery()
				spec.Policy.Operation = permission.OpUpdate
				return []Spec{spec}
			},
			reason: "used under the read operation",
		},
		{
			name: "an action is not none",
			specs: func() []Spec {
				spec := validAction()
				spec.Policy.Egress = []agent.EgressClass{agent.EgressNone}
				return []Spec{spec}
			},
			reason: "none is not one of its classes",
		},
		{
			name: "the default tier is at most the max tier",
			specs: func() []Spec {
				spec := validAction()
				spec.Policy.DefaultTier = agent.TierAutoExecute
				spec.Policy.MaxTier = agent.TierActWithApproval
				return []Spec{spec}
			},
			reason: "is above its max tier",
		},
		{
			name: "the max tier is at most what the classes allow",
			specs: func() []Spec {
				spec := validAction()
				spec.Policy.Egress = []agent.EgressClass{agent.EgressExternalRecipient}
				return []Spec{spec}
			},
			reason: "the most its egress classes allow",
		},
		{
			name: "several classes need a classifier",
			specs: func() []Spec {
				spec := validAction()
				spec.Policy.Egress = []agent.EgressClass{
					agent.EgressInternal,
					agent.EgressMoney,
				}
				return []Spec{spec}
			},
			reason: "so it must classify each call",
		},
		{
			name: "a class is listed once",
			specs: func() []Spec {
				spec := validAction()
				spec.Policy.Egress = []agent.EgressClass{
					agent.EgressInternal,
					agent.EgressInternal,
				}
				spec.Policy.Classify = classifyAsInternal
				return []Spec{spec}
			},
			reason: "is listed twice",
		},
		{
			name: "a self-scoped write is personal",
			specs: func() []Spec {
				spec := validAction()
				spec.Policy.Scope = agent.ToolScopeSelf
				return []Spec{spec}
			},
			reason: "acts only on the caller's own records",
		},
		{
			name: "running unasked needs a personal class",
			specs: func() []Spec {
				spec := validAction()
				spec.Policy.PersonalRunsUnasked = true
				return []Spec{spec}
			},
			reason: "runs a personal call unasked",
		},
		{
			name: "reading outside text names its source",
			specs: func() []Spec {
				spec := validQuery()
				spec.Policy.ReadsExternal = agent.ExternalReadAlways
				return []Spec{spec}
			},
			reason: "names the taint source",
		},
		{
			name: "the external read is valid",
			specs: func() []Spec {
				spec := validQuery()
				spec.Policy.ReadsExternal = ""
				return []Spec{spec}
			},
			reason: "external read \"\" is not valid",
		},
		{
			name: "only a write carries taint",
			specs: func() []Spec {
				spec := validQuery()
				spec.Policy.CarriesTaint = true
				return []Spec{spec}
			},
			reason: "only a write can carry",
		},
		{
			name: "further sources belong to a tool that reads outside text",
			specs: func() []Spec {
				spec := validQuery()
				spec.Policy.Sources = []agent.TaintSource{agent.TaintSourceWeather}
				return []Spec{spec}
			},
			reason: "names further taint sources but reads no outside text",
		},
		{
			name: "a further source is a real one",
			specs: func() []Spec {
				spec := validQuery()
				spec.Policy.ReadsExternal = agent.ExternalReadMarked
				spec.Policy.Source = agent.TaintSourceInboundMessage
				spec.Policy.Sources = []agent.TaintSource{"gossip"}
				return []Spec{spec}
			},
			reason: "taint source \"gossip\" is not valid",
		},
		{
			name: "a further source is not named twice",
			specs: func() []Spec {
				spec := validQuery()
				spec.Policy.ReadsExternal = agent.ExternalReadMarked
				spec.Policy.Source = agent.TaintSourceInboundMessage
				spec.Policy.Sources = []agent.TaintSource{
					agent.TaintSourceWeather,
					agent.TaintSourceInboundMessage,
				}
				return []Spec{spec}
			},
			reason: "taint source \"inbound_message\" is named twice",
		},
		{
			name: "a reporter's artifact is a record-link key",
			specs: func() []Spec {
				spec := validAction()
				spec.Reports = true
				spec.Policy.Artifact = "not_a_record"
				return []Spec{spec}
			},
			reason: "must be a record-link key",
		},
		{
			name: "a reporter names its artifact",
			specs: func() []Spec {
				spec := validAction()
				spec.Reports = true
				return []Spec{spec}
			},
			reason: "must be a record-link key",
		},
		{
			name: "any artifact is a record-link key",
			specs: func() []Spec {
				spec := validAction()
				spec.Policy.Artifact = "not_a_record"
				return []Spec{spec}
			},
			reason: "is not a record-link key",
		},
		{
			name: "a condition says what it does and how",
			specs: func() []Spec {
				spec := validAction()
				spec.Policy.Condition = &serviceports.TierCondition{Limit: holdToPropose}
				return []Spec{spec}
			},
			reason: "condition needs both a description and a limit",
		},
		{
			name: "a runtime tool never leaves",
			specs: func() []Spec {
				spec := validAction()
				spec.Policy.Kind = agent.ToolKindRuntime
				spec.Policy.Egress = []agent.EgressClass{agent.EgressMoney}
				return []Spec{spec}
			},
			reason: "a runtime tool never leaves the organization",
		},
		{
			name: "every policy explains itself",
			specs: func() []Spec {
				spec := validAction()
				spec.Policy.Rationale = " "
				return []Spec{spec}
			},
			reason: "has no rationale",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := Validate(tc.specs())
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.reason)
		})
	}
}

func TestValidate_ReportsEveryProblemAtOnce(t *testing.T) {
	t.Parallel()

	broken := validAction()
	broken.Policy.Effect = ""
	broken.Policy.Rationale = ""

	err := Validate([]Spec{broken})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "effect")
	assert.Contains(t, err.Error(), "rationale")
}
