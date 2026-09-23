package assistantturnservice

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"go.uber.org/zap"
)

// RelayRequest asks to follow one turn from a position in it.
type RelayRequest struct {
	Turn *conversation.AssistantTurn
	// Cursor is the last frame the reader applied. Empty replays the turn
	// from its beginning, which is what a fresh tab attaching to a reply
	// already in progress wants.
	Cursor string
}

// Relay follows a turn's events until it ends.
//
// There are three ways a turn stops producing, and a reader has to be told the
// truth about which one happened:
//
// The turn finished. Its last frame says so, the relay forwards it and stops.
//
// The turn is over but nothing said so — the writer died between its last
// event and its ending. Silence alone cannot prove this, because a model
// thinking for a minute is also silent, so every quiet interval re-reads the
// turn's record. A record that says the turn ended, with no frame to match,
// is the proof.
//
// The stream expired. The events are a tail buffer with an hour on them; the
// conversation is permanent. A reader who comes back after that is not told
// the reply failed — it did not — but that the answer is in the thread.
func (s *Service) Relay(
	ctx context.Context,
	req RelayRequest,
	onFrame serviceports.TurnFrameFunc,
) error {
	turn := req.Turn
	ref := streamRef(turn)

	// A turn that was already over before anybody attached needs no stream at
	// all: the answer is written down, and saying so immediately beats
	// blocking on a key that may not exist.
	if turn.Status.Terminal() {
		s.metrics.RecordStreamAttach("already_ended")

		return onFrame(closingFrame(turn))
	}

	live, err := s.reader.Exists(ctx, ref)
	if err != nil {
		return err
	}
	// No stream is not the same as an expired one. A turn's stream is made by
	// its first event, and a reader attaches the moment the turn exists: the
	// follow-up to a decision is recorded before the decision returns, and a
	// model can think for a minute before it says anything. A turn still
	// running is followed from wherever its stream begins; the read blocks
	// until it does, and the idle check below still ends it if the turn dies
	// first.
	if !live {
		if !s.stillRunning(ctx, turn) {
			s.metrics.RecordStreamAttach("expired")

			return onFrame(closingFrame(turn))
		}
		if streamAgedOut(turn) {
			// Running long past the point its stream would have begun, and
			// there is none: the events aged out from under a turn still
			// going. Say the live view is gone rather than invent an ending.
			s.metrics.RecordStreamAttach("expired")

			return onFrame(errorFrame(
				"This reply is still being written, but the live view of it has expired. " +
					"It will appear in the conversation when it finishes.",
			))
		}
	}

	if req.Cursor == "" {
		s.metrics.RecordStreamAttach("live")
	} else {
		// A cursor means somebody came back to a reply they had already
		// started watching, which is the recovery this work exists to make
		// possible and therefore the number worth counting.
		s.metrics.RecordStreamAttach("resumed")
	}

	// seen guards the ending: a turn whose record says it finished, whose
	// stream never said so, must still close the reader's connection.
	var sawTerminal bool
	err = s.reader.Read(ctx, serviceports.ReadTurnStreamRequest{
		Ref:    ref,
		Cursor: req.Cursor,
		OnFrame: func(frame serviceports.TurnStreamFrame) error {
			if frame.Terminal() {
				sawTerminal = true
			}

			return onFrame(frame)
		},
		OnIdle: func() error {
			if s.stillRunning(ctx, turn) {
				return nil
			}

			// The record says it is over and the stream never said so.
			if fErr := onFrame(closingFrame(turn)); fErr != nil {
				return fErr
			}

			return errRelayStopped
		},
	})
	switch {
	case err == nil, errors.Is(err, errRelayStopped):
		return nil
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		// The reader left. Nothing is wrong with the turn.
		return nil
	default:
		if sawTerminal {
			return nil
		}

		return err
	}
}

// stillRunning re-reads the turn's record. A read that fails is treated as
// still running: the relay keeps waiting rather than telling a reader their
// reply ended because the database hiccupped.
func (s *Service) stillRunning(ctx context.Context, turn *conversation.AssistantTurn) bool {
	current, err := s.turns.GetByID(ctx, repositories.GetAssistantTurnRequest{
		ID:         turn.ID,
		TenantInfo: tenantOf(turn),
	})
	if err != nil {
		s.l.Warn("could not check whether a turn is still running",
			zap.String("turn", turn.ID.String()),
			zap.Error(err),
		)

		return true
	}

	if current.Status.Terminal() {
		turn.Status = current.Status
		turn.ErrorMessage = current.ErrorMessage

		return false
	}

	if diedWithProcess(current) && s.reapStale(ctx, tenantOf(turn), turn.ThreadID) > 0 {
		// Nothing will ever write this turn's ending: the process running it
		// is gone. It is closed now, and the reader told so, rather than left
		// waiting on a stream nobody is writing.
		turn.Status = conversation.AssistantTurnStatusFailed
		turn.ErrorMessage = staleTurnError

		return false
	}

	return true
}

// diedWithProcess reports whether a live turn ran in an API process that has
// stopped heartbeating for it.
func diedWithProcess(turn *conversation.AssistantTurn) bool {
	if turn.WorkflowID != "" {
		return false
	}

	return turn.HeartbeatAt == 0 ||
		time.Since(time.Unix(turn.HeartbeatAt, 0)) > staleAfter
}

// streamStartGrace is how long a running turn may go without a stream before
// its absence means the stream expired rather than has not begun.
const streamStartGrace = 2 * time.Minute

func streamAgedOut(turn *conversation.AssistantTurn) bool {
	if turn.StartedAt <= 0 {
		return false
	}

	return time.Since(time.Unix(turn.StartedAt, 0)) > streamStartGrace
}

// closingFrame is the ending a reader gets when the stream could not supply
// one. It names the turn so the client refetches the thread rather than
// trusting whatever half a reply it has on screen.
func closingFrame(turn *conversation.AssistantTurn) serviceports.TurnStreamFrame {
	if turn.Status == conversation.AssistantTurnStatusFailed && turn.ErrorMessage != "" {
		return errorFrame("This reply did not finish. What ran has been kept in the conversation.")
	}

	return frameOf(serviceports.AssistantEventDone, map[string]any{
		"turnId":   turn.ID.String(),
		"threadId": turn.ThreadID.String(),
		"status":   string(turn.Status),
		// replay says the ending was reconstructed from the turn's record
		// rather than forwarded from the turn itself, so a client knows to go
		// and read the conversation instead of trusting what it has.
		"replay": true,
	})
}

func errorFrame(message string) serviceports.TurnStreamFrame {
	return frameOf(serviceports.AssistantEventError, map[string]any{"message": message})
}
