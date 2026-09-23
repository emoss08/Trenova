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
// The events live in the turn's workflow, so there is no stream to expire and
// no way for a reader to arrive before it exists. There are two ways a relay
// ends, and a reader is told the truth about both:
//
// The turn said how it ended. Its last frame is forwarded and the relay stops.
//
// The turn's workflow closed before the reader caught up, which happens to a
// reader who comes back after the turn finished, or one who fell far enough
// behind that the workflow stopped waiting for it. The turn's record says how
// it ended, and the reader is sent to read the conversation, which has it all.
func (s *Service) Relay(
	ctx context.Context,
	req RelayRequest,
	onFrame serviceports.TurnFrameFunc,
) error {
	turn := req.Turn

	// A turn that was already over before anybody attached needs no stream at
	// all: the answer is written down, and saying so immediately beats
	// reading a workflow that has closed.
	if turn.Status.Terminal() || turn.WorkflowID == "" {
		s.metrics.RecordStreamAttach("already_ended")

		return onFrame(closingFrame(s.current(ctx, turn)))
	}

	fresh := req.Cursor == ""
	if fresh {
		s.metrics.RecordStreamAttach("live")
	} else {
		// A cursor means somebody came back to a reply they had already
		// started watching, which is the recovery this work exists to make
		// possible and therefore the number worth counting.
		s.metrics.RecordStreamAttach("resumed")
	}

	attached := time.Now()
	spoke := false
	sawTerminal := false
	err := s.reader.Read(ctx, serviceports.ReadTurnStreamRequest{
		Ref:    streamRef(turn),
		Cursor: req.Cursor,
		OnFrame: func(frame serviceports.TurnStreamFrame) error {
			if fresh && !spoke {
				spoke = true
				s.metrics.RecordFirstEvent(time.Since(attached).Seconds())
			}
			if frame.Terminal() {
				sawTerminal = true
			}

			return onFrame(frame)
		},
	})
	switch {
	case err == nil:
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		// The reader left. Nothing is wrong with the turn.
		return nil
	case sawTerminal:
		return nil
	default:
		return err
	}

	if sawTerminal {
		return nil
	}

	// The workflow closed without this reader seeing how the turn ended.
	s.metrics.RecordStreamAttach("closed_before_end")

	return onFrame(closingFrame(s.current(ctx, turn)))
}

// current re-reads a turn's record, for how it ended. A read that fails keeps
// what the caller already had: the reader is sent to the conversation either
// way, and a failed read is no reason to tell them anything worse.
func (s *Service) current(
	ctx context.Context,
	turn *conversation.AssistantTurn,
) *conversation.AssistantTurn {
	current, err := s.turns.GetByID(ctx, repositories.GetAssistantTurnRequest{
		ID:         turn.ID,
		TenantInfo: tenantOf(turn),
	})
	if err != nil {
		s.l.Warn("could not read how a turn ended",
			zap.String("turn", turn.ID.String()),
			zap.Error(err),
		)

		return turn
	}

	return current
}

// ClosingEvent is the ending a reader gets when the turn's own ending cannot
// be given to them: it names the turn, so the client refetches the thread
// rather than trusting whatever half a reply it has on screen.
func ClosingEvent(turn *conversation.AssistantTurn) serviceports.StreamEvent {
	if turn.Status == conversation.AssistantTurnStatusFailed && turn.ErrorMessage != "" {
		return serviceports.StreamEvent{
			Event: serviceports.AssistantEventError,
			Data: map[string]any{
				"message": "This reply did not finish. What ran has been kept in the conversation.",
			},
		}
	}

	return serviceports.StreamEvent{
		Event: serviceports.AssistantEventDone,
		Data: map[string]any{
			"turnId":   turn.ID.String(),
			"threadId": turn.ThreadID.String(),
			"status":   string(turn.Status),
			// replay says the ending was reconstructed from the turn's record
			// rather than forwarded from the turn itself, so a client knows to
			// go and read the conversation instead of trusting what it has.
			"replay": true,
		},
	}
}

func closingFrame(turn *conversation.AssistantTurn) serviceports.TurnStreamFrame {
	event := ClosingEvent(turn)

	return frameOf(event.Event, event.Data)
}
