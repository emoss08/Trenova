package assistantjobs

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/assistantservice"
	"github.com/emoss08/trenova/internal/core/services/assistantturnservice"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/contrib/workflowstreams"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Error types the workflow must not retry. Each names an outcome a second
// attempt would reach identically, and the issue this work comes from is
// explicit that validation and authorization outcomes are surfaced rather
// than retried.
const (
	errTypeNoActor = "AssistantTurnHasNoActor"
	// errTypeRejected is a question turned away before it was answered, for
	// a reason written for the person who asked: an empty message, a
	// disabled agent, a spent budget, a conversation that is full.
	errTypeRejected = "AssistantTurnRejected"
)

// The messages a reader is shown for a turn that did not finish. They say
// what happened to the conversation, because that is what the reader will
// look at next.
const (
	stoppedMessage = "Stopped. What was said so far has been kept in the conversation."
	failedMessage  = "The assistant could not finish this reply. Try again in a moment."
)

// turnFinishKey names saving the turn in the ledger. Saving appends the turn
// to the conversation, and a retry of a save that landed before its worker was
// lost would append the same turn twice. Each attempt claims its own key under
// this prefix, after checking that no earlier attempt saved the turn or began
// to: an attempt that failed cleanly settles its key as failed, which is the
// one outcome a later attempt may save after.
const turnFinishKey = "assistant-turn-finish"

// streamFlushTimeout bounds the last flush of what an activity published. It
// rides a context cancellation cannot reach, so it needs a bound of its own.
const streamFlushTimeout = 10 * time.Second

type ActivitiesParams struct {
	fx.In

	Logger        *zap.Logger
	Assistant     *assistantservice.Service
	Turns         *assistantturnservice.Service
	TurnRepo      repositories.AssistantTurnRepository
	Steps         serviceports.RunStepLedger
	Notifications *notificationservice.Service
	Trajectory    serviceports.AgentRunEventRecorder   `optional:"true"`
	FollowUps     serviceports.DecisionFollowUpResumer `optional:"true"`
}

// Activities are a turn's first and last steps. Everything between them is
// agentflow's.
type Activities struct {
	logger        *zap.Logger
	assistant     *assistantservice.Service
	turns         *assistantturnservice.Service
	turnRepo      repositories.AssistantTurnRepository
	steps         serviceports.RunStepLedger
	trajectory    serviceports.AgentRunEventRecorder
	threads       replyThreads
	notifications replyNotifications
	followUps     serviceports.DecisionFollowUpResumer
}

func NewActivities(p ActivitiesParams) *Activities {
	a := &Activities{
		logger:     p.Logger.Named("job.assistant-turn"),
		assistant:  p.Assistant,
		turns:      p.Turns,
		turnRepo:   p.TurnRepo,
		steps:      p.Steps,
		trajectory: p.Trajectory,
		followUps:  p.FollowUps,
	}
	// Assigned only when present: a nil pointer held by an interface is not
	// a nil interface, and the notice would dereference it.
	if p.Assistant != nil {
		a.threads = p.Assistant
	}
	if p.Notifications != nil {
		a.notifications = p.Notifications
	}

	return a
}

// PrepareTurnActivity checks the question and makes it ready to answer.
//
// The name carries its package because Temporal's activity registry is keyed
// by method name across a whole task queue, and a collision is a panic at
// startup rather than a compile error.
func (a *Activities) PrepareTurnActivity(
	ctx context.Context,
	payload *AssistantTurnPayload,
) (*assistantservice.TurnPlan, error) {
	if payload.Actor.UserID.IsNil() {
		// A turn runs as the person who asked. A worker that carried on
		// without one would be running as something nobody granted, which is
		// precisely what a background worker must never do.
		return nil, temporal.NewNonRetryableApplicationError(
			"this turn arrived without the person who asked for it", errTypeNoActor, nil,
		)
	}

	plan, err := a.assistant.PrepareTurn(ctx, payload.sendRequest(), &payload.Actor)
	if err != nil {
		if rejected(err) {
			return nil, temporal.NewNonRetryableApplicationError(err.Error(), errTypeRejected, err)
		}

		return nil, fmt.Errorf("prepare this turn: %w", err)
	}

	return plan, nil
}

// rejected reports an error that is an answer rather than a fault: something
// the person asked for that cannot be done, which asking again will not
// change.
func rejected(err error) bool {
	return errortypes.IsBusinessError(err) ||
		errortypes.IsMultiError(err) ||
		errortypes.IsNotFoundError(err) ||
		errortypes.IsAuthorizationError(err)
}

