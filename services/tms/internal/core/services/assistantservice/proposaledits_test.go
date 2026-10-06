package assistantservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingEdits struct {
	saved []repositories.SetPendingModificationsRequest
}

func (r *recordingEdits) SetPendingModifications(
	_ context.Context,
	req repositories.SetPendingModificationsRequest,
) (*agent.AgentProposal, error) {
	r.saved = append(r.saved, req)

	return &agent.AgentProposal{ID: req.ID, PendingModifications: req.Modifications}, nil
}

type editsFixture struct {
	svc      *Service
	edits    *recordingEdits
	request  repositories.GetThreadRequest
	proposal *agent.AgentProposal
}

func newEditsFixture(status agent.ProposalStatus) *editsFixture {
	threadID := pulid.MustNew("thr_")
	proposal := &agent.AgentProposal{
		ID:       pulid.MustNew("ap_"),
		RunID:    pulid.MustNew("ar_"),
		ToolName: "send_email",
		Status:   status,
		ToolParams: map[string]any{
			"subject": "Proof of delivery",
			"body":    "Could you send the POD?",
		},
	}
	edits := &recordingEdits{}
	svc := newProposalService(
		&stubRunRepo{},
		&stubProposalRepo{byThread: []*agent.AgentProposal{proposal}},
		&stubConversationRepo{thread: &conversation.Thread{ID: threadID}},
	)
	svc.proposalEdits = edits

	return &editsFixture{
		svc:      svc,
		edits:    edits,
		request:  repositories.GetThreadRequest{ID: threadID},
		proposal: proposal,
	}
}

// What a person rewrote on a draft has to outlive the page, so it is kept on
// the proposal, and only what differs from the agent's wording is kept.
func TestSaveProposalEdits_KeepsOnlyWhatChanged(t *testing.T) {
	t.Parallel()

	f := newEditsFixture(agent.ProposalStatusPending)

	saved, err := f.svc.SaveProposalEdits(t.Context(), f.request, f.proposal.ID, map[string]any{
		"subject": "PO needed",
		"body":    "Could you send the POD?",
	})
	require.NoError(t, err)

	require.Len(t, f.edits.saved, 1)
	assert.Equal(t, f.proposal.ID, f.edits.saved[0].ID)
	assert.Equal(t, map[string]any{"subject": "PO needed"}, f.edits.saved[0].Modifications)
	assert.Equal(t, f.proposal.ID, saved.ProposalID)
	assert.Equal(t, map[string]any{"subject": "PO needed"}, saved.PendingModifications)
}

// Putting the agent's wording back is clearing the edit, not saving a copy of
// what was proposed.
func TestSaveProposalEdits_ClearsWhenNothingDiffers(t *testing.T) {
	t.Parallel()

	f := newEditsFixture(agent.ProposalStatusPending)

	saved, err := f.svc.SaveProposalEdits(t.Context(), f.request, f.proposal.ID, map[string]any{
		"subject": "Proof of delivery",
	})
	require.NoError(t, err)

	require.Len(t, f.edits.saved, 1)
	assert.Nil(t, f.edits.saved[0].Modifications)
	assert.Nil(t, saved.PendingModifications)
}

// An edit kept on a decided proposal would go with no approval.
func TestSaveProposalEdits_RefusesADecidedProposal(t *testing.T) {
	t.Parallel()

	f := newEditsFixture(agent.ProposalStatusAccepted)

	_, err := f.svc.SaveProposalEdits(t.Context(), f.request, f.proposal.ID, map[string]any{
		"subject": "PO needed",
	})

	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Empty(t, f.edits.saved)
}

// A proposal the conversation did not raise is not found, whoever's it is.
func TestSaveProposalEdits_RefusesAProposalFromAnotherConversation(t *testing.T) {
	t.Parallel()

	f := newEditsFixture(agent.ProposalStatusPending)

	_, err := f.svc.SaveProposalEdits(t.Context(), f.request, pulid.MustNew("ap_"), map[string]any{
		"subject": "PO needed",
	})

	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))
	assert.Empty(t, f.edits.saved)
}

// Reading the thread is the authorization check: someone else's conversation
// is not found, and nothing is saved.
func TestSaveProposalEdits_RefusesAThreadTheReaderCannotSee(t *testing.T) {
	t.Parallel()

	f := newEditsFixture(agent.ProposalStatusPending)
	f.svc.conversations = &stubConversationRepo{getErr: errors.New("thread not found")}

	_, err := f.svc.SaveProposalEdits(t.Context(), f.request, f.proposal.ID, map[string]any{
		"subject": "PO needed",
	})

	require.Error(t, err)
	assert.Empty(t, f.edits.saved)
}

// The parameter naming the record a tool acts on is shown but not editable,
// and a parameter the tool does not take cannot be smuggled in.
func TestSaveProposalEdits_RefusesAParameterThatIsNotEditable(t *testing.T) {
	t.Parallel()

	f := newEditsFixture(agent.ProposalStatusPending)
	f.proposal.ToolName = "transfer_to_billing"
	f.proposal.ToolParams = map[string]any{"billType": "Invoice"}
	f.svc.tools = subsetTools{}

	_, err := f.svc.SaveProposalEdits(t.Context(), f.request, f.proposal.ID, map[string]any{
		"notAParameter": "x",
	})

	require.Error(t, err)
	assert.Empty(t, f.edits.saved)

	_, err = f.svc.SaveProposalEdits(t.Context(), f.request, f.proposal.ID, map[string]any{
		"billType": "CreditMemo",
	})
	require.NoError(t, err)
	require.Len(t, f.edits.saved, 1)
	assert.Equal(t, map[string]any{"billType": "CreditMemo"}, f.edits.saved[0].Modifications)
}

// The list carries what was saved while the proposal waits, and nothing once
// it is decided: the decision's own modifications say what was approved.
func TestToAssistantProposal_CarriesPendingModificationsOnlyWhilePending(t *testing.T) {
	t.Parallel()

	pending := toAssistantProposal(&agent.AgentProposal{
		ID:                   pulid.MustNew("ap_"),
		Status:               agent.ProposalStatusPending,
		PendingModifications: map[string]any{"subject": "PO needed"},
	}, nil)
	assert.Equal(t, map[string]any{"subject": "PO needed"}, pending.PendingModifications)

	decided := toAssistantProposal(&agent.AgentProposal{
		ID:                   pulid.MustNew("ap_"),
		Status:               agent.ProposalStatusAccepted,
		PendingModifications: map[string]any{"subject": "PO needed"},
	}, nil)
	assert.Nil(t, decided.PendingModifications)
}
