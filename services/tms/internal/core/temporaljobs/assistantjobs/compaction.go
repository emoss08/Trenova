package assistantjobs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/services/assistantservice"
	"github.com/emoss08/trenova/internal/core/services/assistantturnservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

const CompactionWorkflowName = "ConversationCompactionWorkflow"

// errTypeCompactionRejected is a compaction turned away for a reason written
// for the person: nothing to compact, an agent switched off, a spent budget.
const errTypeCompactionRejected = "CompactionRejected"

// The messages a compaction that did not finish ends with.
const (
	compactionFailedMessage = "The conversation could not be compacted. Nothing was changed; " +
		"try again in a moment."
	compactionCancelledMessage = "Compacting was cancelled. The conversation is as it was."
)

// summaryTimeout bounds the summarizing model call. A summary is short, but
// the model reads a long stretch first, and a reasoning model thinks before it
// writes.
const summaryTimeout = 5 * time.Minute

var prepareCompactionOptions = workflow.ActivityOptions{
	StartToCloseTimeout: prepareTimeout,
	Summary:             "Read the conversation to compact",
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:        time.Second,
		BackoffCoefficient:     2,
		MaximumInterval:        10 * time.Second,
		MaximumAttempts:        3,
		NonRetryableErrorTypes: []string{errTypeNoActor, errTypeCompactionRejected},
	},
}

var summaryOptions = workflow.ActivityOptions{
	StartToCloseTimeout: summaryTimeout,
	HeartbeatTimeout:    modelcall.HeartbeatTimeout,
	// A cancel reaches the model call and the workflow waits for it to stop,
	// so nothing is still being billed once the person is told it stopped.
	WaitForCancellation: true,
	Summary:             "Summarize the conversation",
	RetryPolicy:         modelcall.RetryPolicy(3),
}

var finishCompactionOptions = workflow.ActivityOptions{
	StartToCloseTimeout: finishTimeout,
	Summary:             "Save the summary",
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:        time.Second,
		BackoffCoefficient:     2,
		MaximumInterval:        30 * time.Second,
		MaximumAttempts:        10,
		NonRetryableErrorTypes: []string{errTypeCompactionRejected},
	},
}

// startCompactionOptions try twice. Starting is one row and one workflow
// start, and a compaction that cannot start leaves the conversation as it is.
var startCompactionOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 30 * time.Second,
	Summary:             "Start compacting the conversation",
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval: time.Second,
		MaximumAttempts: 2,
	},
}

// CompactionPayload is one compaction of a conversation.
type CompactionPayload struct {
	temporaltype.BasePayload

	TurnID   pulid.ID                  `json:"turnId"`
	ThreadID pulid.ID                  `json:"threadId"`
	Actor    serviceports.RequestActor `json:"actor"`
	// Auto says the conversation is compacting itself.
	Auto bool `json:"auto"`
}

func (p *CompactionPayload) tenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  p.OrganizationID,
		BuID:   p.BusinessUnitID,
		UserID: p.Actor.UserID,
	}
}

// StartedCompaction names the turn compacting a conversation; empty when none
// was started.
type StartedCompaction struct {
	TurnID pulid.ID `json:"turnId"`
}

// SummarizedInput is a compaction's plan and what the model wrote for it.
type SummarizedInput struct {
	Payload *CompactionPayload               `json:"payload"`
	Plan    *assistantservice.CompactionPlan `json:"plan"`
	Reply   *agentruntime.ModelReply         `json:"reply"`
}

// CompactionEndInput is how a compaction that saved nothing ended.
type CompactionEndInput struct {
	Payload *CompactionPayload               `json:"payload"`
	Status  conversation.AssistantTurnStatus `json:"status"`
	Message string                           `json:"message"`
}

// CompactionEnding is the last event a compaction's reader is sent.
type CompactionEnding struct {
	Event temporaltype.StreamItem `json:"event"`
}

