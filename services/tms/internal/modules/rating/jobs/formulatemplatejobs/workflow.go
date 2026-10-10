package formulatemplatejobs

import (
	"time"

	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const ExpireStaleSubmissionsWorkflowName = "ExpireStaleFormulaSubmissionsWorkflow"

var expireStaleSubmissionsRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    2 * time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    3,
}

var expireStaleSubmissionsActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 5 * time.Minute,
	RetryPolicy:         expireStaleSubmissionsRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        ExpireStaleSubmissionsWorkflowName,
			Fn:          ExpireStaleSubmissionsWorkflow,
			TaskQueue:   string(temporaltype.TaskQueueSystem),
			Description: "Return formula templates that waited in review past the expiry to draft",
		},
	}
}

func ExpireStaleSubmissionsWorkflow(ctx workflow.Context) (*ExpireStaleSubmissionsResult, error) {
	activityCtx := workflow.WithActivityOptions(ctx, expireStaleSubmissionsActivityOptions)

	var a *Activities
	var result *ExpireStaleSubmissionsResult
	if err := workflow.ExecuteActivity(
		activityCtx,
		a.ExpireStaleSubmissionsActivity,
		&ExpireStaleSubmissionsPayload{Limit: temporaljobs.DefaultTenantScanLimit},
	).Get(ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}
