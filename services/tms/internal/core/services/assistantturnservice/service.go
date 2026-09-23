package assistantturnservice

import (
	"context"
	"errors"
	"fmt"
	"sync"
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
	running    *runningTurns
}

func New(p Params) *Service {
	return &Service{
		l:          p.Logger.Named("service.assistantturn"),
		turns:      p.Turns,
		stream:     p.Stream,
		reader:     p.Reader,
		metrics:    assistantMetrics(p.Metrics),
		trajectory: p.Trajectory,
		running:    newRunningTurns(),
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
	turn, err := s.insert(ctx, req)
	if errors.Is(err, repositories.ErrTurnAlreadyRunning) &&
		s.reapStale(ctx, req.TenantInfo, req.ThreadID) > 0 {
		// The turn holding the conversation died with its process. It is
		// closed now, so the question that found it in the way is asked once
		// more rather than refused for a reply that was never coming.
		turn, err = s.insert(ctx, req)
	}
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

func (s *Service) insert(ctx context.Context, req StartRequest) (*conversation.AssistantTurn, error) {
	return s.turns.Start(ctx, &conversation.AssistantTurn{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		ThreadID:       req.ThreadID,
		UserID:         req.UserID,
		Origin:         req.Origin,
		Input:          req.Input,
		Status:         conversation.AssistantTurnStatusRunning,
	})
}

// Active is the turn a conversation is still producing, or nil.
//
// A turn whose process died is closed first, so a reader rejoining the
// conversation is not handed a reply that will never arrive.
func (s *Service) Active(
	ctx context.Context,
	req repositories.ActiveAssistantTurnRequest,
) (*conversation.AssistantTurn, error) {
	s.reapStale(ctx, req.TenantInfo, req.ThreadID)

	return s.turns.Active(ctx, req)
}

const (
	// heartbeatInterval is how often a turn running in an API process records
	// that it is alive.
	heartbeatInterval = 15 * time.Second
	// staleAfter is how long an in-process turn may go without a heartbeat
	// before it is taken to have died with its process. It spans several
	// missed heartbeats, so a slow database write is not mistaken for a death.
	staleAfter = time.Minute
	// staleTurnError is what a turn closed that way records.
	staleTurnError = "The server stopped before this reply finished."
)

// reapStale closes the in-process turns on a conversation that stopped
// heartbeating, and reports how many it closed. A failure to check is logged
// and treated as nothing to close: the reader then sees the turn as running,
// which Stop still ends.
func (s *Service) reapStale(
	ctx context.Context,
	tenant pagination.TenantInfo,
	threadID pulid.ID,
) int {
	closed, err := s.turns.FailStale(ctx, repositories.FailStaleAssistantTurnsRequest{
		ThreadID:   threadID,
		TenantInfo: tenant,
		Before:     time.Now().Add(-staleAfter).Unix(),
		Error:      staleTurnError,
	})
	if err != nil {
		s.l.Warn("could not close turns a stopped process left running",
			zap.String("thread", threadID.String()),
			zap.Error(err),
		)

		return 0
	}
	if closed > 0 {
		s.l.Info("closed turns a stopped process left running",
			zap.String("thread", threadID.String()),
			zap.Int("turns", closed),
		)
	}

	return closed
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
		return s.stopInProcess(ctx, turn)
	}

	if err := cancel(turn.WorkflowID); err != nil {
		s.metrics.RecordTurnStopped(metrics.TransportDurable, "error")

		return fmt.Errorf("stop this reply: %w", err)
	}

	s.metrics.RecordTurnStopped(metrics.TransportDurable, "cancelled")

	return nil
}

// stopInProcess ends a turn running inside an API process rather than on a
// worker. Aborting the reader used to be the only way to stop one, which
// left a turn nobody's request carried — a decision's follow-up, or one a
// reader rejoined after a reload — impossible to stop at all.
//
// The record is marked first, so every instance's Stoppable sees it; the
// turn is then cancelled directly when it happens to be running here.
func (s *Service) stopInProcess(ctx context.Context, turn *conversation.AssistantTurn) error {
	err := s.turns.Complete(ctx, repositories.CompleteAssistantTurnRequest{
		ID:         turn.ID,
		TenantInfo: tenantOf(turn),
		Status:     conversation.AssistantTurnStatusStopped,
	})
	if err != nil {
		s.metrics.RecordTurnStopped(metrics.TransportInProcess, "error")

		return fmt.Errorf("stop this reply: %w", err)
	}

	s.running.cancel(turn.ID)
	s.metrics.RecordTurnStopped(metrics.TransportInProcess, "cancelled")

	return nil
}

// stopPollInterval is how often a turn running in-process re-reads its record
// to learn it was stopped from another instance.
const stopPollInterval = 2 * time.Second

// Stoppable derives the context an in-process turn runs on, one that a Stop
// cancels wherever it was asked. A Stop reaching this instance cancels it at
// once; one reaching another instance is seen on the next read of the turn's
// record.
//
// The returned cancel must be called when the turn ends.
func (s *Service) Stoppable(
	ctx context.Context,
	turn *conversation.AssistantTurn,
) (context.Context, context.CancelFunc) {
	runCtx, cancel := context.WithCancel(ctx)
	release := s.running.add(turn.ID, cancel)

	// The poll reads a copy: stillRunning writes what it learns back onto
	// the turn it is given, and the caller still holds the original.
	watched := *turn
	go func() {
		ticker := time.NewTicker(stopPollInterval)
		defer ticker.Stop()
		beat := time.Now()

		for {
			select {
			case <-runCtx.Done():
				return
			case now := <-ticker.C:
				if !s.stillRunning(runCtx, &watched) {
					cancel()
					return
				}
				if now.Sub(beat) >= heartbeatInterval {
					beat = now
					s.heartbeat(runCtx, &watched)
				}
			}
		}
	}()

	return runCtx, func() {
		release()
		cancel()
	}
}

// heartbeat records that a turn running here is alive. A missed beat is only
// logged: the turn carries on, and it takes several before one is mistaken for
// a death.
func (s *Service) heartbeat(ctx context.Context, turn *conversation.AssistantTurn) {
	if err := s.turns.Heartbeat(ctx, turn.ID, tenantOf(turn)); err != nil &&
		!errors.Is(err, context.Canceled) {
		s.l.Warn("could not record that a turn is alive",
			zap.String("turn", turn.ID.String()),
			zap.Error(err),
		)
	}
}

// runningTurns holds how to cancel each turn running in this process.
type runningTurns struct {
	mu      sync.Mutex
	cancels map[pulid.ID]context.CancelFunc
}

func newRunningTurns() *runningTurns {
	return &runningTurns{cancels: make(map[pulid.ID]context.CancelFunc)}
}

func (r *runningTurns) add(id pulid.ID, cancel context.CancelFunc) func() {
	r.mu.Lock()
	r.cancels[id] = cancel
	r.mu.Unlock()

	return func() {
		r.mu.Lock()
		delete(r.cancels, id)
		r.mu.Unlock()
	}
}

func (r *runningTurns) cancel(id pulid.ID) {
	r.mu.Lock()
	cancel, ok := r.cancels[id]
	r.mu.Unlock()

	if ok {
		cancel()
	}
}

// Ending is the event that closes a turn's stream: the saved result, or a
// sentence saying why there is none. Every path that runs a turn ends it the
// same way, so a reader cannot tell a worker's reply from a request's.
func Ending(
	result *serviceports.SendMessageResult,
	cause error,
) serviceports.StreamEvent {
	if cause != nil {
		message := "The assistant could not finish this reply. Try again in a moment."
		if errors.Is(cause, context.Canceled) {
			message = "Stopped. What was said so far has been kept in the conversation."
		}

		return serviceports.StreamEvent{
			Event: serviceports.AssistantEventError,
			Data:  map[string]any{"message": message},
		}
	}

	return serviceports.StreamEvent{Event: serviceports.AssistantEventDone, Data: result}
}