// StartCompactionWorkflow hands a recorded compaction turn to a worker.
func StartCompactionWorkflow(
	ctx context.Context,
	workflows serviceports.WorkflowStarter,
	turn *conversation.AssistantTurn,
	actor serviceports.RequestActor,
	auto bool,
) (client.WorkflowRun, error) {
	return workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID:                                       WorkflowIDFor(turn.ID),
		TaskQueue:                                temporaltype.TaskQueueAgentChat.String(),
		WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		WorkflowExecutionErrorWhenAlreadyStarted: true,
		StaticSummary:                            "Compact a conversation",
		Priority: temporal.Priority{
			PriorityKey: agentflow.PriorityInteractive,
			FairnessKey: actor.OrganizationID.String(),
		},
	}, CompactionWorkflowName, &CompactionPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: actor.OrganizationID,
			BusinessUnitID: actor.BusinessUnitID,
			UserID:         actor.UserID,
			Timestamp:      timeutils.NowUnix(),
		},
		TurnID:   turn.ID,
		ThreadID: turn.ThreadID,
		Actor:    actor,
		Auto:     auto,
	})
}

// ConversationCompactionWorkflow summarizes the older part of a conversation
// so it fits its model's window again.
//
// It runs as a turn of the conversation: it holds the conversation's one live
// slot, so nothing is asked while the history is rewritten, and it is
// followed and stopped like a reply. A stop cancels the summarizing call and
// leaves the conversation exactly as it was; the summary is saved only once
// it is whole.
func (w *Workflows) ConversationCompactionWorkflow(
	ctx workflow.Context,
	payload *CompactionPayload,
) error {
	stream, err := agentflow.HostStream(ctx)
	if err != nil {
		return err
	}

	ending := w.compact(ctx, stream, payload)

	keep, _ := workflow.NewDisconnectedContext(ctx)
	stream.Publish(keep, ending.Event)
	stream.Close(keep)

	return nil
}

// compact runs the compaction to its end and says how it ended. Nothing here
// fails the workflow: every ending is told to the reader and recorded.
func (w *Workflows) compact(
	ctx workflow.Context,
	stream *agentflow.Stream,
	payload *CompactionPayload,
) CompactionEnding {
	var a *Activities
	var plan assistantservice.CompactionPlan
	err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, prepareCompactionOptions),
		a.PrepareCompactionActivity, payload,
	).Get(ctx, &plan)
	if err != nil {
		return w.endCompaction(ctx, payload, err, rejectionMessage(err))
	}

	stream.Publish(ctx, temporaltype.StreamItem{
		Event: serviceports.AssistantEventCompactionStarted,
		Data: serviceports.AssistantCompactionEvent{
			TurnID:   payload.TurnID,
			ThreadID: payload.ThreadID,
			Auto:     plan.Auto,
			Before:   plan.Before,
			After:    plan.After,
		},
		At: workflow.Now(ctx).Unix(),
	})

	var flow *agentflow.Activities
	var reply agentruntime.ModelReply
	summaryCtx := workflow.WithActivityOptions(ctx, withCompactionPriority(summaryOptions, payload))
	err = workflow.ExecuteActivity(summaryCtx, flow.ModelCallActivity, &agentflow.ModelCallInput{
		Request: plan.Request,
	}).Get(summaryCtx, &reply)
	if err != nil {
		return w.endCompaction(ctx, payload, err, "")
	}

	// The summary is whole: saving it is no longer the person's to stop.
	keep, _ := workflow.NewDisconnectedContext(ctx)
	var ending CompactionEnding
	err = workflow.ExecuteActivity(
		workflow.WithActivityOptions(keep, finishCompactionOptions),
		a.FinishCompactionActivity, &SummarizedInput{Payload: payload, Plan: &plan, Reply: &reply},
	).Get(keep, &ending)
	if err != nil {
		return w.endCompaction(keep, payload, err, rejectionMessage(err))
	}

	return ending
}

// endCompaction closes a compaction that saved nothing: cancelled, turned
// away, or failed.
func (w *Workflows) endCompaction(
	ctx workflow.Context,
	payload *CompactionPayload,
	cause error,
	rejection string,
) CompactionEnding {
	in := &CompactionEndInput{
		Payload: payload,
		Status:  conversation.AssistantTurnStatusFailed,
		Message: compactionFailedMessage,
	}
	switch {
	case temporal.IsCanceledError(cause):
		in.Status = conversation.AssistantTurnStatusStopped
		in.Message = compactionCancelledMessage
	case rejection != "":
		in.Message = rejection
	}

	keep, _ := workflow.NewDisconnectedContext(ctx)
	var a *Activities
	var ending CompactionEnding
	err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(keep, closeOptions), a.EndCompactionActivity, in,
	).Get(keep, &ending)
	if err != nil {
		workflow.GetLogger(ctx).Error("could not close a compaction's record",
			"turnId", payload.TurnID.String(),
			"error", err.Error(),
		)

		return compactionEndingFor(in, false)
	}

	return ending
}

