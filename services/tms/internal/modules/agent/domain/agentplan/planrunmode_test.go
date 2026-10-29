package agentplan_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestPlanRunModeOf(t *testing.T) {
	t.Parallel()

	shared := pulid.MustNew("inv_")
	tests := []struct {
		name  string
		steps []*agent.AgentProposal
		want  agentplan.PlanRunMode
	}{
		{name: "no steps", steps: nil, want: agentplan.PlanRunModeInOrder},
		{
			name:  "one step has nothing to be independent of",
			steps: []*agent.AgentProposal{{TargetResource: "invoice", TargetID: shared}},
			want:  agentplan.PlanRunModeInOrder,
		},
		{
			name: "each step changes a different record of one kind",
			steps: []*agent.AgentProposal{
				{TargetResource: "invoice", TargetID: pulid.MustNew("inv_")},
				{TargetResource: "invoice", TargetID: pulid.MustNew("inv_")},
				{TargetResource: "invoice", TargetID: pulid.MustNew("inv_")},
			},
			want: agentplan.PlanRunModeEachStep,
		},
		{
			name: "two steps change the same record",
			steps: []*agent.AgentProposal{
				{TargetResource: "invoice", TargetID: shared},
				{TargetResource: "invoice", TargetID: pulid.MustNew("inv_")},
				{TargetResource: "invoice", TargetID: shared},
			},
			want: agentplan.PlanRunModeInOrder,
		},
		{
			name: "the steps change records of different kinds",
			steps: []*agent.AgentProposal{
				{TargetResource: "invoice", TargetID: pulid.MustNew("inv_")},
				{TargetResource: "shipment", TargetID: pulid.MustNew("shp_")},
			},
			want: agentplan.PlanRunModeInOrder,
		},
		{
			name: "a step without a target",
			steps: []*agent.AgentProposal{
				{TargetResource: "invoice", TargetID: pulid.MustNew("inv_")},
				{TargetResource: "invoice"},
			},
			want: agentplan.PlanRunModeInOrder,
		},
		{
			name: "a step without a resource",
			steps: []*agent.AgentProposal{
				{TargetID: pulid.MustNew("inv_")},
				{TargetID: pulid.MustNew("inv_")},
			},
			want: agentplan.PlanRunModeInOrder,
		},
		{
			name: "a missing first step",
			steps: []*agent.AgentProposal{
				nil,
				{TargetResource: "invoice", TargetID: pulid.MustNew("inv_")},
			},
			want: agentplan.PlanRunModeInOrder,
		},
		{
			name: "a missing step",
			steps: []*agent.AgentProposal{
				{TargetResource: "invoice", TargetID: pulid.MustNew("inv_")},
				nil,
			},
			want: agentplan.PlanRunModeInOrder,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, agentplan.PlanRunModeOf(tt.steps))
		})
	}
}

func TestPlanRunModeIsValid(t *testing.T) {
	t.Parallel()

	assert.True(t, agentplan.PlanRunModeInOrder.IsValid())
	assert.True(t, agentplan.PlanRunModeEachStep.IsValid())
	assert.False(t, agentplan.PlanRunMode("Parallel").IsValid())
	assert.False(t, agentplan.PlanRunMode("").IsValid())
}
