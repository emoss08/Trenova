package assistantturnservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger    *zap.Logger
	Turns     repositories.AssistantTurnRepository
	Reader    serviceports.TurnStreamReader
	Canceller serviceports.AssistantTurnCanceller
	Realtime  serviceports.RealtimeService
	Metrics   *metrics.Registry `optional:"true"`
}

type Service struct {
	l         *zap.Logger
	turns     repositories.AssistantTurnRepository
	reader    serviceports.TurnStreamReader
	canceller serviceports.AssistantTurnCanceller
	realtime  serviceports.RealtimeService
	metrics   *metrics.Assistant
}

var _ serviceports.AssistantTurnStopper = (*Service)(nil)

func New(p Params) *Service {
	return &Service{
		l:         p.Logger.Named("service.assistantturn"),
		turns:     p.Turns,
		reader:    p.Reader,
		canceller: p.Canceller,
		realtime:  p.Realtime,
		metrics:   assistantMetrics(p.Metrics),
	}
}

// assistantMetrics tolerates a service built without a registry, which a test
// does and an install with metrics switched off does too. Every method on the
// returned value is safe on a disabled collector.
func assistantMetrics(registry *metrics.Registry) *metrics.Assistant {
	if registry == nil {
		return metrics.NewAssistant(nil, zap.NewNop(), false)
	}

	return registry.Assistant
}

