package decisionnotes

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
An agent_create_memory held for review was approved and ran. The follow-up told
the person "it still needs an administrator's approval to take effect": the
note said only that it ran, and the call that filed it had said it was awaiting
review. The note now names what the memory came to, and retires the stale
"awaiting review".
*/
func TestDecisionNote_SaysAnApprovedProposalRanAndIsNoLongerAwaitingReview(t *testing.T) {
	t.Parallel()

	proposalID := pulid.MustNew("ap_")
	notes := &Notes{
		Proposals: &assistanttest.StubProposalRepo{ByThread: []*agent.AgentProposal{{
			ID:       proposalID,
			ToolName: "agent_create_memory",
			Status:   agent.ProposalStatusExecuted,
			ExecutionResult: &agentrecord.ToolExecutionResult{
				Action: "saved",
				Kind:   "memory",
				Name:   "Plan Delayed shipments in batches of 50.",
				IDs:    map[string]string{"memoryId": "amem_1"},
				State: "kept and in effect, read in the later conversations of everyone in " +
					"this person's role",
			},
		}}},
	}
	actor := assistanttest.Actor()

	followed, err := notes.DecisionNote(t.Context(), &DecisionNoteParams{
		Thread:     &conversation.Thread{ID: pulid.MustNew("ath_")},
		ProposalID: proposalID,
		Tenant:     actor.TenantInfo(),
	})
	require.NoError(t, err)

	assert.Equal(t,
		`Approved agent_create_memory, and it ran. It saved the memory "Plan Delayed `+
			`shipments in batches of 50.". Now: kept and in effect, read in the later `+
			`conversations of everyone in this person's role.`,
		DecisionHeadline(followed.Note))
	assert.Contains(t, followed.Note, "that it was awaiting a person's review, is out of "+
		"date: the person approved it and it ran. Do not say it still waits for approval")
}

func TestSupersededReview_FollowsHowTheDecisionEnded(t *testing.T) {
	t.Parallel()

	proposal := func(status agent.ProposalStatus) *agent.AgentProposal {
		return &agent.AgentProposal{ToolName: "agent_create_memory", Status: status}
	}

	assert.Contains(t, supersededReview(proposal(agent.ProposalStatusRejected)),
		"the person rejected it, and it will not run")
	assert.Contains(t, supersededReview(proposal(agent.ProposalStatusExecutionFailed)),
		"the person approved it")
	assert.Empty(t, supersededReview(proposal(agent.ProposalStatusAccepted)),
		"one still being carried out has not ended")
}
