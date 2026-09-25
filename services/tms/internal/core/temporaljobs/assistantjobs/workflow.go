package assistantjobs

import (
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/services/assistantservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	// prepareTimeout bounds reading what a question needs: the thread, its
	// history, its files, and the scope guard's verdict.
	prepareTimeout = 2 * time.Minute

	// finishTimeout bounds saving a turn.
	finishTimeout = 2 * time.Minute
)

var prepareOptions = workflow.ActivityOptions{
	StartToCloseTimeout: prepareTimeout,
	Summary:             "Read the question",
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:        time.Second,
		BackoffCoefficient:     2,
		MaximumInterval:        10 * time.Second,
		MaximumAttempts:        3,
		NonRetryableErrorTypes: []string{errTypeNoActor, errTypeRejected},
	},
}

// finishOptions retry for longer than anything else in the turn. By the time
// the turn is saved, a model has been paid for and a tool may have written
// something, and the conversation is the only place a person can see that.
var finishOptions = workflow.ActivityOptions{
	StartToCloseTimeout: finishTimeout,
	Summary:             "Save the turn",
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2,
		MaximumInterval:    30 * time.Second,
		MaximumAttempts:    10,
	},
}

// closeOptions retry briefly. Closing is one row, and a turn whose save just
// failed ten times gains nothing from waiting longer on the same database.
var closeOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 30 * time.Second,
	Summary:             "Close the turn's record",
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2,
		MaximumInterval:    10 * time.Second,
		MaximumAttempts:    5,
	},
}

// notifyOptions retry a few times over a minute. Telling someone their reply
// is ready is worth a retry, not worth holding the turn open for.
var notifyOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 30 * time.Second,
	Summary:             "Tell the person their reply is ready",
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2,
		MaximumInterval:    15 * time.Second,
		MaximumAttempts:    5,
	},
}

// changeCloseUnsavedTurn closes a turn's record when saving the turn failed on
// every attempt. Executions that began before it replay without the step.
const changeCloseUnsavedTurn = "assistant-turn-close-unsaved"

// changeNotifyUnseenTurn tells the person who asked when their reply ended
// with nobody reading it. Executions that began before it replay without the
// step.
const changeNotifyUnseenTurn = "assistant-turn-notify-unseen"

// Workflows are the assistant's workflows. They hold the agent runtime
// because the agent loop runs in workflow code, and the loop is the runtime's.
type Workflows struct {
	runtime *agentruntime.Service
}

func NewWorkflows(runtime *agentruntime.Service) *Workflows {
	return &Workflows{runtime: runtime}
}

// AssistantTurnWorkflow answers one question.
//
// The reader follows it on the Workflow Stream it hosts, from the question
// being taken to the saved answer. A stop is a cancellation of this workflow:
// the model call in flight is cancelled, and what had happened by then is
// saved like any other ending.
func (w *Workflows) AssistantTurnWorkflow(
	ctx workflow.Context,
	payload *AssistantTurnPayload,
) (*AssistantTurnResult, error) {
	stream, err := agentflow.HostStream(ctx)
	if err != nil {
		return nil, err
	}

	finish := w.answer(ctx, stream, payload)

	// Whatever happened above, the turn is saved on a context a stop cannot
	// reach. A stopped turn is the one most worth keeping: the person saw
	// part of an answer, and the conversation should show them the same.
	keep, _ := workflow.NewDisconnectedContext(ctx)

	var a *Activities
	var ending TurnEnding
	err = workflow.ExecuteActivity(
		workflow.WithActivityOptions(keep, finishOptions), a.FinishTurnActivity, finish,
	).Get(keep, &ending)
	if err != nil {
		workflow.GetLogger(ctx).Error("assistant turn could not be saved",
			"turnId", payload.TurnID.String(),
			"error", err.Error(),
		)
		ending = *failedEnding("Failed", failedMessage)
		w.closeRecord(keep, payload, err)
	}

	stream.Publish(keep, ending.Event)
	stream.Close(keep)
	w.notifyUnseen(keep, stream, finish, &ending)

	if finish.Failure != nil && finish.Failure.Stopped {
		// Recorded as cancelled rather than completed, which is what it was.
		return nil, temporal.NewCanceledError(ending.Result.Message)
	}
	if err != nil {
		return nil, err
	}

	return &ending.Result, nil
}