// StartRequest opens a turn on a conversation.
type StartRequest struct {
	ThreadID   pulid.ID
	UserID     pulid.ID
	TenantInfo pagination.TenantInfo
	// Origin and Input say what the turn answers. An empty origin is a
	// person's question.
	Origin conversation.AssistantTurnOrigin
	Input  string
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
	id := pulid.MustNew("atrn_")
	turn, err := s.turns.Start(ctx, &conversation.AssistantTurn{
		ID:             id,
		TraceID:        aitrace.AnchorFor(aitrace.AnchorAssistantTurn, id.String()).TraceID.String(),
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		ThreadID:       req.ThreadID,
		UserID:         req.UserID,
		Origin:         req.Origin,
		Input:          req.Input,
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

// ListLive is every reply one person still has in progress, across all of
// their conversations, so a reply started in one tab can be found from any
// other.
func (s *Service) ListLive(
	ctx context.Context,
	req repositories.ListLiveAssistantTurnsRequest,
) ([]*repositories.LiveAssistantTurn, error) {
	return s.turns.ListLive(ctx, req)
}

// Get reads one turn, scoped to whoever asked for it.
func (s *Service) Get(
	ctx context.Context,
	req repositories.GetAssistantTurnRequest,
) (*conversation.AssistantTurn, error) {
	return s.turns.GetByID(ctx, req)
}

// Close closes the turn's record and tells the person's other tabs the reply
// is over. It reports a failure to close, for a caller that retries.
func (s *Service) Close(
	ctx context.Context,
	turn *conversation.AssistantTurn,
	status conversation.AssistantTurnStatus,
	message string,
) error {
	err := s.turns.Complete(ctx, repositories.CompleteAssistantTurnRequest{
		ID:         turn.ID,
		TenantInfo: tenantOf(turn),
		Status:     status,
		Error:      message,
	})
	if err != nil {
		return err
	}

	s.announce(ctx, turn, turnActionFinished, status)

	return nil
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

	if err := s.Close(ctx, turn, status, message); err != nil {
		// A turn left Running is one the relay will keep waiting on until its
		// stream expires. Worth a loud log; not worth failing a reply that
		// already arrived.
		s.l.Error("could not close a turn's record",
			zap.String("turn", turn.ID.String()),
			zap.String("status", string(status)),
			zap.Error(err),
		)
	}

	s.metrics.RecordTurn(string(status), elapsedSince(turn.StartedAt))
}

func elapsedSince(startedAt int64) float64 {
	if startedAt <= 0 {
		return 0
	}

	return float64(timeutils.NowUnix() - startedAt)
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
	return serviceports.TurnStreamRef{
		TenantInfo: tenantOf(turn),
		TurnID:     turn.ID,
		WorkflowID: turn.ExecutionID(),
	}
}

// StartTurn records a turn and hands it to a worker.
//
// The reply is produced somewhere the request cannot reach, which is the whole
// point: an API restart no longer ends every conversation in flight, and the
// reader follows the turn's stream rather than holding a connection open for
// the length of an answer.
func (s *Service) StartTurn(
	ctx context.Context,
	req StartRequest,
	start func(turn *conversation.AssistantTurn) (string, error),
) (*conversation.AssistantTurn, error) {
	turn, err := s.Start(ctx, req)
	if err != nil {
		return nil, err
	}

	workflowID, err := start(turn)
	if err != nil {
		// The record must not outlive the failure to start. A turn left
		// Running with nothing running it would hold the conversation's one
		// live slot until it expired, and refuse every later question.
		s.Complete(ctx, turn, conversation.AssistantTurnStatusFailed, err)

		return nil, err
	}

	if mErr := s.turns.MarkWorkflow(ctx, turn.ID, req.TenantInfo, workflowID); mErr != nil {
		s.l.Error("could not record the workflow carrying a turn",
			zap.String("turn", turn.ID.String()),
			zap.Error(mErr),
		)
	}
	turn.WorkflowID = workflowID
	s.announce(ctx, turn, turnActionStarted, turn.Status)

	return turn, nil
}

// ErrNoExecution is what a cancel reports when no execution carries the
// turn, which is how Stop tells a turn it must close itself from one it asked
// to stop.
var ErrNoExecution = serviceports.ErrNoTurnExecution

// Stop ends a turn somebody is no longer waiting for.
//
// Closing a reader stops nothing: the turn runs on a worker, and bills for it.
// So stopping is asked for explicitly, and cancels the turn's workflow.
func (s *Service) Stop(ctx context.Context, turn *conversation.AssistantTurn) error {
	if turn.Status.Terminal() {
		// Already over. Saying so beats reporting a failure for something the
		// person got what they wanted from.
		s.metrics.RecordTurnStopped("already_ended")

		return nil
	}

	err := s.canceller.CancelTurn(ctx, turn.ExecutionID())
	switch {
	case err == nil:
	case errors.Is(err, ErrNoExecution):
		// Nothing carries the turn: it was never handed to a worker, or its
		// execution ended without closing the record. Closing it here is what
		// frees the conversation's one live slot for the next question.
		if cErr := s.Close(
			ctx, turn, conversation.AssistantTurnStatusStopped, "",
		); cErr != nil {
			s.metrics.RecordTurnStopped("error")

			return fmt.Errorf("stop this reply: %w", cErr)
		}
		s.metrics.RecordTurnStopped("no_execution")

		return nil
	default:
		s.metrics.RecordTurnStopped("error")

		return fmt.Errorf("stop this reply: %w", err)
	}

	s.metrics.RecordTurnStopped("cancelled")

	return nil
}

// StopAllForUser ends every reply a person has in progress in one tenant.
//
// A turn does not record the session that asked for it, so signing out of one
// browser stops the replies started from every other one too. Each turn is
// stopped on its own: one that cannot be stopped does not keep the rest
// running, and every failure is reported together.
func (s *Service) StopAllForUser(
	ctx context.Context,
	req serviceports.StopUserTurnsRequest,
) error {
	live, err := s.turns.ListLive(ctx, repositories.ListLiveAssistantTurnsRequest{
		UserID:     req.UserID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return fmt.Errorf("read the replies still in progress: %w", err)
	}

	var errs []error
	for _, turn := range live {
		if sErr := s.Stop(ctx, &turn.AssistantTurn); sErr != nil {
			errs = append(errs, fmt.Errorf("turn %s: %w", turn.ID, sErr))
		}
	}

	return errors.Join(errs...)
}
