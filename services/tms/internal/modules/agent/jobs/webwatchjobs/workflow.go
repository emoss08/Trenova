package webwatchjobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var sweepOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 15 * time.Minute,
	HeartbeatTimeout:    2 * checkTimeout,
	Summary:             "Check the web watches that are due and hand on what they found",
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    3,
	},
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        SweepWorkflowName,
			Fn:          CheckDueWebWatchesWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Check the web watches whose check is due and hand on what they found",
		},
	}
}

// CheckDueWebWatchesWorkflow is the timer behind every web watch, run each
// minute.
func CheckDueWebWatchesWorkflow(ctx workflow.Context) (*SweepResult, error) {
	var a *Activities
	ctx = workflow.WithActivityOptions(ctx, sweepOptions)

	var result SweepResult
	if err := workflow.ExecuteActivity(ctx, a.CheckDueWebWatchesActivity).
		Get(ctx, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
