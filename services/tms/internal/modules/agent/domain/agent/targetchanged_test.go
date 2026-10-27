package agent_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/stretchr/testify/assert"
)

func TestPlanTargetChanged(t *testing.T) {
	t.Parallel()

	refused := agent.ErrPlanTargetChanged.Error() +
		"; step 2: the record changed since this was proposed: the shipment S-1 was at version 3"

	tests := []struct {
		name string
		plan *agent.AgentPlan
		want bool
	}{
		{name: "no plan", plan: nil, want: false},
		{
			name: "a plan refused because a record it changes moved on",
			plan: &agent.AgentPlan{Status: agent.PlanStatusFailed, FailureError: refused},
			want: true,
		},
		{
			name: "a plan whose step failed when it ran",
			plan: &agent.AgentPlan{
				Status:       agent.PlanStatusFailed,
				FailureError: "1 of 2 steps did not run; step 2: the customer is on credit hold",
			},
			want: false,
		},
		{
			name: "a plan waiting out its undo window",
			plan: &agent.AgentPlan{Status: agent.PlanStatusApproving},
			want: false,
		},
		{
			name: "only a failed plan is refused",
			plan: &agent.AgentPlan{Status: agent.PlanStatusApproved, FailureError: refused},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, agent.PlanTargetChanged(tt.plan))
		})
	}
}
