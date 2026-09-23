package assistantservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
A decision is not something the person said.

The note sent after a decision was exported as "## You" with the agent's
instructions under it: "Decision on proposal ap_… (create_report). Tell the
person in one or two sentences… Do not propose the same change again." The
person never typed it and the thread never shows those lines. The transcript
now heads it as a decision and shows only its first line, the one the thread
shows.
*/
func TestTranscript_RendersADecisionNoteAsADecision(t *testing.T) {
	t.Parallel()

	thread := transcriptThread()
	conversations := &stubConversations{thread: thread, messages: []conversation.Message{{
		Sequence: 1,
		Role:     conversation.RoleUser,
		Kind:     conversation.MessageKindDecisionNote,
		Content: "Approved create_report, and it ran. It created the report \"Lanes\".\n" +
			"Decision on proposal ap_01 (create_report). It produced definitionId rd_01; use " +
			"that id, not the proposal's, when you refer to what it made. " + followUpInstruction,
		CreatedAt: 1_790_000_000,
	}}}
	svc := newTranscriptService(conversations, &stubProposalRepo{})

	transcript, err := svc.Transcript(t.Context(), repositories.GetThreadRequest{ID: thread.ID})
	require.NoError(t, err)

	body := transcript.Body
	assert.Contains(t, body, "## Decision · Sep 21, 2026 2:13:20 PM UTC\n\n"+
		"Approved create_report, and it ran. It created the report \"Lanes\".\n")
	assert.NotContains(t, body, "## You")
	assert.NotContains(t, body, "Decision on proposal")
	assert.NotContains(t, body, "Tell the person")
	assert.NotContains(t, body, "Do not propose the same change again")
}

// A note with nothing readable in it still reads as a decision.
func TestTranscript_NamesAnEmptyDecisionNote(t *testing.T) {
	t.Parallel()

	thread := transcriptThread()
	conversations := &stubConversations{thread: thread, messages: []conversation.Message{{
		Sequence:  1,
		Role:      conversation.RoleUser,
		Kind:      conversation.MessageKindDecisionNote,
		CreatedAt: 1_790_000_000,
	}}}
	svc := newTranscriptService(conversations, &stubProposalRepo{})

	transcript, err := svc.Transcript(t.Context(), repositories.GetThreadRequest{ID: thread.ID})
	require.NoError(t, err)

	assert.Contains(t, transcript.Body, "## Decision · Sep 21, 2026 2:13:20 PM UTC\n\n"+
		"Following up on a decision.\n")
}

// An executed proposal lists what it made, with the ids in code so they can
// be copied.
func TestTranscript_ListsWhatAnExecutedProposalMade(t *testing.T) {
	t.Parallel()

	thread := transcriptThread()
	conversations := &stubConversations{thread: thread}
	proposals := &stubProposalRepo{byThread: []*agent.AgentProposal{{
		ToolName:     "create_report",
		ToolParams:   map[string]any{"name": "Lanes"},
		Rationale:    "Asked for a saved report.",
		AutonomyTier: agent.TierAutoExecute,
		Status:       agent.ProposalStatusExecuted,
		ExecutionResult: &agent.ToolExecutionResult{
			Action: "created",
			Kind:   "report",
			Name:   "Lanes",
			IDs:    map[string]string{"definitionId": "rd_01"},
		},
	}}}
	svc := newTranscriptService(conversations, proposals)

	transcript, err := svc.Transcript(t.Context(), repositories.GetThreadRequest{ID: thread.ID})
	require.NoError(t, err)

	assert.Contains(t, transcript.Body,
		"- **Result:** It created the report \"Lanes\" (definitionId `rd_01`).\n")
}
