package assistantturnservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
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

	Logger     *zap.Logger
	Turns      repositories.AssistantTurnRepository
	Stream     serviceports.TurnStreamPublisher
	Reader     serviceports.TurnStreamReader
	Metrics    *metrics.Registry                  `optional:"true"`
	Trajectory serviceports.AgentRunEventRecorder `optional:"true"`
}

type Service struct {
	l          *zap.Logger
	turns      repositories.AssistantTurnRepository
	stream     serviceports.TurnStreamPublisher
	reader     serviceports.TurnStreamReader
	metrics    *metrics.Assistant
	trajectory serviceports.AgentRunEventRecorder
}

func New(p Params) *Service {
	return &Service{
		l:          p.Logger.Named("service.assistantturn"),
		turns:      p.Turns,
		stream:     p.Stream,
		reader:     p.Reader,
		metrics:    assistantMetrics(p.Metrics),
		trajectory: p.Trajectory,
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
	pub := newPublisher(s.stream, ref, s.l, s.metrics)

	// The same events, kept. The publisher above feeds a screen and its stream
	// is gone a quarter of an hour after the reply ends; this feeds the record
	// that answers what the assistant did months later.
	trajectory := s.trajectoryFor(ctx, ref.TenantInfo, turn)

	// The publish rides a context cancellation cannot reach. The turn's own
	// context dies the moment the reader closes the tab, which is precisely
	// when the events are worth keeping: the turn runs on, and somebody may
	// come back for what it said.
	keep := context.WithoutCancel(ctx)

	// When the turn first said anything. It is measured here rather than from
	// the turn's record because the record knows when the turn started and
	// how it ended, not when the person stopped looking at nothing — which is
	// the figure a durable hop could plausibly have made worse, and therefore
	// the one worth watching.
	began := time.Now()
	var spoke time.Time

	observed := func(event serviceports.StreamEvent) {
		if spoke.IsZero() {
			spoke = time.Now()
			s.metrics.RecordFirstEvent(transportOf(turn), spoke.Sub(began).Seconds())
		}
		if emit != nil {
			emit(event)
		}
		pub.emit(keep, event)
		serviceports.RecordTrajectory(trajectory, keep, event)
	}

	return observed, func(final serviceports.StreamEvent) {
		pub.close(keep, final)
		serviceports.RecordTrajectory(trajectory, keep, final)
		serviceports.FlushTrajectory(trajectory, keep)
	}
}

// trajectoryFor opens the durable writer for a turn, when there is one to open.
//
// The recorder is optional: an installation without it still answers questions,
// it just does not keep an account of how. Every call through the returned
// writer is nil-safe, so nothing downstream has to know which case it is in.
func (s *Service) trajectoryFor(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	turn *conversation.AssistantTurn,
) serviceports.AgentRunEventWriter {
	if s.trajectory == nil {
		return nil
	}

	return s.trajectory.Recorder(ctx, tenantInfo, serviceports.RunStepOwner{
		Kind: serviceports.RunStepOwnerAssistantTurn,
		ID:   turn.ID,
	})
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

	s.metrics.RecordTurn(
		transportOf(turn),
		string(status),
		elapsedSince(turn.StartedAt),
		// The first event is not known here. A turn's record carries when it
		// started and how it ended, not when it first spoke, so that figure
		// is observed by whoever held the stream.
		-1,
	)
}

// transportOf says where a turn ran, which is what makes the durable path
// comparable against the one it replaces. A turn with no execution behind it
// ran in the request that asked for it.
func transportOf(turn *conversation.AssistantTurn) string {
	if turn.WorkflowID == "" {
		return metrics.TransportInProcess
	}

	return metrics.TransportDurable
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
	return serviceports.TurnStreamRef{TenantInfo: tenantOf(turn), TurnID: turn.ID}
}

var errRelayStopped = fmt.Errorf("relay stopped")

// StartDurable records a turn and hands it to a worker.
//
// The reply is produced somewhere the request cannot reach, which is the whole
// point: an API restart no longer ends every conversation in flight, and the
// reader follows the turn's stream rather than holding a connection open for
// the length of an answer.
func (s *Service) StartDurable(
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

	return turn, nil
}

// Stop ends a turn somebody is no longer waiting for.
//
// Stopping used to be a property of the connection: aborting the request
// cancelled the context the loop ran on, and that was what stopped the model.
// With the work on a worker, closing a reader stops nothing — the turn runs
// on, and bills for it. So stopping is now something asked for explicitly.
func (s *Service) Stop(
	ctx context.Context,
	turn *conversation.AssistantTurn,
	cancel func(workflowID string) error,
) error {
	if turn.Status.Terminal() {
		// Already over. Saying so beats reporting a failure for something the
		// person got what they wanted from.
		s.metrics.RecordTurnStopped(transportOf(turn), "already_ended")

		return nil
	}

	if turn.WorkflowID == "" {
		// A turn still running in the request that asked for it. Its reader
		// aborting is what stops it, exactly as before, and there is no
		// execution to cancel.
		s.metrics.RecordTurnStopped(metrics.TransportInProcess, "no_execution")

		return nil
	}

	if err := cancel(turn.WorkflowID); err != nil {
		s.metrics.RecordTurnStopped(metrics.TransportDurable, "error")

		return fmt.Errorf("stop this reply: %w", err)
	}

	s.metrics.RecordTurnStopped(metrics.TransportDurable, "cancelled")

	return nil
}
