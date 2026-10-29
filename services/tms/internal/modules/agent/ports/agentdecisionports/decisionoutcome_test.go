package agentdecisionports

import (
	"errors"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestDecisionOutcomeResult(t *testing.T) {
	t.Parallel()

	proposalID := pulid.MustNew("ap_")
	commitsAt := int64(1_790_000_030)
	refusal := fmt.Errorf(
		"%w: the shipment S-1 was at version 3 when this was proposed and is at version 5 now, "+
			"so nothing was changed",
		agentplan.ErrTargetChanged,
	)

	tests := []struct {
		name    string
		outcome DecisionOutcome
		want    AgentProposalDecisionResult
	}{
		{
			name: "an approval that ran",
			outcome: DecisionOutcome{
				Decision: &agentdecision.AgentDecision{Decision: agentdecision.DecisionAccepted},
			},
			want: AgentProposalDecisionResult{Executed: true},
		},
		{
			name: "an approval with changes that ran",
			outcome: DecisionOutcome{
				Decision: &agentdecision.AgentDecision{Decision: agentdecision.DecisionModified},
			},
			want: AgentProposalDecisionResult{Executed: true},
		},
		{
			name: "an approval waiting out its undo window has not run",
			outcome: DecisionOutcome{
				Decision: &agentdecision.AgentDecision{
					Decision:  agentdecision.DecisionAccepted,
					CommitsAt: &commitsAt,
				},
			},
			want: AgentProposalDecisionResult{},
		},
		{
			name: "a rejection runs nothing",
			outcome: DecisionOutcome{
				Decision: &agentdecision.AgentDecision{Decision: agentdecision.DecisionRejected},
			},
			want: AgentProposalDecisionResult{},
		},
		{
			name: "an approval refused because its record moved on",
			outcome: DecisionOutcome{
				Decision: &agentdecision.AgentDecision{
					Decision: agentdecision.DecisionAccepted,
				},
				ExecutionError: refusal,
			},
			want: AgentProposalDecisionResult{Error: refusal.Error(), Refused: true},
		},
		{
			name: "an approval whose write failed is not a refusal",
			outcome: DecisionOutcome{
				Decision: &agentdecision.AgentDecision{
					Decision: agentdecision.DecisionAccepted,
				},
				ExecutionError: errors.New("the customer is on credit hold"),
			},
			want: AgentProposalDecisionResult{Error: "the customer is on credit hold"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.outcome.Result(proposalID)

			tt.want.ProposalID = proposalID
			tt.want.Decision = tt.outcome.Decision
			assert.Equal(t, tt.want, got)
		})
	}
}
