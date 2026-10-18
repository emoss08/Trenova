package webwatchjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/jobs/webwatchjobs/webwatchcontract"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var checkOptions = workflow.ActivityOptions{
	StartToCloseTimeout: checkTimeout,
	Summary:             "Check the web watch for new findings and hand them on",
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2,
		MaximumInterval:    2 * time.Minute,
		MaximumAttempts:    5,
	},
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        webwatchcontract.WorkflowName,
			Fn:          WebWatchWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Follow a web watch to its end and hand on what it finds",
		},
	}
}

// WebWatchWorkflow checks one watch every half hour, or at once when it is
// signalled, until a check says the watch has ended. A check that keeps
// failing is tried again at the next one rather than ending the watch: the
// watch ends only when its time runs out, it is stopped, or its vendor can no
// longer run it. After a run of checks it continues as a new execution of
// the same watch, so its history stays small.
func WebWatchWorkflow(ctx workflow.Context, payload *webwatchcontract.Payload) error {
	var a *Activities
	checkCtx := workflow.WithActivityOptions(ctx, checkOptions)
	signals := workflow.GetSignalChannel(ctx, webwatchcontract.SignalCheck)
	logger := workflow.GetLogger(ctx)

	for range continueAfter {
		var result webwatchcontract.CheckResult
		if err := workflow.ExecuteActivity(checkCtx, a.CheckWebWatchActivity, payload).
			Get(ctx, &result); err != nil {
			logger.Warn("web watch check failed; trying again at the next one",
				"watchId", payload.WatchID.String(), "error", err)
		} else if result.Ended {
			return nil
		}

		waitForNextCheck(ctx, signals)
	}

	drain(signals)

	return workflow.NewContinueAsNewError(ctx, WebWatchWorkflow, payload)
}

// waitForNextCheck sleeps until the next scheduled check or a signal, and
// takes every signal already waiting so a burst of them makes one check.
func waitForNextCheck(ctx workflow.Context, signals workflow.ReceiveChannel) {
	timerCtx, cancel := workflow.WithCancel(ctx)
	timer := workflow.NewTimer(timerCtx, checkInterval)

	selector := workflow.NewSelector(ctx)
	selector.AddFuture(timer, func(workflow.Future) {})
	selector.AddReceive(signals, func(channel workflow.ReceiveChannel, _ bool) {
		channel.Receive(ctx, nil)
	})
	selector.Select(ctx)
	cancel()
	drain(signals)
}

// drain takes every signal already waiting: one check answers them all, and
// the next execution checks first anyway.
func drain(signals workflow.ReceiveChannel) {
	for signals.ReceiveAsync(nil) {
		continue
	}
}
