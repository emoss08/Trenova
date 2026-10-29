package assistantproposals

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListThreadProposals_SaysWhoDecidedEachAndWhatTheyWrote(t *testing.T) {
	t.Parallel()

	threadID := pulid.MustNew("thr_")
	decider := pulid.MustNew("usr_")
	rejected := &agent.AgentProposal{
		ID:        pulid.MustNew("ap_"),
		ToolName:  "invoice_post",
		Status:    agent.ProposalStatusRejected,
		CreatedAt: 1_700_000_000,
	}
	waiting := &agent.AgentProposal{
		ID:        pulid.MustNew("ap_"),
		ToolName:  "invoice_post",
		Status:    agent.ProposalStatusExpired,
		CreatedAt: 1_700_000_100,
	}
	older := assistanttest.DecisionWithNote(
		rejected.ID,
		agentdecision.DecisionRejected,
		"first word",
	)
	older.CreatedAt = 1_700_000_050
	latest := assistanttest.DecisionWithNote(
		rejected.ID,
		agentdecision.DecisionRejected,
		"Wrong customer.",
	)
	latest.CreatedAt = 1_700_000_060
	latest.DecidedByUserID = decider
	svc := newProposalReads(
		&assistanttest.StubRunRepo{},
		&assistanttest.StubProposalRepo{ByThread: []*agent.AgentProposal{rejected, waiting}},
		&assistanttest.StubConversationRepo{Thread: &conversation.Thread{ID: threadID}},
	)
	svc.decisions = &assistanttest.StubDecisionRepo{
		Decisions: []*agentdecision.AgentDecision{latest, older},
	}

	result, err := svc.ListThreadProposals(
		t.Context(),
		&conversationports.GetThreadRequest{ID: threadID},
	)
	require.NoError(t, err)
	require.Len(t, result, 2)

	assert.Equal(t, int64(1_700_000_000), result[0].CreatedAt)
	require.NotNil(t, result[0].DecidedAt)
	assert.Equal(t, int64(1_700_000_060), *result[0].DecidedAt, "the latest decision is the one")
	assert.Equal(t, decider, result[0].DecidedByUserID)
	assert.Equal(t, "Wrong customer.", result[0].DecisionNote)

	assert.Equal(t, int64(1_700_000_100), result[1].CreatedAt, "the approval box orders by it")
	assert.Nil(t, result[1].DecidedAt)
	assert.Empty(t, result[1].DecisionNote)
}
