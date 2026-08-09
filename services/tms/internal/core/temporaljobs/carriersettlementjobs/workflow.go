package carriersettlementjobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var carrierSettlementRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    3,
	MaximumInterval:    30 * time.Second,
}

var carrierSettlementActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 15 * time.Minute,
	HeartbeatTimeout:    time.Minute,
	RetryPolicy:         carrierSettlementRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        GenerateCarrierSettlementBatchesWorkflowName,
			Fn:          GenerateCarrierSettlementBatchesWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Auto-generate carrier settlement batches for closed pay periods",
		},
	}
}

func GenerateCarrierSettlementBatchesWorkflow(
	ctx workflow.Context,
) (*GenerateCarrierSettlementBatchesResult, error) {
	ctx = workflow.WithActivityOptions(ctx, carrierSettlementActivityOptions)

	var a *Activities
	result := new(GenerateCarrierSettlementBatchesResult)
	if err := workflow.ExecuteActivity(
		ctx,
		a.GenerateCarrierSettlementBatchesActivity,
	).Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).
			Error("Carrier settlement batch generation workflow failed", "error", err)
		return nil, err
	}

	workflow.GetLogger(ctx).Info("Carrier settlement batch generation workflow completed",
		"organizationsChecked", result.OrganizationsChecked,
		"batchesGenerated", result.BatchesGenerated,
		"failed", result.Failed,
	)
	return result, nil
}
