package agentwaitjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agentwait"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	readTimeout = 30 * time.Second
	// finishWindow bounds how long a wait that ended keeps trying to pick the
	// work up, which is how long its conversation may stay busy.
	finishWindow = 24 * time.Hour
)

var readOptions = workflow.ActivityOptions{
	StartToCloseTimeout: readTimeout,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    10,
	},
}

var finishOptions = workflow.ActivityOptions{
	StartToCloseTimeout:    time.Minute,
	ScheduleToCloseTimeout: finishWindow,
	Summary:                "Pick the parked work up",
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    15 * time.Second,
		BackoffCoefficient: 1.5,
		MaximumInterval:    5 * time.Minute,
	},
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        WorkflowName,
			Fn:          AgentWaitWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Hold an agent's parked work until what it waits for happens",
		},
		{
			Name:        ReconcileWorkflowName,
			Fn:          ReconcileAgentWaitsWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Close and pick up the agent waits whose workflow vanished",
		},
	}
}

// AgentWaitWorkflow holds one wait. It sleeps until the wait comes due or runs
// out, or until it is told that what it waits for happened, and then hands
// the work back. A cancelled wait is a cancelled workflow: the service has
// already closed the wait, so there is nothing left to do.
func AgentWaitWorkflow(ctx workflow.Context, payload *Payload) error {
	var a *Activities
	readCtx := workflow.WithActivityOptions(ctx, readOptions)

	var schedule Schedule
	if err := workflow.ExecuteActivity(readCtx, a.ScheduleWaitActivity, payload).
		Get(ctx, &schedule); err != nil {
		return err
	}
	if !schedule.Open {
		return nil
	}

	met := workflow.GetSignalChannel(ctx, MetSignal)
	reschedule := workflow.GetSignalChannel(ctx, RescheduleSignal)
	for {
		woke, signal, err := sleepUntilNext(ctx, met, reschedule, &schedule)
		if ctx.Err() != nil {
			return nil
		}
		if err != nil {
			return err
		}
		switch woke {
		case wokeMet:
			return finish(ctx, payload, agentwait.StatusMet, signal.Detail)
		case wokeRescheduled:
			if err = workflow.ExecuteActivity(readCtx, a.ScheduleWaitActivity, payload).
				Get(ctx, &schedule); err != nil {
				return err
			}
			if !schedule.Open {
				return nil
			}
			continue
		case wokeTimer:
		}

		now := workflow.Now(ctx).Unix()
		if now >= schedule.ExpiresAt {
			return finish(ctx, payload, agentwait.StatusTimedOut, "")
		}

		var check Check
		if err = workflow.ExecuteActivity(readCtx, a.CheckWaitActivity, &CheckInput{
			Payload: payload,
			Now:     now,
		}).Get(ctx, &check); err != nil {
			return err
		}
		switch {
		case check.Closed:
			return nil
		case check.Met:
			return finish(ctx, payload, agentwait.StatusMet, check.Detail)
		default:
			schedule.DueAt = check.DueAt
		}
	}
}

type wakeReason int

const (
	wokeTimer wakeReason = iota
	wokeMet
	wokeRescheduled
)

// sleepUntilNext waits for the wait to come due, run out, be met or be told
// its due time moved, and says which.
func sleepUntilNext(
	ctx workflow.Context,
	met, reschedule workflow.ReceiveChannel,
	schedule *Schedule,
) (wakeReason, Met, error) {
	wake := schedule.ExpiresAt
	if schedule.DueAt != nil && *schedule.DueAt < wake {
		wake = *schedule.DueAt
	}
	wait := time.Unix(wake, 0).Sub(workflow.Now(ctx))
	if wait < 0 {
		wait = 0
	}

	timerCtx, cancelTimer := workflow.WithCancel(ctx)
	defer cancelTimer()
	timer := workflow.NewTimer(timerCtx, wait)

	var signal Met
	woke := wokeTimer
	var timerErr error
	selector := workflow.NewSelector(ctx)
	selector.AddReceive(met, func(c workflow.ReceiveChannel, _ bool) {
		c.Receive(ctx, &signal)
		woke = wokeMet
	})
	selector.AddReceive(reschedule, func(c workflow.ReceiveChannel, _ bool) {
		for c.ReceiveAsync(nil) {
		}
		woke = wokeRescheduled
	})
	selector.AddFuture(timer, func(f workflow.Future) {
		timerErr = f.Get(ctx, nil)
	})
	selector.Select(ctx)

	if woke != wokeTimer {
		return woke, signal, nil
	}

	return wokeTimer, Met{}, timerErr
}

// ReconcileAgentWaitsWorkflow closes the waits whose own workflow is gone
// without having closed them, and picks their work up.
func ReconcileAgentWaitsWorkflow(ctx workflow.Context) (int, error) {
	var a *Activities
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		Summary:             "Close the waits whose workflow vanished",
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 3},
	})

	var closed int
	err := workflow.ExecuteActivity(ctx, a.ReconcileOverdueWaitsActivity).Get(ctx, &closed)

	return closed, err
}

func finish(
	ctx workflow.Context,
	payload *Payload,
	status agentwait.Status,
	outcome string,
) error {
	var a *Activities
	finishCtx := workflow.WithActivityOptions(ctx, finishOptions)

	return workflow.ExecuteActivity(finishCtx, a.FinishWaitActivity, &FinishInput{
		Payload: payload,
		Status:  status,
		Outcome: outcome,
	}).Get(ctx, nil)
}
