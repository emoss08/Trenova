package assistantturnservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aimetrics"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantchapterports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantturnports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/app/planservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/quotaservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
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
	Turns     assistantturnports.AssistantTurnRepository
	Reader    assistantstreamports.TurnStreamReader
	Canceller assistantturnports.AssistantTurnCanceller
	Realtime  realtimeports.RealtimeService
	Metrics   *metrics.Registry                      `optional:"true"`
	Quota     platformplanports.QuotaGuard           `optional:"true"`
	Plans     platformplanports.PlanService          `optional:"true"`
	Chapters  assistantchapterports.ChapterScheduler `optional:"true"`
}

type Service struct {
	l         *zap.Logger
	turns     assistantturnports.AssistantTurnRepository
	reader    assistantstreamports.TurnStreamReader
	canceller assistantturnports.AssistantTurnCanceller
	realtime  realtimeports.RealtimeService
	metrics   *aimetrics.Assistant
	quota     platformplanports.QuotaGuard
	plans     platformplanports.PlanService
	chapters  assistantchapterports.ChapterScheduler
}

var _ assistantturnports.AssistantTurnStopper = (*Service)(nil)

func New(p Params) *Service {
	return &Service{
		l:         p.Logger.Named("service.assistantturn"),
		turns:     p.Turns,
		reader:    p.Reader,
		canceller: p.Canceller,
		realtime:  p.Realtime,
		metrics:   metrics.AssistantFrom(p.Metrics),
		quota:     p.Quota,
		plans:     p.Plans,
		chapters:  p.Chapters,
	}
}

// StartRequest opens a turn on a conversation.
type StartRequest struct {
	ThreadID   pulid.ID
	UserID     pulid.ID
	TenantInfo pagination.TenantInfo
	// Origin and Input say what the turn answers. An empty origin is a
	// person's question.
	Origin conversationturn.AssistantTurnOrigin
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
) (*conversationturn.AssistantTurn, error) {
	if err := s.assertPlanAllows(ctx, req); err != nil {
		return nil, err
	}

	id := pulid.MustNew("atrn_")
	anchor := aitrace.AnchorFor(aitrace.AnchorAssistantTurn, id.String())
	turn, err := s.turns.Start(ctx, &conversationturn.AssistantTurn{
		ID:             id,
		TraceID:        anchor.TraceID.String(),
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		ThreadID:       req.ThreadID,
		UserID:         req.UserID,
		Origin:         req.Origin,
		Input:          req.Input,
		Status:         conversation.AssistantTurnStatusRunning,
	})
	if err != nil {
		if errors.Is(err, assistantturnports.ErrTurnAlreadyRunning) {
			return nil, errortypes.NewBusinessError(
				"This conversation is already working on a reply. Wait for it to finish, or stop it first.",
			)
		}

		return nil, err
	}

	return turn, nil
}

func (s *Service) assertPlanAllows(ctx context.Context, req StartRequest) error {
	switch req.Origin {
	case "", conversationturn.AssistantTurnOriginPerson:
		return quotaservice.Preflight(ctx, s.quota, &platformplanports.QuotaRequest{
			TenantInfo: req.TenantInfo,
			Meter:      platformcatalog.MeterAIAssistantMessages,
			Quantity:   1,
		})
	case conversationturn.AssistantTurnOriginScheduled:
		return planservice.RequireCapability(
			ctx,
			s.plans,
			req.TenantInfo,
			platformplan.CapabilityAgentAutomation,
		)
	default:
		return nil
	}
}

// Active is the turn a conversation is still producing, or nil.
func (s *Service) Active(
	ctx context.Context,
	req assistantturnports.ActiveAssistantTurnRequest,
) (*conversationturn.AssistantTurn, error) {
	return s.turns.Active(ctx, req)
}

// ListLive is every reply one person still has in progress, across all of
// their conversations, so a reply started in one tab can be found from any
// other.
func (s *Service) ListLive(
	ctx context.Context,
	req assistantturnports.ListLiveAssistantTurnsRequest,
) ([]*assistantturnports.LiveAssistantTurn, error) {
	return s.turns.ListLive(ctx, req)
}

// Get reads one turn, scoped to whoever asked for it.
func (s *Service) Get(
	ctx context.Context,
	req assistantturnports.GetAssistantTurnRequest,
) (*conversationturn.AssistantTurn, error) {
	return s.turns.GetByID(ctx, req)
}

// Close closes the turn's record and tells the person's other tabs the reply
// is over. It reports a failure to close, for a caller that retries.
func (s *Service) Close(
	ctx context.Context,
	turn *conversationturn.AssistantTurn,
	status conversation.AssistantTurnStatus,
	message string,
) error {
	err := s.turns.Complete(ctx, assistantturnports.CompleteAssistantTurnRequest{
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
	turn *conversationturn.AssistantTurn,
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

func tenantOf(turn *conversationturn.AssistantTurn) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: turn.OrganizationID, BuID: turn.BusinessUnitID}
}

func streamRef(turn *conversationturn.AssistantTurn) assistantstreamports.TurnStreamRef {
	return assistantstreamports.TurnStreamRef{
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
	start func(turn *conversationturn.AssistantTurn) (string, error),
) (*conversationturn.AssistantTurn, error) {
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
	s.cueChapter(ctx, turn)

	return turn, nil
}

func (s *Service) cueChapter(ctx context.Context, turn *conversationturn.AssistantTurn) {
	if s.chapters == nil {
		return
	}
	s.chapters.Cue(ctx, &assistantchapterports.ChapterTurnRequest{
		TenantInfo: tenantOf(turn),
		ThreadID:   turn.ThreadID,
		UserID:     turn.UserID,
		TurnID:     turn.ID,
		Phase:      assistantchapterports.ChapterPhaseOpened,
	})
}

// ErrNoExecution is what a cancel reports when no execution carries the
// turn, which is how Stop tells a turn it must close itself from one it asked
// to stop.
var ErrNoExecution = assistantturnports.ErrNoTurnExecution

// Stop ends a turn somebody is no longer waiting for.
//
// Closing a reader stops nothing: the turn runs on a worker, and bills for it.
// So stopping is asked for explicitly, and cancels the turn's workflow.
func (s *Service) Stop(ctx context.Context, turn *conversationturn.AssistantTurn) error {
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
	req assistantturnports.StopUserTurnsRequest,
) error {
	live, err := s.turns.ListLive(ctx, assistantturnports.ListLiveAssistantTurnsRequest{
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
