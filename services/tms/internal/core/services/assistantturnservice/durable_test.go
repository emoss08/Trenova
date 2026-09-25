package assistantturnservice

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
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

// stopWith stops a turn with cancel standing in for the engine running it.
func stopWith(
	ctx context.Context,
	svc *Service,
	turn *conversation.AssistantTurn,
	cancel func(workflowID string) error,
) error {
	svc.canceller = serviceports.AssistantTurnCancellerFunc(
		func(_ context.Context, workflowID string) error {
			return cancel(workflowID)
		},
	)

	return svc.Stop(ctx, turn)
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
	require.NoError(t, stopWith(t.Context(), svc, turn, func(id string) error {
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
	require.NoError(t, stopWith(t.Context(), svc, turn, func(string) error {
		called = true

		return nil
	}))

	assert.False(t, called)
}

// A turn recorded but never handed to a worker has no execution to cancel.
// Stopping it closes its record, which frees the conversation's one live slot.
func TestStop_ClosesTheRecordOfATurnNothingPickedUp(t *testing.T) {
	t.Parallel()

	turns := &recordingTurns{}
	svc := newDurable(turns)
	turn := &conversation.AssistantTurn{
		ID:     pulid.MustNew("atrn_"),
		Status: conversation.AssistantTurnStatusRunning,
	}

	require.NoError(t, stopWith(t.Context(), svc, turn, func(string) error {
		return ErrNoExecution
	}))

	require.Len(t, turns.completed, 1)
	assert.Equal(t, turn.ID, turns.completed[0].ID)
	assert.Equal(t, conversation.AssistantTurnStatusStopped, turns.completed[0].Status)
}

/*
A turn whose start was not recorded used to be closed without cancelling
anything, on the reading that nothing had started it. The start is recorded
after the workflow begins, so the workflow could be running: the record said
Stopped while the reply went on writing into the conversation.
*/
func TestStop_CancelsATurnWhoseStartWasNotRecorded(t *testing.T) {
	t.Parallel()

	turns := &recordingTurns{}
	svc := newDurable(turns)
	turn := &conversation.AssistantTurn{
		ID:     pulid.MustNew("atrn_"),
		Status: conversation.AssistantTurnStatusRunning,
	}

	cancelled := ""
	require.NoError(t, stopWith(t.Context(), svc, turn, func(id string) error {
		cancelled = id

		return nil
	}))

	assert.Equal(t, conversation.AssistantTurnWorkflowID(turn.ID), cancelled)
	assert.Empty(t, turns.completed, "the execution closes the record as it stops")
}

// An execution that ended without closing the record leaves nothing to
// cancel. The stop still frees the conversation.
func TestStop_ClosesTheRecordWhenTheExecutionIsGone(t *testing.T) {
	t.Parallel()

	turns := &recordingTurns{}
	svc := newDurable(turns)
	turn := &conversation.AssistantTurn{
		ID:         pulid.MustNew("atrn_"),
		Status:     conversation.AssistantTurnStatusRunning,
		WorkflowID: "assistant-turn:atrn_1",
	}

	require.NoError(t, stopWith(t.Context(), svc, turn, func(string) error {
		return fmt.Errorf("cancel: %w", ErrNoExecution)
	}))

	require.Len(t, turns.completed, 1)
	assert.Equal(t, conversation.AssistantTurnStatusStopped, turns.completed[0].Status)
}

func TestStop_ReportsACancelThatFailed(t *testing.T) {
	t.Parallel()

	turns := &recordingTurns{}
	svc := newDurable(turns)
	turn := &conversation.AssistantTurn{
		ID:         pulid.MustNew("atrn_"),
		Status:     conversation.AssistantTurnStatusRunning,
		WorkflowID: "assistant-turn:atrn_1",
	}

	err := stopWith(t.Context(), svc, turn, func(string) error {
		return errors.New("temporal is down")
	})

	require.Error(t, err)
	assert.Empty(t, turns.completed)
}

func (r *recordingTurns) RecordFingerprint(
	context.Context,
	repositories.RecordAssistantTurnFingerprintRequest,
) error {
	return nil
}

func TestStart_RecordsTheTurnsTrace(t *testing.T) {
	t.Parallel()

	turns := &recordingTurns{}
	turn, err := newDurable(turns).Start(t.Context(), startRequest())
	require.NoError(t, err)

	require.True(t, turn.ID.IsNotNil(), "the id is known before the row is written")
	assert.Equal(t,
		aitrace.AnchorFor(aitrace.AnchorAssistantTurn, turn.ID.String()).TraceID.String(),
		turn.TraceID,
		"the turn names the trace its workflow is anchored in")
}
