package agentplanservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecide_RecordsTheNoteOnEveryRejectedStep(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 3, false)
	h.req.Decision = agent.DecisionRejected
	h.req.Note = "Do the second load first; the first can wait for Monday."

	_, err := h.svc.Decide(t.Context(), h.req, h.actor)
	require.NoError(t, err)

	require.Len(t, h.decider.decided, 3)
	for _, decided := range h.decider.decided {
		assert.Equal(t, h.req.Note, decided.Note,
			"the plan's follow-up reads the note from any of its steps")
	}
}

func TestDecide_CarriesTheNoteIntoEveryApprovedStep(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 2, false)
	h.req.Note = "Go ahead, and tell dispatch."

	_, err := h.svc.Decide(t.Context(), h.req, h.actor)
	require.NoError(t, err)

	require.Len(t, h.decider.decided, 2)
	for _, decided := range h.decider.decided {
		assert.Equal(t, h.req.Note, decided.Note)
	}
}

func TestDecide_RefusesAnOverlongNoteBeforeClaimingThePlan(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 2, false)
	h.req.Decision = agent.DecisionRejected
	h.req.Note = strings.Repeat("x", agent.MaxDecisionNoteLength+1)

	_, err := h.svc.Decide(t.Context(), h.req, h.actor)

	require.Error(t, err)
	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "note", validation.Field)
	assert.Empty(t, h.plans.statuses, "the plan is not claimed")
	assert.Empty(t, h.decider.decided, "no step is decided")
}
