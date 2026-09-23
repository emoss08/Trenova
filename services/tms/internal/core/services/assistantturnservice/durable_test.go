package assistantturnservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// recordingTurns remembers how a turn was closed, which is the thing worth
// asserting when a start fails.
type recordingTurns struct {
	stubTurns

	completed []repositories.CompleteAssistantTurnRequest
	marked    string
	startErr  error
}

func (r *recordingTurns) Start(
	_ context.Context, turn *conversation.AssistantTurn,
) (*conversation.AssistantTurn, error) {
	if r.startErr != nil {
		return nil, r.startErr
	}
	if turn.ID.IsNil() {
		turn.ID = pulid.MustNew("atrn_")
	}
	r.turn = turn

	return turn, nil
}

func (r *recordingTurns) Complete(
	_ context.Context, req repositories.CompleteAssistantTurnRequest,
) error {
	r.completed = append(r.completed, req)

	return nil
}

func (r *recordingTurns) MarkWorkflow(
	_ context.Context, _ pulid.ID, _ pagination.TenantInfo, workflowID string,
) error {
	r.marked = workflowID

	return nil
}

func newDurable(turns *recordingTurns) *Service {
	return &Service{l: zap.NewNop(), turns: turns}
}

func startRequest() StartRequest {
	return StartRequest{
		ThreadID: pulid.MustNew("athr_"),
		UserID:   pulid.MustNew("usr_"),
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
	}
}

func TestStartTurn_RecordsTheWorkflowCarryingTheTurn(t *testing.T) {
	t.Parallel()

	turns := &recordingTurns{}
	svc := newDurable(turns)

	turn, err := svc.StartTurn(
		t.Context(),
		startRequest(),
		func(*conversation.AssistantTurn) (string, error) {
			return "assistant-turn:atrn_1", nil
		},
	)
	require.NoError(t, err)

	assert.Equal(t, "assistant-turn:atrn_1", turns.marked)
	assert.Equal(t, "assistant-turn:atrn_1", turn.WorkflowID,
		"the caller needs it to stop the turn without reading it back")
	assert.Empty(t, turns.completed)
}

// A turn left Running with nothing running it holds the conversation's one
// live slot until it expires, and refuses every later question.
func TestStartTurn_ClosesTheRecordWhenNothingPickedTheTurnUp(t *testing.T) {
	t.Parallel()

	turns := &recordingTurns{}
	svc := newDurable(turns)

	_, err := svc.StartTurn(
		t.Context(),
		startRequest(),
		func(*conversation.AssistantTurn) (string, error) {
			return "", errors.New("temporal is unreachable")
		},
	)
	require.Error(t, err)

	require.Len(t, turns.completed, 1)
	assert.Equal(t, conversation.AssistantTurnStatusFailed, turns.completed[0].Status)
}

func TestStop_CancelsTheExecutionCarryingTheTurn(t *testing.T) {
	t.Parallel()

	svc := newDurable(&recordingTurns{})
	turn := &conversation.AssistantTurn{
		ID:         pulid.MustNew("atrn_"),
		Status:     conversation.AssistantTurnStatusRunning,
		WorkflowID: "assistant-turn:atrn_1",
	}

	cancelled := ""
	require.NoError(t, svc.Stop(t.Context(), turn, func(id string) error {
		cancelled = id

		return nil
	}))

	assert.Equal(t, "assistant-turn:atrn_1", cancelled)
}

// Stopping something already finished is not a failure. The person got what
// they were asking for.
func TestStop_IsQuietAboutATurnThatHasAlreadyEnded(t *testing.T) {
	t.Parallel()

	svc := newDurable(&recordingTurns{})
	turn := &conversation.AssistantTurn{
		Status:     conversation.AssistantTurnStatusCompleted,
		WorkflowID: "assistant-turn:atrn_1",
	}

	called := false
	require.NoError(t, svc.Stop(t.Context(), turn, func(string) error {
		called = true

		return nil
	}))

	assert.False(t, called)
}

// A turn whose start never reached a worker has no execution to cancel.
func TestStop_HasNothingToCancelForATurnNothingPickedUp(t *testing.T) {
	t.Parallel()

	svc := newDurable(&recordingTurns{})
	turn := &conversation.AssistantTurn{Status: conversation.AssistantTurnStatusRunning}

	called := false
	require.NoError(t, svc.Stop(t.Context(), turn, func(string) error {
		called = true

		return nil
	}))

	assert.False(t, called)
}
