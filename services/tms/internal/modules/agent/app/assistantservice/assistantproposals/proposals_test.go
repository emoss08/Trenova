package assistantproposals

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentshadow"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newProposalReads(
	runs *assistanttest.StubRunRepo,
	proposals *assistanttest.StubProposalRepo,
	conversations *assistanttest.StubConversationRepo,
) *Service {
	return newProposalReadsBehind(runs, proposals, conversations, assistanttest.ShadowSwitches{})
}

func newProposalReadsBehind(
	runs *assistanttest.StubRunRepo,
	proposals *assistanttest.StubProposalRepo,
	conversations *assistanttest.StubConversationRepo,
	switches assistanttest.ShadowSwitches,
) *Service {
	return &Service{
		logger:        zap.NewNop(),
		proposals:     proposals,
		conversations: conversations,
		shadow: agentshadow.New(agentshadow.Params{
			Control:     assistanttest.StubControlRepo{Paused: switches.OrganizationPaused},
			Runs:        runs,
			Definitions: assistanttest.StubDefinitionRepo{Definition: switches.Definition},
		}),
	}
}

func TestListThreadProposals_ScopesToTheThreadAndReportsExecution(t *testing.T) {
	t.Parallel()

	executedAt := int64(1700000000)
	threadID := pulid.MustNew("thr_")
	proposals := &assistanttest.StubProposalRepo{ByThread: []*agent.AgentProposal{
		{
			ID:             pulid.MustNew("ap_"),
			ToolName:       "reassign_move",
			ToolParams:     map[string]any{"moveId": "mv_1"},
			Status:         agent.ProposalStatusExecuted,
			ExecutedAt:     &executedAt,
			ExecutionError: "",
		},
		{
			ID:             pulid.MustNew("ap_"),
			ToolName:       "hold_shipment",
			Status:         agent.ProposalStatusExecutionFailed,
			ExecutionError: "shipment is already delivered",
		},
	}}
	conversations := &assistanttest.StubConversationRepo{Thread: &conversation.Thread{ID: threadID}}
	svc := newProposalReads(&assistanttest.StubRunRepo{}, proposals, conversations)

	result, err := svc.ListThreadProposals(
		t.Context(),
		&conversationports.GetThreadRequest{ID: threadID},
	)
	require.NoError(t, err)

	assert.Equal(t, threadID, proposals.LastList.ThreadID)
	require.Len(t, result, 2)
	assert.Equal(t, agent.ProposalStatusExecuted, result[0].Status)
	require.NotNil(t, result[0].ExecutedAt)
	assert.Equal(t, executedAt, *result[0].ExecutedAt)
	assert.Equal(t, "shipment is already delivered", result[1].ExecutionError)
}

// Reading the thread is the authorization check. Someone else's thread is not
// found, and its proposals must never be reached.
func TestListThreadProposals_RefusesAThreadTheReaderCannotSee(t *testing.T) {
	t.Parallel()

	proposals := &assistanttest.StubProposalRepo{}
	conversations := &assistanttest.StubConversationRepo{GetErr: errors.New("thread not found")}
	svc := newProposalReads(&assistanttest.StubRunRepo{}, proposals, conversations)

	_, err := svc.ListThreadProposals(
		t.Context(),
		&conversationports.GetThreadRequest{ID: pulid.MustNew("thr_")},
	)

	require.Error(t, err)
	assert.Empty(t, proposals.LastList.ThreadID, "the proposal query must not run at all")
}

// Reading the thread back decides the hold from the run's agent as it is now,
// not as it was when the proposal was made: turning the switch off is what
// releases the card. A proposal already decided is history and is never held,
// and its run is not even read.
func TestListThreadProposals_HoldsOnlyWhatIsStillPending(t *testing.T) {
	t.Parallel()

	def := &agentdefinition.Definition{
		ID:         pulid.MustNew("agd_"),
		Name:       "Dispatch desk",
		ShadowMode: true,
	}
	heldRun := &agent.AgentRun{ID: pulid.MustNew("ar_"), AgentDefinitionID: def.ID}
	decidedRun := &agent.AgentRun{ID: pulid.MustNew("ar_"), AgentDefinitionID: def.ID}
	runs := &assistanttest.StubRunRepo{
		ByID: map[pulid.ID]*agent.AgentRun{heldRun.ID: heldRun, decidedRun.ID: decidedRun},
	}
	threadID := pulid.MustNew("thr_")
	proposals := &assistanttest.StubProposalRepo{ByThread: []*agent.AgentProposal{
		{
			ID:       pulid.MustNew("ap_"),
			RunID:    heldRun.ID,
			ToolName: "reassign_move",
			Status:   agent.ProposalStatusPending,
		},
		{
			ID:       pulid.MustNew("ap_"),
			RunID:    decidedRun.ID,
			ToolName: "hold_shipment",
			Status:   agent.ProposalStatusRejected,
		},
	}}
	svc := newProposalReadsBehind(
		runs,
		proposals,
		&assistanttest.StubConversationRepo{Thread: &conversation.Thread{ID: threadID}},
		assistanttest.ShadowSwitches{Definition: def},
	)

	result, err := svc.ListThreadProposals(
		t.Context(),
		&conversationports.GetThreadRequest{ID: threadID},
	)
	require.NoError(t, err)

	require.Len(t, result, 2)
	require.NotNil(t, result[0].Hold)
	assert.Equal(t, string(agentshadow.CauseDefinition), result[0].Hold.Reason)
	assert.Equal(t, "Dispatch desk", result[0].Hold.AgentName)
	assert.Nil(t, result[1].Hold, "a decided proposal is never held")
	assert.Equal(t, 1, runs.Reads, "only the pending proposal's run is read")
}
