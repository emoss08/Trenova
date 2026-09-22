package assistantturnservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger *zap.Logger
	Turns  repositories.AssistantTurnRepository
	Stream serviceports.TurnStreamPublisher
	Reader serviceports.TurnStreamReader
}

type Service struct {
	l      *zap.Logger
	turns  repositories.AssistantTurnRepository
	stream serviceports.TurnStreamPublisher
	reader serviceports.TurnStreamReader
}

func New(p Params) *Service {
	return &Service{
		l:      p.Logger.Named("service.assistantturn"),
		turns:  p.Turns,
		stream: p.Stream,
		reader: p.Reader,
	}
}

// StartRequest opens a turn on a conversation.
type StartRequest struct {
	ThreadID   pulid.ID
	UserID     pulid.ID
	TenantInfo pagination.TenantInfo
}

// Start records that a conversation is about to produce a reply.
//
// A conversation already producing one is refused rather than raced: two turns
// appending to the same thread interleave their message sequence numbers, and
// the transcript stops describing anything that happened.
func (s *Service) Start(
	ctx context.Context,
	req StartRequest,
) (*conversation.AssistantTurn, error) {
	turn, err := s.turns.Start(ctx, &conversation.AssistantTurn{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		ThreadID:       req.ThreadID,
		UserID:         req.UserID,
		Status:         conversation.AssistantTurnStatusRunning,
	})
	if err != nil {
		if errors.Is(err, repositories.ErrTurnAlreadyRunning) {
			return nil, errortypes.NewBusinessError(
				"This conversation is already working on a reply. Wait for it to finish, or stop it first.",
			)
		}

		return nil, err
	}

	return turn, nil
}

// Active is the turn a conversation is still producing, or nil.
func (s *Service) Active(
	ctx context.Context,
	req repositories.ActiveAssistantTurnRequest,
) (*conversation.AssistantTurn, error) {
	return s.turns.Active(ctx, req)
}

// Get reads one turn, scoped to whoever asked for it.
func (s *Service) Get(
	ctx context.Context,
	req repositories.GetAssistantTurnRequest,
) (*conversation.AssistantTurn, error) {
	return s.turns.GetByID(ctx, req)
}

// Observe wraps an emitter so everything it carries is also published to the
// turn's stream, where a second reader — or the same one after a reconnect —
// can find it.
//
// The returned close must be called exactly once, with the event that ended
// the turn. A stream that ends without one is how a relay learns its writer
// died, so forgetting it makes a finished turn look like a crashed one.
func (s *Service) Observe(
	ctx context.Context,
	turn *conversation.AssistantTurn,
	emit serviceports.AssistantStreamEmitter,
) (serviceports.AssistantStreamEmitter, func(serviceports.StreamEvent)) {
	ref := serviceports.TurnStreamRef{
		TenantInfo: pagination.TenantInfo{
			OrgID: turn.OrganizationID,
			BuID:  turn.BusinessUnitID,
		},
		TurnID: turn.ID,
	}
	pub := newPublisher(s.stream, ref, s.l)

	// The publish rides a context cancellation cannot reach. The turn's own
	// context dies the moment the reader closes the tab, which is precisely
	// when the events are worth keeping: the turn runs on, and somebody may
	// come back for what it said.
	keep := context.WithoutCancel(ctx)

	observed := func(event serviceports.StreamEvent) {
		if emit != nil {
			emit(event)
		}
		pub.emit(keep, event)
	}

	return observed, func(final serviceports.StreamEvent) { pub.close(keep, final) }
}

// Complete closes the turn's record.
func (s *Service) Complete(
	ctx context.Context,
	turn *conversation.AssistantTurn,
	status conversation.AssistantTurnStatus,
	cause error,
) {
	message := ""
	if cause != nil {
		message = cause.Error()
	}

	err := s.turns.Complete(ctx, repositories.CompleteAssistantTurnRequest{
		ID:         turn.ID,
		TenantInfo: pagination.TenantInfo{OrgID: turn.OrganizationID, BuID: turn.BusinessUnitID},
		Status:     status,
		Error:      message,
	})
	if err != nil {
		// A turn left Running is one the relay will keep waiting on until its
		// stream expires. Worth a loud log; not worth failing a reply that
		// already arrived.
		s.l.Error("could not close a turn's record",
			zap.String("turn", turn.ID.String()),
			zap.String("status", string(status)),
			zap.Error(err),
		)
	}
}

// StatusFor maps how a turn ended onto how it is recorded.
func StatusFor(refused bool, cause error) conversation.AssistantTurnStatus {
	switch {
	case cause != nil && errors.Is(cause, context.Canceled):
		return conversation.AssistantTurnStatusStopped
	case cause != nil:
		return conversation.AssistantTurnStatusFailed
	case refused:
		return conversation.AssistantTurnStatusRefused
	default:
		return conversation.AssistantTurnStatusCompleted
	}
}

func tenantOf(turn *conversation.AssistantTurn) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: turn.OrganizationID, BuID: turn.BusinessUnitID}
}

func streamRef(turn *conversation.AssistantTurn) serviceports.TurnStreamRef {
	return serviceports.TurnStreamRef{TenantInfo: tenantOf(turn), TurnID: turn.ID}
}

var errRelayStopped = fmt.Errorf("relay stopped")