func withCompactionPriority(
	options workflow.ActivityOptions,
	payload *CompactionPayload,
) workflow.ActivityOptions {
	options.Priority = temporal.Priority{
		PriorityKey: agentflow.PriorityInteractive,
		FairnessKey: payload.OrganizationID.String(),
	}

	return options
}

// rejectionMessage is the reason a compaction was turned away, when it was
// turned away for a reason written for the person.
func rejectionMessage(err error) string {
	var appErr *temporal.ApplicationError
	if errors.As(err, &appErr) && appErr.Type() == errTypeCompactionRejected {
		return appErr.Message()
	}

	return ""
}

// compactionEndingFor is the event that ends a compaction that saved
// nothing: cancelled, which leaves the conversation as it was, or failed.
func compactionEndingFor(in *CompactionEndInput, autoOff bool) CompactionEnding {
	if in.Status == conversation.AssistantTurnStatusStopped {
		return CompactionEnding{Event: temporaltype.StreamItem{
			Event: serviceports.AssistantEventCompactionCancelled,
			Data: serviceports.AssistantCompactionEvent{
				TurnID:         in.Payload.TurnID,
				ThreadID:       in.Payload.ThreadID,
				Auto:           in.Payload.Auto,
				AutoCompactOff: autoOff,
			},
		}}
	}

	return CompactionEnding{Event: temporaltype.StreamItem{
		Event: serviceports.AssistantEventError,
		Data: map[string]any{
			"message":    in.Message,
			"compaction": true,
		},
	}}
}

// PrepareCompactionActivity reads the conversation and makes the summarizing
// request ready.
func (a *Activities) PrepareCompactionActivity(
	ctx context.Context,
	payload *CompactionPayload,
) (*assistantservice.CompactionPlan, error) {
	if payload.Actor.UserID.IsNil() {
		return nil, temporal.NewNonRetryableApplicationError(
			"this compaction arrived without the person who asked for it", errTypeNoActor, nil,
		)
	}

	plan, err := a.assistant.PrepareCompaction(ctx, &assistantservice.CompactRequest{
		ThreadID:   payload.ThreadID,
		TurnID:     payload.TurnID,
		TenantInfo: payload.tenantInfo(),
		Auto:       payload.Auto,
	}, &payload.Actor)
	if err != nil {
		if rejected(err) {
			return nil, temporal.NewNonRetryableApplicationError(
				err.Error(), errTypeCompactionRejected, err,
			)
		}

		return nil, fmt.Errorf("prepare this compaction: %w", err)
	}

	return plan, nil
}

// FinishCompactionActivity saves the summary, closes the compaction's record
// and says how the conversation reads now.
func (a *Activities) FinishCompactionActivity(
	ctx context.Context,
	in *SummarizedInput,
) (*CompactionEnding, error) {
	payload := in.Payload
	reply := compactionReply(in.Reply)

	result, err := a.assistant.FinishCompaction(ctx, in.Plan, reply, &payload.Actor)
	if err != nil {
		if errortypes.IsBusinessError(err) {
			return nil, temporal.NewNonRetryableApplicationError(
				err.Error(), errTypeCompactionRejected, err,
			)
		}

		return nil, fmt.Errorf("save this compaction: %w", err)
	}

	a.closeCompaction(ctx, payload, conversation.AssistantTurnStatusCompleted, "")

	after := 0
	if result.Message.Compaction != nil {
		after = result.Message.Compaction.After
	}

	return &CompactionEnding{Event: temporaltype.StreamItem{
		Event: serviceports.AssistantEventCompactionFinished,
		Data: serviceports.AssistantCompactionEvent{
			TurnID:   payload.TurnID,
			ThreadID: payload.ThreadID,
			Auto:     in.Plan.Auto,
			Before:   in.Plan.Before,
			After:    after,
			Message:  result.Message,
			Usage:    result.Usage,
		},
	}}, nil
}

// EndCompactionActivity closes a compaction that saved nothing. Cancelling a
// compaction that started on its own also stops the conversation compacting
// itself, or it would start again on the next turn.
func (a *Activities) EndCompactionActivity(
	ctx context.Context,
	in *CompactionEndInput,
) (*CompactionEnding, error) {
	payload := in.Payload
	autoOff := false
	if in.Status == conversation.AssistantTurnStatusStopped && payload.Auto {
		if err := a.assistant.StopCompactingItself(
			ctx,
			payload.ThreadID,
			payload.tenantInfo(),
		); err != nil {
			a.logger.Warn("could not stop a conversation compacting itself",
				zap.String("thread", payload.ThreadID.String()),
				zap.Error(err),
			)
		} else {
			autoOff = true
		}
	}

	a.closeCompaction(ctx, payload, in.Status, in.Message)
	ending := compactionEndingFor(in, autoOff)

	return &ending, nil
}

