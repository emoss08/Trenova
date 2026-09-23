package assistantturnservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// stubTurns is the turn's record. Status is what a re-read returns, which is
// how the relay learns a turn ended without saying so.
type stubTurns struct {
	turn *conversation.AssistantTurn
	err  error
	gets int
}

func (s *stubTurns) Start(
	context.Context, *conversation.AssistantTurn,
) (*conversation.AssistantTurn, error) {
	return s.turn, nil
}

func (s *stubTurns) GetByID(
	context.Context, repositories.GetAssistantTurnRequest,
) (*conversation.AssistantTurn, error) {
	s.gets++
	if s.err != nil {
		return nil, s.err
	}

	return s.turn, nil
}

func (s *stubTurns) Active(
	context.Context, repositories.ActiveAssistantTurnRequest,
) (*conversation.AssistantTurn, error) {
	return nil, nil
}

func (s *stubTurns) Complete(context.Context, repositories.CompleteAssistantTurnRequest) error {
	return nil
}

func (s *stubTurns) MarkWorkflow(
	context.Context, pulid.ID, pagination.TenantInfo, string,
) error {
	return nil
}

// stubReader stands in for the turn's workflow stream. After the frames it
// returns nil, which is what a stream does when the workflow has closed.
type stubReader struct {
	frames []serviceports.TurnStreamFrame
	err    error
	reads  int
	ref    serviceports.TurnStreamRef
	cursor string
}

func (s *stubReader) Read(
	_ context.Context,
	req serviceports.ReadTurnStreamRequest,
) error {
	s.reads++
	s.ref = req.Ref
	s.cursor = req.Cursor
	for _, frame := range s.frames {
		if err := req.OnFrame(frame); err != nil {
			return err
		}
		if frame.Terminal() {
			return nil
		}
	}

	return s.err
}

func newRelay(turns *stubTurns, reader *stubReader) *Service {
	return &Service{l: zap.NewNop(), turns: turns, reader: reader}
}

