package portaljobs

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/extensionportal"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/portaljobs/portalcontract"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var pollOptions = workflow.ActivityOptions{
	StartToCloseTimeout: pollTimeout,
	Summary:             "Ask the vendor how the portal task is going",
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    2 * time.Second,
		BackoffCoefficient: 2,
		MaximumInterval:    30 * time.Second,
		MaximumAttempts:    5,
	},
}

var notifyOptions = workflow.ActivityOptions{
	StartToCloseTimeout: pollTimeout,
	Summary:             "Hand the portal's answer back to the work that asked",
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    10,
	},
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        portalcontract.WorkflowName,
			Fn:          PortalRunWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Follow an approved portal task to its end and hand its answer back",
		},
	}
}

// PortalRunWorkflow polls one portal task until it ends or runs past its
// deadline, then hands what the portal said back to the work that asked. A
// poll that keeps failing still ends the task: it is stopped and recorded as
// failed rather than left running unwatched. Time spent waiting for a person
// to enter a sign-in code is not counted against the deadline; the run's own
// code wait bounds it.
func PortalRunWorkflow(ctx workflow.Context, payload *portalcontract.Payload) error {
	var a *Activities
	pollCtx := workflow.WithActivityOptions(ctx, pollOptions)
	notifyCtx := workflow.WithActivityOptions(ctx, notifyOptions)

	deadline := workflow.Now(ctx).Add(actDeadline)
	for {
		var poll portalcontract.PollResult
		err := workflow.ExecuteActivity(pollCtx, a.PollPortalRunActivity, payload).
			Get(ctx, &poll)
		if err != nil {
			return stop(ctx, pollCtx, notifyCtx, &portalcontract.StopInput{
				Payload: payload,
				Status:  extensionportal.PortalRunStatusFailed,
				Reason:  unreachableNote,
			})
		}
		if poll.Finished {
			return workflow.ExecuteActivity(notifyCtx, a.NotifyPortalRunActivity, payload).
				Get(ctx, nil)
		}
		if poll.AwaitingCode {
			deadline = deadline.Add(pollInterval)
		}
		if !workflow.Now(ctx).Before(deadline) {
			return stop(ctx, pollCtx, notifyCtx, &portalcontract.StopInput{
				Payload: payload,
				Status:  extensionportal.PortalRunStatusTimedOut,
				Reason:  timedOutNote,
			})
		}
		if err = workflow.Sleep(ctx, pollInterval); err != nil {
			return err
		}
	}
}

func stop(ctx, pollCtx, notifyCtx workflow.Context, in *portalcontract.StopInput) error {
	var a *Activities
	if err := workflow.ExecuteActivity(pollCtx, a.StopPortalRunActivity, in).
		Get(ctx, nil); err != nil {
		return err
	}

	return workflow.ExecuteActivity(notifyCtx, a.NotifyPortalRunActivity, in.Payload).
		Get(ctx, nil)
}
