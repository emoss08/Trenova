package samsarajobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var samsaraSyncRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    3,
	MaximumInterval:    30 * time.Second,
}

var samsaraSyncActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 5 * time.Minute,
	HeartbeatTimeout:    30 * time.Second,
	RetryPolicy:         samsaraSyncRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        SyncWorkersToSamsaraWorkflowName,
			Fn:          SyncWorkersToSamsaraWorkflow,
			TaskQueue:   temporaltype.IntegrationTaskQueue,
			Description: "Synchronize workers from TMS to Samsara",
		},
	}
}

func SyncWorkersToSamsaraWorkflow(
	ctx workflow.Context,
	payload *WorkersSyncWorkflowPayload,
) (*WorkersSyncWorkflowResult, error) {
	ctx = workflow.WithActivityOptions(ctx, samsaraSyncActivityOptions)

	var a *Activities
	result := new(WorkersSyncWorkflowResult)
	if err := workflow.ExecuteActivity(
		ctx,
		a.SyncWorkersToSamsaraActivity,
		payload,
	).Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("Samsara worker sync workflow failed", "error", err)
		return nil, err
	}

	workflow.GetLogger(ctx).Info("Samsara worker sync workflow completed")
	return result, nil
}