// FinishTurnActivity saves what the turn came to, closes its record, and says
// how it ended.
//
// It runs however the turn ended: answered, refused, failed partway or
// stopped. Whatever ran is kept in the conversation.
func (a *Activities) FinishTurnActivity(
	ctx context.Context,
	in *FinishTurnInput,
) (*TurnEnding, error) {
	payload := in.Payload
	tenant := payload.tenantInfo()

	turn, err := a.turnRepo.GetByID(ctx, repositories.GetAssistantTurnRequest{
		ID:         payload.TurnID,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, fmt.Errorf("read this turn's record: %w", err)
	}

	fresh, err := a.claimSave(ctx, payload)
	if err != nil {
		return nil, err
	}
	if !fresh {
		// An earlier attempt saved it, or began to and was lost. Either way
		// the conversation holds what it holds, and saving again would add
		// the same turn twice. The reader is sent to read it.
		a.logger.Info("this turn was already saved on an earlier attempt",
			zap.String("turn", payload.TurnID.String()),
		)

		ending := a.alreadySaved(ctx, turn)
		a.resumeFollowUps(ctx, in)

		return ending, nil
	}

	ending, cause, err := a.finish(ctx, in)
	if err != nil {
		// Nothing was appended: the save fails before the append or in it.
		// Settling the claim as failed is what lets the retry save.
		a.settle(ctx, payload, serviceports.RunStepFailed, err.Error())

		return nil, err
	}
	a.settle(ctx, payload, serviceports.RunStepCompleted, ending.Result.Status)
	a.turns.Complete(ctx, turn, conversation.AssistantTurnStatus(ending.Result.Status), cause)
	a.recordTrajectory(ctx, tenant, payload, in.Events, ending.Event)
	a.resumeFollowUps(ctx, in)

	return ending, nil
}

// resumeFollowUps starts the report of a decision made while this turn held
// the conversation, now that the conversation is free. The turn read its
// proposals before that decision, so it could not report it, and the
// decision's own follow-up found the conversation busy.
//
// A turn the person started resumes however it ended, since it held the
// conversation either way. A follow-up that could not be prepared does not:
// it saved no note, so resuming from it would start the same failing
// follow-up again, and the conversation's next turn resumes instead.
func (a *Activities) resumeFollowUps(ctx context.Context, in *FinishTurnInput) {
	if a.followUps == nil || (in.Plan == nil && in.Payload.Request.reportsDecision()) {
		return
	}

	a.followUps.ResumeFollowUps(ctx, serviceports.ResumeFollowUpsRequest{
		TenantInfo: in.Payload.tenantInfo(),
		ThreadID:   in.Payload.ThreadID,
	})
}

// finish saves the turn and decides how it ended. cause is why the turn did
// not finish, for the turn's record; err is a failure to save, which fails the
// activity so it is tried again.
func (a *Activities) finish(
	ctx context.Context,
	in *FinishTurnInput,
) (ending *TurnEnding, cause, err error) {
	if in.Plan == nil {
		// Turned away, or stopped, before there was anything to save.
		if in.Failure != nil && in.Failure.Stopped {
			return failedEnding(conversation.AssistantTurnStatusStopped, stoppedMessage),
				in.Failure.Err(), nil
		}
		if in.Rejection != "" {
			return failedEnding(conversation.AssistantTurnStatusFailed, in.Rejection),
				errors.New(in.Rejection), nil
		}

		return failedEnding(conversation.AssistantTurnStatusFailed, failedMessage),
			in.Failure.Err(), nil
	}

	cause = in.Failure.Err()
	stream, emit := a.openStream(ctx)
	defer a.closeStream(ctx, stream)

	result, err := a.assistant.FinishTurn(ctx, &assistantservice.FinishTurnRequest{
		Plan:       in.Plan,
		Actor:      &in.Payload.Actor,
		TenantInfo: in.Payload.tenantInfo(),
		Run:        in.Run,
		Failure:    cause,
		Artifacts:  in.Artifacts,
	}, emit)
	if err != nil {
		return nil, nil, fmt.Errorf("save this turn: %w", err)
	}
	a.recordFingerprint(ctx, in)

	status := assistantturnservice.StatusFor(result.Refused, cause)
	switch status {
	case conversation.AssistantTurnStatusStopped:
		return failedEnding(status, stoppedMessage), cause, nil
	case conversation.AssistantTurnStatusFailed:
		return failedEnding(status, failedMessage), cause, nil
	default:
		return &TurnEnding{
			Result: AssistantTurnResult{
				Status:  string(status),
				Refused: result.Refused,
				Result:  result,
			},
			Event: temporaltype.StreamItem{Event: serviceports.AssistantEventDone, Data: result},
		}, nil, nil
	}
}

func (a *Activities) recordFingerprint(ctx context.Context, in *FinishTurnInput) {
	fingerprint := assistantservice.TurnFingerprint(in.Plan, in.Run)
	if fingerprint == nil || a.turnRepo == nil {
		return
	}

	if err := a.turnRepo.RecordFingerprint(ctx, repositories.RecordAssistantTurnFingerprintRequest{
		ID:          in.Payload.TurnID,
		TenantInfo:  in.Payload.tenantInfo(),
		Fingerprint: fingerprint,
	}); err != nil {
		a.logger.Warn("could not keep the agent a turn ran as",
			zap.String("turn", in.Payload.TurnID.String()),
			zap.Error(err),
		)
	}
}

func failedEnding(status conversation.AssistantTurnStatus, message string) *TurnEnding {
	return &TurnEnding{
		Result: AssistantTurnResult{Status: string(status), Message: message},
		Event: temporaltype.StreamItem{
			Event: serviceports.AssistantEventError,
			Data:  map[string]any{"message": message},
		},
	}
}

// alreadySaved is the ending for a turn an earlier attempt saved. It carries
// no result, only where to read one, which is what tells the client to fetch
// the conversation rather than trust what it has on screen.
func (a *Activities) alreadySaved(
	ctx context.Context,
	turn *conversation.AssistantTurn,
) *TurnEnding {
	current, err := a.turnRepo.GetByID(ctx, repositories.GetAssistantTurnRequest{
		ID:         turn.ID,
		TenantInfo: pagination.TenantInfo{OrgID: turn.OrganizationID, BuID: turn.BusinessUnitID},
	})
	if err == nil {
		turn = current
	}
	if !turn.Status.Terminal() {
		// The attempt that saved it was lost before closing the record. Left
		// open, the record would hold the conversation's one live slot for
		// ever; what the turn came to is in the conversation either way.
		a.turns.Complete(ctx, turn, conversation.AssistantTurnStatusCompleted, nil)
		turn.Status = conversation.AssistantTurnStatusCompleted
	}

	event := assistantturnservice.ClosingEvent(turn)

	return &TurnEnding{
		Result: AssistantTurnResult{Status: string(turn.Status)},
		Event:  temporaltype.StreamItem{Event: event.Event, Data: event.Data},
	}
}

// CloseTurnActivity closes a turn's record the save could not close. It is
// the last thing a turn does when saving it failed on every attempt, and it
// never touches the conversation, only the record, which is what frees the
// conversation for its next question.
func (a *Activities) CloseTurnActivity(
	ctx context.Context,
	payload *AssistantTurnPayload,
	message string,
) error {
	turn, err := a.turnRepo.GetByID(ctx, repositories.GetAssistantTurnRequest{
		ID:         payload.TurnID,
		TenantInfo: payload.tenantInfo(),
	})
	if err != nil {
		return fmt.Errorf("read this turn's record: %w", err)
	}
	if turn.Status.Terminal() {
		return nil
	}

	err = a.turns.Close(ctx, turn, conversation.AssistantTurnStatusFailed, message)
	if err != nil {
		return fmt.Errorf("close this turn's record: %w", err)
	}

	return nil
}

// claimSave reserves saving the turn for this attempt, and reports whether
// this attempt may save it.
func (a *Activities) claimSave(ctx context.Context, payload *AssistantTurnPayload) (bool, error) {
	tenant := payload.tenantInfo()
	recorded, err := a.steps.Loaded(ctx, tenant, serviceports.RunStepOwner{
		Kind: serviceports.RunStepOwnerAssistantTurn,
		ID:   payload.TurnID,
	})
	if err != nil {
		return false, fmt.Errorf("read what this turn already did: %w", err)
	}
	for i := range recorded {
		step := &recorded[i]
		if step.Kind == serviceports.RunStepCompletion &&
			strings.HasPrefix(step.Key, turnFinishKey) &&
			step.Status != serviceports.RunStepFailed {
			return false, nil
		}
	}

	verdict, err := a.steps.Claim(ctx, tenant, saveStep(ctx, payload))
	if err != nil {
		return false, fmt.Errorf("claim saving this turn: %w", err)
	}

	return verdict.State == serviceports.StepFresh, nil
}

func saveStep(ctx context.Context, payload *AssistantTurnPayload) serviceports.RunStep {
	attempt := int(activity.GetInfo(ctx).Attempt)

	return serviceports.RunStep{
		OwnerKind: serviceports.RunStepOwnerAssistantTurn,
		OwnerID:   payload.TurnID,
		Attempt:   attempt,
		Kind:      serviceports.RunStepCompletion,
		Key:       turnFinishKey + ":" + strconv.Itoa(attempt),
		ToolName:  turnFinishKey,
	}
}

// settle closes this attempt's claim so the ledger agrees with the record.
func (a *Activities) settle(
	ctx context.Context,
	payload *AssistantTurnPayload,
	status serviceports.RunStepStatus,
	content string,
) {
	step := saveStep(ctx, payload)
	step.Status = status
	step.Outcome = serviceports.RunStepOutcome{
		Content: content,
		Failed:  status == serviceports.RunStepFailed,
	}

	if err := a.steps.Settle(context.WithoutCancel(ctx), payload.tenantInfo(), step); err != nil {
		a.logger.Error("could not close the claim on saving a turn",
			zap.String("turn", payload.TurnID.String()),
			zap.Error(err),
		)
	}
}

// recordTrajectory keeps the durable account of what the turn said: every
// event but the streamed text, which the transcript already holds whole.
func (a *Activities) recordTrajectory(
	ctx context.Context,
	tenant pagination.TenantInfo,
	payload *AssistantTurnPayload,
	events []temporaltype.StreamItem,
	ending temporaltype.StreamItem,
) {
	if a.trajectory == nil {
		return
	}

	writer := a.trajectory.Recorder(ctx, tenant, serviceports.RunStepOwner{
		Kind: serviceports.RunStepOwnerAssistantTurn,
		ID:   payload.TurnID,
	})
	for _, event := range events {
		serviceports.RecordTrajectory(writer, ctx, serviceports.StreamEvent{
			Event: event.Event,
			Data:  event.Data,
		})
	}
	serviceports.RecordTrajectory(writer, ctx, serviceports.StreamEvent{
		Event: ending.Event,
		Data:  ending.Data,
	})
	serviceports.FlushTrajectory(writer, ctx)
}

// openStream is how the save reaches the reader: a proposal the turn raised
// is shown as a draft beside the conversation the moment it is saved. A
// stream that cannot be opened costs the reader those announcements, not the
// turn, so it is logged and the save goes on.
func (a *Activities) openStream(
	ctx context.Context,
) (*workflowstreams.Client, serviceports.AssistantStreamEmitter) {
	stream, err := workflowstreams.NewClientFromActivity(ctx, workflowstreams.Options{})
	if err != nil {
		a.logger.Warn("could not open this turn's stream to save it", zap.Error(err))

		return nil, nil
	}

	events := stream.Topic(temporaltype.StreamEventsTopic)

	return stream, func(event serviceports.StreamEvent) {
		events.Publish(temporaltype.StreamItem{Event: event.Event, Data: event.Data}, true)
	}
}

func (a *Activities) closeStream(ctx context.Context, stream *workflowstreams.Client) {
	if stream == nil {
		return
	}

	flushCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), streamFlushTimeout)
	defer cancel()

	if err := stream.Close(flushCtx); err != nil {
		a.logger.Warn("could not flush the last of a turn's stream", zap.Error(err))
	}
}

// artifactObserver keeps what a chat turn's tool calls produce, where they
// ran. It is agentflow's observer for a run on a conversation.
type artifactObserver struct {
	assistant *assistantservice.Service
}

func newArtifactObserver(assistant *assistantservice.Service) agentflow.ToolObserver {
	return &artifactObserver{assistant: assistant}
}

func (o *artifactObserver) ObserveTool(
	ctx context.Context,
	run agentflow.RunContext,
	observation serviceports.ToolObservation,
	emit serviceports.AssistantStreamEmitter,
) (*serviceports.ShownArtifact, []*assistantartifact.Artifact, error) {
	return o.assistant.ObserveTool(ctx, run.ThreadID, run.Actor, observation, emit)
}