// closeCompaction closes the compaction's record, which frees the
// conversation, and starts the report of any decision made while the
// conversation was busy compacting.
func (a *Activities) closeCompaction(
	ctx context.Context,
	payload *CompactionPayload,
	status conversation.AssistantTurnStatus,
	message string,
) {
	turn, err := a.turnRepo.GetByID(ctx, repositories.GetAssistantTurnRequest{
		ID:         payload.TurnID,
		TenantInfo: payload.tenantInfo(),
	})
	if err != nil {
		a.logger.Error("could not read a compaction's record",
			zap.String("turn", payload.TurnID.String()),
			zap.Error(err),
		)
		return
	}
	if !turn.Status.Terminal() {
		var cause error
		if message != "" {
			cause = errors.New(message)
		}
		a.turns.Complete(ctx, turn, status, cause)
	}

	a.continueConversation(ctx, &conversationContinuation{
		tenant:   payload.tenantInfo(),
		threadID: payload.ThreadID,
		userID:   payload.Actor.UserID,
		status:   status,
		resume:   true,
	})
}

// StartAutoCompactionActivity starts a conversation compacting itself, once
// the turn that filled it has closed. It reports no turn when the
// conversation is busy again, which is not a failure: whatever took it will
// cue the compaction when it ends.
func (a *Activities) StartAutoCompactionActivity(
	ctx context.Context,
	payload *AssistantTurnPayload,
) (*StartedCompaction, error) {
	turn, err := StartCompaction(ctx, a.turns, a.workflows, StartCompactionRequest{
		ThreadID:   payload.ThreadID,
		TenantInfo: payload.tenantInfo(),
		Actor:      payload.Actor,
		Auto:       true,
	})
	if err != nil {
		if errortypes.IsBusinessError(err) {
			return &StartedCompaction{}, nil
		}

		return nil, err
	}

	return &StartedCompaction{TurnID: turn.ID}, nil
}

// StartCompactionRequest asks for a conversation to be compacted.
type StartCompactionRequest struct {
	ThreadID   pulid.ID
	TenantInfo pagination.TenantInfo
	Actor      serviceports.RequestActor
	Auto       bool
}

// compactionTurns is the part of the turn service starting a compaction
// needs.
type compactionTurns interface {
	StartTurn(
		ctx context.Context,
		req assistantturnservice.StartRequest,
		start func(turn *conversation.AssistantTurn) (string, error),
	) (*conversation.AssistantTurn, error)
}

// StartCompaction records a compaction as the conversation's live turn and
// hands it to a worker. A person asking and a conversation compacting itself
// both start here.
func StartCompaction(
	ctx context.Context,
	turns compactionTurns,
	workflows serviceports.WorkflowStarter,
	req StartCompactionRequest,
) (*conversation.AssistantTurn, error) {
	if workflows == nil {
		return nil, errors.New("no workflow client to start a compaction with")
	}

	return turns.StartTurn(ctx, assistantturnservice.StartRequest{
		ThreadID:   req.ThreadID,
		UserID:     req.Actor.UserID,
		TenantInfo: req.TenantInfo,
		Origin:     conversation.AssistantTurnOriginCompaction,
	}, func(turn *conversation.AssistantTurn) (string, error) {
		run, err := StartCompactionWorkflow(ctx, workflows, turn, req.Actor, req.Auto)
		if err != nil {
			return "", err
		}

		return run.GetID(), nil
	})
}

// compactionReply is what the model wrote, as the conversation keeps it.
func compactionReply(reply *agentruntime.ModelReply) *assistantservice.CompactionReply {
	out := &assistantservice.CompactionReply{}
	if reply == nil || reply.Completion == nil {
		return out
	}
	completion := reply.Completion
	out.Summary = completion.Text
	out.Model = completion.ModelIdentifier
	out.ProviderID = completion.ProviderID
	out.ContextWindow = completion.ContextWindow
	out.Input = completion.InputTokens
	out.Output = completion.OutputTokens

	return out
}
