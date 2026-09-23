package assistantjobs

import (
	"errors"
	"time"

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

// changeCloseUnsavedTurn closes a turn's record when saving the turn failed on
// every attempt. Executions that began before it replay without the step.
const changeCloseUnsavedTurn = "assistant-turn-close-unsaved"

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
	stream.Publish(ctx, temporaltype.StreamItem{Event: opening.Event, Data: opening.Data})
	finish.Events = append(finish.Events, temporaltype.StreamItem{
		Event: opening.Event,
		Data:  opening.Data,
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
