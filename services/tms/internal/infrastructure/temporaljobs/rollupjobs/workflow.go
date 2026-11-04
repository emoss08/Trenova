package rollupjobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// refreshOptions let a run drain a large backlog: a heartbeat every few seconds shows it is
// alive, and a failed run is retried a few times before the next scheduled run takes over.
var refreshOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 30 * time.Minute,
	HeartbeatTimeout:    time.Minute,
	Summary:             "Recompute the rollup days writes marked dirty",
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    3,
	},
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{{
		Name:        RefreshWorkflowName,
		Fn:          RefreshRollupsWorkflow,
		TaskQueue:   temporaltype.TaskQueueSystem.String(),
		Description: "Recompute the rollup days writes marked dirty",
	}}
}

// RefreshRollupsWorkflow runs one refresh of every rollup.
func RefreshRollupsWorkflow(ctx workflow.Context) (*RefreshResult, error) {
	var a *Activities
	var result RefreshResult
	err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, refreshOptions),
		a.RefreshRollupsActivity,
	).Get(ctx, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
