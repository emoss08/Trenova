package assistantturnservice

import (
	"context"
	"errors"
	"testing"
	"time"

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

// stubReader stands in for the redis stream.
type stubReader struct {
	exists bool
	frames []serviceports.TurnStreamFrame
	// idles is how many empty read windows pass before the frames run out.
	idles int
}

func (s *stubReader) Exists(context.Context, serviceports.TurnStreamRef) (bool, error) {
	return s.exists, nil
}

func (s *stubReader) Read(
	_ context.Context,
	req serviceports.ReadTurnStreamRequest,
) error {
	for _, frame := range s.frames {
		if err := req.OnFrame(frame); err != nil {
			return err
		}
		if frame.Terminal() {
			return nil
		}
	}

	for range s.idles {
		if req.OnIdle == nil {
			break
		}
		if err := req.OnIdle(); err != nil {
			return err
		}
	}

	return nil
}

func newRelay(turns *stubTurns, reader *stubReader) *Service {
	return &Service{l: zap.NewNop(), turns: turns, reader: reader, running: newRunningTurns()}
}

func runningTurn() *conversation.AssistantTurn {
	return &conversation.AssistantTurn{
		ID:             pulid.MustNew("atrn_"),
		ThreadID:       pulid.MustNew("athr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Status:         conversation.AssistantTurnStatusRunning,
	}
}

func collect(t *testing.T, svc *Service, turn *conversation.AssistantTurn) []serviceports.TurnStreamFrame {
	t.Helper()

	seen := make([]serviceports.TurnStreamFrame, 0, 4)
	err := svc.Relay(t.Context(), RelayRequest{Turn: turn}, func(f serviceports.TurnStreamFrame) error {
		seen = append(seen, f)

		return nil
	})
	require.NoError(t, err)

	return seen
}

func TestRelay_ForwardsTheTurnsOwnEnding(t *testing.T) {
	t.Parallel()

	turn := runningTurn()
	svc := newRelay(&stubTurns{turn: turn}, &stubReader{
		exists: true,
		frames: []serviceports.TurnStreamFrame{
			{ID: "1-0", Event: serviceports.AssistantEventDelta, Data: []byte(`{"text":"hi"}`)},
			{ID: "1-1", Event: serviceports.AssistantEventDone, Data: []byte(`{}`)},
		},
	})

	seen := collect(t, svc, turn)

	require.Len(t, seen, 2)
	assert.Equal(t, "1-1", seen[1].ID, "the turn's own ending is forwarded, cursor and all")
}

// The writer died between its last event and its ending. Silence alone cannot
// prove that — a model thinking for a minute is also silent — so the turn's
// record is what settles it.
func TestRelay_ClosesAReaderWhoseTurnDiedWithoutSayingSo(t *testing.T) {
	t.Parallel()

	turn := runningTurn()
	record := &conversation.AssistantTurn{
		ID:       turn.ID,
		ThreadID: turn.ThreadID,
		Status:   conversation.AssistantTurnStatusCompleted,
	}
	svc := newRelay(&stubTurns{turn: record}, &stubReader{
		exists: true,
		frames: []serviceports.TurnStreamFrame{
			{ID: "1-0", Event: serviceports.AssistantEventDelta, Data: []byte(`{"text":"hi"}`)},
		},
		idles: 3,
	})

	seen := collect(t, svc, turn)

	require.Len(t, seen, 2)
	assert.Equal(t, serviceports.AssistantEventDone, seen[1].Event)
	assert.Contains(t, string(seen[1].Data), `"replay":true`,
		"the client is told this ending was reconstructed, so it refetches the thread")
	assert.Empty(t, seen[1].ID, "an invented frame has no position to resume from")
}

// A quiet turn is not a dead one. The relay must keep waiting rather than
// inventing an ending for a model that is still thinking.
func TestRelay_KeepsWaitingWhileTheTurnIsStillRunning(t *testing.T) {
	t.Parallel()

	turn := runningTurn()
	turns := &stubTurns{turn: turn}
	svc := newRelay(turns, &stubReader{exists: true, idles: 3})

	seen := collect(t, svc, turn)

	assert.Empty(t, seen, "nothing is invented for a turn that is merely quiet")
	assert.Equal(t, 3, turns.gets, "every quiet interval re-reads the record")
}

// A database that hiccups must not end somebody's reply.
func TestRelay_TreatsAnUnreadableRecordAsStillRunning(t *testing.T) {
	t.Parallel()

	turn := runningTurn()
	svc := newRelay(
		&stubTurns{turn: turn, err: errors.New("connection refused")},
		&stubReader{exists: true, idles: 2},
	)

	seen := collect(t, svc, turn)

	assert.Empty(t, seen)
}

// The events are a tail buffer; the conversation is permanent. A reader who
// comes back after the buffer aged out is told where the answer is, not that
// the reply failed.
func TestRelay_SendsAReaderToTheConversationWhenTheStreamHasExpired(t *testing.T) {
	t.Parallel()

	turn := runningTurn()
	record := &conversation.AssistantTurn{
		ID:       turn.ID,
		ThreadID: turn.ThreadID,
		Status:   conversation.AssistantTurnStatusCompleted,
	}
	svc := newRelay(&stubTurns{turn: record}, &stubReader{exists: false})

	seen := collect(t, svc, turn)

	require.Len(t, seen, 1)
	assert.Equal(t, serviceports.AssistantEventDone, seen[0].Event)
	assert.Contains(t, string(seen[0].Data), turn.ThreadID.String())
}

// A turn's stream is made by its first event, and a reader can attach before
// that: a decision's follow-up is recorded before the decision returns. The
// reader waits for the stream rather than being told it expired.
func TestRelay_FollowsARunningTurnWhoseStreamHasNotBegun(t *testing.T) {
	t.Parallel()

	turn := runningTurn()
	turn.StartedAt = time.Now().Unix()
	svc := newRelay(&stubTurns{turn: turn}, &stubReader{
		exists: false,
		frames: []serviceports.TurnStreamFrame{
			frameOf(serviceports.AssistantEventDelta, map[string]any{"text": "On it"}),
			frameOf(serviceports.AssistantEventDone, map[string]any{"turnId": turn.ID.String()}),
		},
	})

	seen := collect(t, svc, turn)

	require.Len(t, seen, 2)
	assert.Equal(t, serviceports.AssistantEventDelta, seen[0].Event)
	assert.Equal(t, serviceports.AssistantEventDone, seen[1].Event)
}

func TestRelay_SaysSoWhenALiveTurnHasLostItsStream(t *testing.T) {
	t.Parallel()

	turn := runningTurn()
	turn.StartedAt = time.Now().Add(-10 * time.Minute).Unix()
	svc := newRelay(&stubTurns{turn: turn}, &stubReader{exists: false})

	seen := collect(t, svc, turn)

	require.Len(t, seen, 1)
	assert.Equal(t, serviceports.AssistantEventError, seen[0].Event)
	assert.Contains(t, string(seen[0].Data), "still being written")
}

// A turn already over before anybody attached needs no stream at all.
func TestRelay_AnswersATurnThatWasAlreadyOverWithoutReadingTheStream(t *testing.T) {
	t.Parallel()

	turn := runningTurn()
	turn.Status = conversation.AssistantTurnStatusRefused
	reader := &stubReader{exists: true}
	svc := newRelay(&stubTurns{turn: turn}, reader)

	seen := collect(t, svc, turn)

	require.Len(t, seen, 1)
	assert.Contains(t, string(seen[0].Data), string(conversation.AssistantTurnStatusRefused))
}
