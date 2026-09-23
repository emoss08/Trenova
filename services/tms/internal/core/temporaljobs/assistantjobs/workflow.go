package assistantjobs

import (
	"errors"
	"time"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/services/assistantservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/contrib/workflowstreams"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	// drainWindow is how long a finished turn waits for its reader to say it
	// has the last event. A reader that is there says so within a poll or
	// two; one that is not is not coming, and the turn closes without it.
	drainWindow = 15 * time.Second

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
	stream, err := workflowstreams.NewWorkflowStream(ctx, nil)
	if err != nil {
		return nil, err
	}
	events := stream.Topic(agentflow.EventsTopic)

	drained := false
	workflow.Go(ctx, func(ctx workflow.Context) {
		workflow.GetSignalChannel(ctx, temporaltype.SignalStreamDrained).Receive(ctx, nil)
		drained = true
	})

	finish := w.answer(ctx, events, payload)

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
	}

	publish(keep, events, ending.Event)
	close(keep, stream, &drained)

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
	events *workflowstreams.WorkflowTopicHandle,
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
		finish.Failure = agentflow.FailureOf(err)
		finish.Rejection = rejectionOf(err)

		return finish
	}
	finish.Plan = &plan

	opening := plan.Opening()
	publish(ctx, events, temporaltype.StreamItem{Event: opening.Event, Data: opening.Data})
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

	outcome, err := agentflow.Run(ctx, w.runtime, events, run, plan.Turn)
	finish.Run = outcome.Result
	finish.Artifacts = outcome.Artifacts
	finish.Events = append(finish.Events, outcome.Events...)
	finish.Failure = agentflow.FailureOf(err)

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

func publish(
	ctx workflow.Context,
	events *workflowstreams.WorkflowTopicHandle,
	item temporaltype.StreamItem,
) {
	if err := events.Publish(item); err != nil {
		workflow.GetLogger(ctx).Warn("could not publish a turn event",
			"event", item.Event,
			"error", err.Error(),
		)
	}
}

// close hands the last events to the reader before the workflow ends.
//
// A stream is read by polling the workflow, so a workflow that has closed can
// no longer be read. It waits, bounded, for the reader to say it has the last
// event, then releases any poll still waiting and lets its handlers finish.
func close(
	ctx workflow.Context,
	stream *workflowstreams.WorkflowStream,
	drained *bool,
) {
	if _, err := workflow.AwaitWithTimeout(ctx, drainWindow, func() bool {
		return *drained
	}); err != nil {
		workflow.GetLogger(ctx).Warn("waiting for the reader was cut short",
			"error", err.Error())
	}

	stream.DetachPollers()
	if err := workflow.Await(ctx, func() bool {
		return workflow.AllHandlersFinished(ctx)
	}); err != nil {
		workflow.GetLogger(ctx).Warn("waiting for the turn's readers was cut short",
			"error", err.Error())
	}
}
