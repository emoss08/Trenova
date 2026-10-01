package agentdecisionservice

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordedFollowUps struct {
	requests []services.DecisionFollowUpRequest
}

func (r *recordedFollowUps) FollowUp(_ context.Context, req services.DecisionFollowUpRequest) {
	r.requests = append(r.requests, req)
}

func (h *decideHarness) decideWithNote(
	t *testing.T,
	decision agent.DecisionType,
	note string,
) (*services.DecisionOutcome, error) {
	t.Helper()

	return h.service.DecideWithOutcome(t.Context(), &services.DecideAgentProposalRequest{
		ProposalID: h.proposals.proposal.ID,
		Decision:   decision,
		ReasonCode: "rejected_in_conversation",
		Note:       note,
		TenantInfo: pagination.TenantInfo{
			OrgID: h.actor.OrganizationID,
			BuID:  h.actor.BusinessUnitID,
		},
	}, h.actor)
}

func TestDecide_RecordsWhatThePersonToldTheAgent(t *testing.T) {
	t.Parallel()

	h := newDecideHarness(t)
	followUps := &recordedFollowUps{}
	h.service.followUps = followUps

	_, err := h.decideWithNote(t, agent.DecisionRejected,
		"  Hold it until the customer calls back, not today.  ")
	require.NoError(t, err)

	require.NotNil(t, h.decisions.created)
	assert.Equal(t, "Hold it until the customer calls back, not today.", h.decisions.created.Note,
		"the note is kept on the decision, trimmed")
	assert.Equal(t, "rejected_in_conversation", h.decisions.created.ReasonCode,
		"the note does not replace the reason code")
	require.Len(t, followUps.requests, 1,
		"telling the agent is the rejection's own follow-up, never a second turn")
	assert.Equal(t, h.proposals.proposal.ID, followUps.requests[0].ProposalID)
}

func TestDecide_KeepsNoNoteWhenNoneWasWritten(t *testing.T) {
	t.Parallel()

	h := newDecideHarness(t)
	_, err := h.decideWithNote(t, agent.DecisionRejected, "   ")
	require.NoError(t, err)

	require.NotNil(t, h.decisions.created)
	assert.Empty(t, h.decisions.created.Note)
}

func TestDecide_RefusesANoteLongerThanTheBound(t *testing.T) {
	t.Parallel()

	h := newDecideHarness(t)
	_, err := h.decideWithNote(t, agent.DecisionRejected,
		strings.Repeat("é", agent.MaxDecisionNoteLength+1))

	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Equal(t, "note", multiErr.Errors[0].Field)
	assert.Nil(t, h.decisions.created, "nothing is recorded")
}

func TestDecide_TakesANoteOfExactlyTheBound(t *testing.T) {
	t.Parallel()

	h := newDecideHarness(t)
	_, err := h.decideWithNote(t, agent.DecisionRejected,
		strings.Repeat("é", agent.MaxDecisionNoteLength))
	require.NoError(t, err)

	require.NotNil(t, h.decisions.created)
	assert.Equal(t, agent.MaxDecisionNoteLength, len([]rune(h.decisions.created.Note)),
		"the bound counts characters, not bytes")
}
