package assistantturnservice

import (
	"context"
	"errors"
	"testing"
	"time"

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
	return &Service{l: zap.NewNop(), turns: turns, running: newRunningTurns()}
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

func TestStartDurable_RecordsTheWorkflowCarryingTheTurn(t *testing.T) {
	t.Parallel()

	turns := &recordingTurns{}
	svc := newDurable(turns)

	turn, err := svc.StartDurable(t.Context(), startRequest(), func(*conversation.AssistantTurn) (string, error) {
		return "assistant-turn:atrn_1", nil
	})
	require.NoError(t, err)

	assert.Equal(t, "assistant-turn:atrn_1", turns.marked)
	assert.Equal(t, "assistant-turn:atrn_1", turn.WorkflowID,
		"the caller needs it to stop the turn without reading it back")
	assert.Empty(t, turns.completed)
}

// A turn left Running with nothing running it holds the conversation's one
// live slot until it expires, and refuses every later question.
func TestStartDurable_ClosesTheRecordWhenNothingPickedTheTurnUp(t *testing.T) {
	t.Parallel()

	turns := &recordingTurns{}
	svc := newDurable(turns)

	_, err := svc.StartDurable(t.Context(), startRequest(), func(*conversation.AssistantTurn) (string, error) {
		return "", errors.New("temporal is unreachable")
	})
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

// A turn running in an API process has no execution to cancel, and nobody's
// request may be carrying it: a decision's follow-up, or one a reader rejoined
// after a reload. Stopping it closes its record, which frees the conversation
// and is what every instance watching the turn reads.
func TestStop_ClosesAnInProcessTurnsRecord(t *testing.T) {
	t.Parallel()

	turns := &recordingTurns{}
	svc := newDurable(turns)
	turn := &conversation.AssistantTurn{
		ID:     pulid.MustNew("atrn_"),
		Status: conversation.AssistantTurnStatusRunning,
	}

	called := false
	require.NoError(t, svc.Stop(t.Context(), turn, func(string) error {
		called = true

		return nil
	}))

	assert.False(t, called, "there is no workflow to cancel")
	require.Len(t, turns.completed, 1)
	assert.Equal(t, turn.ID, turns.completed[0].ID)
	assert.Equal(t, conversation.AssistantTurnStatusStopped, turns.completed[0].Status)
}

func TestStop_CancelsAnInProcessTurnRunningHere(t *testing.T) {
	t.Parallel()

	turn := &conversation.AssistantTurn{
		ID:     pulid.MustNew("atrn_"),
		Status: conversation.AssistantTurnStatusRunning,
	}
	svc := newDurable(&recordingTurns{stubTurns: stubTurns{turn: turn}})

	runCtx, release := svc.Stoppable(t.Context(), turn)
	defer release()

	require.NoError(t, svc.Stop(t.Context(), turn, nil))

	select {
	case <-runCtx.Done():
		assert.ErrorIs(t, runCtx.Err(), context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("a stopped turn running in this process kept running")
	}
}

// A Stop asked of another instance reaches this one through the record.
func TestStoppable_CancelsWhenTheRecordSaysTheTurnWasStopped(t *testing.T) {
	t.Parallel()

	turn := &conversation.AssistantTurn{
		ID:     pulid.MustNew("atrn_"),
		Status: conversation.AssistantTurnStatusRunning,
	}
	record := *turn
	record.Status = conversation.AssistantTurnStatusStopped
	svc := newDurable(&recordingTurns{stubTurns: stubTurns{turn: &record}})

	runCtx, release := svc.Stoppable(t.Context(), turn)
	defer release()

	select {
	case <-runCtx.Done():
	case <-time.After(stopPollInterval + 3*time.Second):
		t.Fatal("a turn stopped elsewhere kept running")
	}
	assert.Equal(t, conversation.AssistantTurnStatusRunning, turn.Status,
		"the caller's turn is not written to by the watch")
}

func TestStoppable_LeavesARunningTurnAlone(t *testing.T) {
	t.Parallel()

	turn := &conversation.AssistantTurn{
		ID:     pulid.MustNew("atrn_"),
		Status: conversation.AssistantTurnStatusRunning,
	}
	svc := newDurable(&recordingTurns{stubTurns: stubTurns{turn: turn}})

	runCtx, release := svc.Stoppable(t.Context(), turn)
	defer release()

	select {
	case <-runCtx.Done():
		t.Fatal("a running turn was cancelled")
	case <-time.After(stopPollInterval + 500*time.Millisecond):
	}
}
