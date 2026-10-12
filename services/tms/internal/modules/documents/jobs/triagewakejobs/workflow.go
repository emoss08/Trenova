package triagewakejobs

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/jobs/triagewakejobs/triagewakecontract"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var readOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 30 * time.Second,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    10,
	},
}

var sweepOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 10 * time.Minute,
	RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 3},
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        triagewakecontract.WorkflowName,
			Fn:          TriageWakeWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Bring a snoozed or waiting inbound message back to the queue when its time comes",
		},
		{
			Name:        triagewakecontract.ReconcileWorkflowName,
			Fn:          ReconcileTriageWakesWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Wake the parked inbound messages whose own timer was lost",
		},
		{
			Name:        triagewakecontract.RecomputeDueWorkflowName,
			Fn:          RecomputeTriageDueWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Recompute due times after an organization changes its response targets",
		},
	}
}

func TriageWakeWorkflow(ctx workflow.Context, payload *Payload) error {
	var a *Activities
	readCtx := workflow.WithActivityOptions(ctx, readOptions)

	var schedule Schedule
	if err := workflow.ExecuteActivity(readCtx, a.ScheduleTriageWakeActivity, payload).
		Get(ctx, &schedule); err != nil {
		return err
	}

	reschedule := workflow.GetSignalChannel(ctx, triagewakecontract.RescheduleSignal)
	for schedule.Open {
		rescheduled, err := sleepUntil(ctx, reschedule, schedule.WakeAt)
		if ctx.Err() != nil {
			return nil //nolint:nilerr // a canceled wake ends cleanly; whoever canceled it settled the message
		}
		if err != nil {
			return err
		}
		if rescheduled {
			if err = workflow.ExecuteActivity(readCtx, a.ScheduleTriageWakeActivity, payload).
				Get(ctx, &schedule); err != nil {
				return err
			}
			continue
		}

		var result WakeResult
		if err = workflow.ExecuteActivity(readCtx, a.WakeTriageMessageActivity, &WakeInput{
			Payload: payload,
			Now:     workflow.Now(ctx).Unix(),
		}).Get(ctx, &result); err != nil {
			return err
		}
		if result.Woken {
			return nil
		}
		schedule = result.Schedule
	}

	return nil
}

func sleepUntil(
	ctx workflow.Context,
	reschedule workflow.ReceiveChannel,
	wakeAt int64,
) (bool, error) {
	wait := time.Unix(wakeAt, 0).Sub(workflow.Now(ctx))
	if wait < 0 {
		wait = 0
	}

	timerCtx, cancelTimer := workflow.WithCancel(ctx)
	defer cancelTimer()
	timer := workflow.NewTimer(timerCtx, wait)

	rescheduled := false
	var timerErr error
	selector := workflow.NewSelector(ctx)
	selector.AddReceive(reschedule, func(c workflow.ReceiveChannel, _ bool) {
		for c.ReceiveAsync(nil) {
		}
		rescheduled = true
	})
	selector.AddFuture(timer, func(f workflow.Future) {
		timerErr = f.Get(ctx, nil)
	})
	selector.Select(ctx)

	if rescheduled {
		return true, nil
	}

	return false, timerErr
}

func ReconcileTriageWakesWorkflow(ctx workflow.Context) (int, error) {
	var a *Activities
	ctx = workflow.WithActivityOptions(ctx, sweepOptions)

	var woken int
	err := workflow.ExecuteActivity(ctx, a.ReconcileTriageWakesActivity).Get(ctx, &woken)

	return woken, err
}

func RecomputeTriageDueWorkflow(ctx workflow.Context, payload *RecomputeDuePayload) (int64, error) {
	var a *Activities
	ctx = workflow.WithActivityOptions(ctx, sweepOptions)

	var changed int64
	err := workflow.ExecuteActivity(ctx, a.RecomputeTriageDueActivity, payload).Get(ctx, &changed)

	return changed, err
}
