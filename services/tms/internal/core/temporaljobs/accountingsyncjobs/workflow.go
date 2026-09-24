package accountingsyncjobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var healthActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 10 * time.Minute,
	HeartbeatTimeout:    2 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    3,
		MaximumInterval:    time.Minute,
	},
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        CheckAccountingConnectionsWorkflowName,
			Fn:          CheckAccountingConnectionsWorkflow,
			TaskQueue:   temporaltype.IntegrationTaskQueue,
			Description: "Refresh accounting authorizations that are about to expire and check each connection answers",
		},
	}
}

func CheckAccountingConnectionsWorkflow(ctx workflow.Context) (*HealthSweepResult, error) {
	ctx = workflow.WithActivityOptions(ctx, healthActivityOptions)

	var a *Activities
	result := new(HealthSweepResult)
	if err := workflow.ExecuteActivity(ctx, a.CheckAccountingConnectionsActivity).Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("Accounting connection check failed", "error", err)
		return nil, err
	}

	return result, nil
}
