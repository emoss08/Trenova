package insightjobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var insightRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    3,
	MaximumInterval:    30 * time.Second,
}

// The timeout is generous because the sweep walks every organization and each
// one runs several month-wide aggregates plus a model call. A refresh that gets
// cut off halfway leaves some tenants updated and others not, which the next run
// corrects, but it is better not to need correcting.
var insightActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 30 * time.Minute,
	HeartbeatTimeout:    2 * time.Minute,
	RetryPolicy:         insightRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        InsightRefreshWorkflowName,
			Fn:          InsightRefreshWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Recompute operational insights for every organization",
		},
	}
}

func InsightRefreshWorkflow(ctx workflow.Context) (*InsightRefreshResult, error) {
	ctx = workflow.WithActivityOptions(ctx, insightActivityOptions)

	var a *Activities
	result := new(InsightRefreshResult)
	if err := workflow.ExecuteActivity(ctx, a.RefreshInsightsActivity).Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("Insight refresh workflow failed", "error", err)

		return nil, err
	}

	return result, nil
}