// answer runs the question to the end of the agent loop, and returns what the
// turn did for saving. Nothing here fails the workflow: every failure is part
// of what is saved.
func (w *Workflows) answer(
	ctx workflow.Context,
	stream *agentflow.Stream,
	payload *AssistantTurnPayload,
) *FinishTurnInput {
	finish := &FinishTurnInput{Payload: payload}

	var a *Activities
	var plan assistantservice.TurnPlan
	err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, withPriority(prepareOptions, payload)),
		a.PrepareTurnActivity, payload,
	).Get(ctx, &plan)
	if err != nil {
		finish.Failure = modelcall.FailureOf(err)
		finish.Rejection = rejectionOf(err)

		return finish
	}
	finish.Plan = &plan

	opening := plan.Opening()
	openedAt := workflow.Now(ctx).Unix()
	stream.Publish(ctx, temporaltype.StreamItem{
		Event: opening.Event,
		Data:  opening.Data,
		At:    openedAt,
	})
	finish.Events = append(finish.Events, temporaltype.StreamItem{
		Event: opening.Event,
		Data:  opening.Data,
		At:    openedAt,
	})
	if plan.Refused() {
		return finish
	}

	run := agentflow.NewRunContext(plan.RunRequest(&payload.Actor), agentflow.PriorityInteractive)
	run.StepOwner = serviceports.RunStepOwner{
		Kind: serviceports.RunStepOwnerAssistantTurn,
		ID:   payload.TurnID,
	}

	outcome, err := agentflow.Run(ctx, w.runtime, stream.Events(), run, plan.Turn)
	finish.Run = outcome.Result
	finish.Artifacts = outcome.Artifacts
	finish.Events = append(finish.Events, outcome.Events...)
	finish.Failure = modelcall.FailureOf(err)

	return finish
}

// rejectionOf is the reason a question was turned away, when it was turned
// away for a reason written for the person who asked.
func rejectionOf(err error) string {
	var appErr *temporal.ApplicationError
	if errors.As(err, &appErr) && appErr.Type() == errTypeRejected {
		return appErr.Message()
	}

	return ""
}

func withPriority(
	options workflow.ActivityOptions,
	payload *AssistantTurnPayload,
) workflow.ActivityOptions {
	options.Priority = temporal.Priority{
		PriorityKey: agentflow.PriorityInteractive,
		FairnessKey: payload.OrganizationID.String(),
	}

	return options
}

// notifyUnseen tells the person who asked that their reply ended, when nobody
// was reading when it did: they closed the tab, signed in somewhere else, or
// asked from the palette and moved on. A reply they stopped themselves is not
// news to them, and a caller waiting on the result already has it.
func (w *Workflows) notifyUnseen(
	ctx workflow.Context,
	stream *agentflow.Stream,
	finish *FinishTurnInput,
	ending *TurnEnding,
) {
	payload := finish.Payload
	if stream.Drained() || payload.Request.Awaited {
		return
	}
	if finish.Failure != nil && finish.Failure.Stopped {
		return
	}

	status := conversation.AssistantTurnStatus(ending.Result.Status)
	if !notifiesUnseen(status) {
		return
	}
	if workflow.GetVersion(ctx, changeNotifyUnseenTurn, workflow.DefaultVersion, 1) != 1 {
		return
	}

	var a *Activities
	err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, notifyOptions),
		a.NotifyUnseenTurnActivity, &NotifyUnseenTurnInput{
			Payload: payload,
			Status:  status,
		},
	).Get(ctx, nil)
	if err != nil {
		workflow.GetLogger(ctx).Error("could not tell the person their reply is ready",
			"turnId", payload.TurnID.String(),
			"error", err.Error(),
		)
	}
}

// closeRecord closes the record of a turn that could not be saved, so the
// conversation is not left refusing every later question.
func (w *Workflows) closeRecord(
	ctx workflow.Context,
	payload *AssistantTurnPayload,
	cause error,
) {
	if workflow.GetVersion(ctx, changeCloseUnsavedTurn, workflow.DefaultVersion, 1) != 1 {
		return
	}

	var a *Activities
	err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, closeOptions),
		a.CloseTurnActivity, payload, "This reply could not be saved: "+cause.Error(),
	).Get(ctx, nil)
	if err != nil {
		workflow.GetLogger(ctx).Error("assistant turn record could not be closed",
			"turnId", payload.TurnID.String(),
			"error", err.Error(),
		)
	}
}