func runningTurn() *conversation.AssistantTurn {
	return &conversation.AssistantTurn{
		ID:             pulid.MustNew("atrn_"),
		ThreadID:       pulid.MustNew("athr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Status:         conversation.AssistantTurnStatusRunning,
		WorkflowID:     "assistant-turn:atrn_1",
	}
}

func collect(
	t *testing.T,
	svc *Service,
	turn *conversation.AssistantTurn,
) []serviceports.TurnStreamFrame {
	t.Helper()

	seen := make([]serviceports.TurnStreamFrame, 0, 4)
	err := svc.Relay(
		t.Context(),
		RelayRequest{Turn: turn},
		func(f serviceports.TurnStreamFrame) error {
			seen = append(seen, f)

			return nil
		},
	)
	require.NoError(t, err)

	return seen
}

func TestRelay_ForwardsTheTurnsOwnEnding(t *testing.T) {
	t.Parallel()

	turn := runningTurn()
	reader := &stubReader{
		frames: []serviceports.TurnStreamFrame{
			{ID: "0", Event: serviceports.AssistantEventDelta, Data: []byte(`{"text":"hi"}`)},
			{ID: "1", Event: serviceports.AssistantEventDone, Data: []byte(`{}`)},
		},
	}
	svc := newRelay(&stubTurns{turn: turn}, reader)

	seen := collect(t, svc, turn)

	require.Len(t, seen, 2)
	assert.Equal(t, "1", seen[1].ID, "the turn's own ending is forwarded, cursor and all")
	assert.Equal(
		t,
		turn.WorkflowID,
		reader.ref.WorkflowID,
		"the stream is read from the turn's workflow",
	)
}

// The workflow closed before the reader caught up: a reader who fell behind,
// or came back late. The turn's record says how it ended.
func TestRelay_ClosesAReaderWhoseTurnClosedBeforeTheyCaughtUp(t *testing.T) {
	t.Parallel()

	turn := runningTurn()
	record := &conversation.AssistantTurn{
		ID:       turn.ID,
		ThreadID: turn.ThreadID,
		Status:   conversation.AssistantTurnStatusCompleted,
	}
	svc := newRelay(&stubTurns{turn: record}, &stubReader{
		frames: []serviceports.TurnStreamFrame{
			{ID: "0", Event: serviceports.AssistantEventDelta, Data: []byte(`{"text":"hi"}`)},
		},
	})

	seen := collect(t, svc, turn)

	require.Len(t, seen, 2)
	assert.Equal(t, serviceports.AssistantEventDone, seen[1].Event)
	assert.Contains(t, string(seen[1].Data), `"replay":true`,
		"the client is told this ending was reconstructed, so it refetches the thread")
	assert.Empty(t, seen[1].ID, "an invented frame has no position to resume from")
}

func TestRelay_ResumesFromTheReadersCursor(t *testing.T) {
	t.Parallel()

	turn := runningTurn()
	reader := &stubReader{frames: []serviceports.TurnStreamFrame{
		{ID: "8", Event: serviceports.AssistantEventDone, Data: []byte(`{}`)},
	}}
	svc := newRelay(&stubTurns{turn: turn}, reader)

	err := svc.Relay(t.Context(), RelayRequest{Turn: turn, Cursor: "7"},
		func(serviceports.TurnStreamFrame) error { return nil })
	require.NoError(t, err)

	assert.Equal(t, "7", reader.cursor)
}

// A reader who leaves is not a failure of the turn.
func TestRelay_IsQuietWhenTheReaderLeaves(t *testing.T) {
	t.Parallel()

	turn := runningTurn()
	svc := newRelay(&stubTurns{turn: turn}, &stubReader{err: context.Canceled})

	err := svc.Relay(t.Context(), RelayRequest{Turn: turn},
		func(serviceports.TurnStreamFrame) error { return nil })

	assert.NoError(t, err)
}

func TestRelay_ReportsAStreamThatCouldNotBeRead(t *testing.T) {
	t.Parallel()

	turn := runningTurn()
	svc := newRelay(&stubTurns{turn: turn}, &stubReader{err: errors.New("temporal is down")})

	err := svc.Relay(t.Context(), RelayRequest{Turn: turn},
		func(serviceports.TurnStreamFrame) error { return nil })

	assert.Error(t, err)
}

// A failed turn is not dressed up as a finished one.
func TestRelay_SaysAFailedTurnDidNotFinish(t *testing.T) {
	t.Parallel()

	turn := runningTurn()
	turn.Status = conversation.AssistantTurnStatusFailed
	turn.ErrorMessage = "provider down"
	svc := newRelay(&stubTurns{turn: turn}, &stubReader{})

	seen := collect(t, svc, turn)

	require.Len(t, seen, 1)
	assert.Equal(t, serviceports.AssistantEventError, seen[0].Event)
}

// A turn already over before anybody attached needs no stream at all.
func TestRelay_AnswersATurnThatWasAlreadyOverWithoutReadingTheStream(t *testing.T) {
	t.Parallel()

	turn := runningTurn()
	turn.Status = conversation.AssistantTurnStatusRefused
	reader := &stubReader{}
	svc := newRelay(&stubTurns{turn: turn}, reader)

	seen := collect(t, svc, turn)

	require.Len(t, seen, 1)
	assert.Contains(t, string(seen[0].Data), string(conversation.AssistantTurnStatusRefused))
	assert.Zero(t, reader.reads)
}

// A turn's execution is named after the turn, so a turn whose start was not
// recorded is still followed rather than reported as over.
func TestRelay_FollowsATurnWhoseStartWasNotRecorded(t *testing.T) {
	t.Parallel()

	turn := runningTurn()
	turn.WorkflowID = ""
	reader := &stubReader{frames: []serviceports.TurnStreamFrame{
		{ID: "0", Event: serviceports.AssistantEventDelta},
		{ID: "1", Event: serviceports.AssistantEventDone},
	}}
	svc := newRelay(&stubTurns{turn: turn}, reader)

	seen := collect(t, svc, turn)

	assert.Equal(t, 1, reader.reads)
	assert.Equal(t, conversation.AssistantTurnWorkflowID(turn.ID), reader.ref.WorkflowID)
	require.Len(t, seen, 2)
}

// An execution that cannot be read is no error when the record already says
// how the turn ended.
func TestRelay_ClosesWhenTheStreamIsGoneButTheRecordSaysItEnded(t *testing.T) {
	t.Parallel()

	turn := runningTurn()
	ended := *turn
	ended.Status = conversation.AssistantTurnStatusCompleted
	svc := newRelay(&stubTurns{turn: &ended}, &stubReader{err: errors.New("workflow not found")})

	seen := collect(t, svc, turn)

	require.Len(t, seen, 1)
	assert.True(t, seen[0].Terminal())
}
