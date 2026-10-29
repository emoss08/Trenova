package agentplan_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/stretchr/testify/assert"
)

func TestPlanTargetChanged(t *testing.T) {
	t.Parallel()

	refused := agentplan.ErrPlanTargetChanged.Error() +
		"; step 2: the record changed since this was proposed: the shipment S-1 was at version 3"

	tests := []struct {
		name string
		plan *agentplan.AgentPlan
		want bool
	}{
		{name: "no plan", plan: nil, want: false},
		{
			name: "a plan refused because a record it changes moved on",
			plan: &agentplan.AgentPlan{Status: agentplan.PlanStatusFailed, FailureError: refused},
			want: true,
		},
		{
			name: "a plan whose step failed when it ran",
			plan: &agentplan.AgentPlan{
				Status:       agentplan.PlanStatusFailed,
				FailureError: "1 of 2 steps did not run; step 2: the customer is on credit hold",
			},
			want: false,
		},
		{
			name: "a plan waiting out its undo window",
			plan: &agentplan.AgentPlan{Status: agentplan.PlanStatusApproving},
			want: false,
		},
		{
			name: "only a failed plan is refused",
			plan: &agentplan.AgentPlan{Status: agentplan.PlanStatusApproved, FailureError: refused},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, agentplan.PlanTargetChanged(tt.plan))
		})
	}
}
