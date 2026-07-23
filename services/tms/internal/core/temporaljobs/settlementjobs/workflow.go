package settlementjobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var settlementRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    3,
	MaximumInterval:    30 * time.Second,
}

var settlementActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 15 * time.Minute,
	HeartbeatTimeout:    time.Minute,
	RetryPolicy:         settlementRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        GenerateSettlementBatchesWorkflowName,
			Fn:          GenerateSettlementBatchesWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Auto-generate driver settlement batches for closed pay periods",
		},
		{
			Name:        AccrueEscrowInterestWorkflowName,
			Fn:          AccrueEscrowInterestWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Accrue quarterly escrow interest per 49 CFR 376.12(k)",
		},
	}
}

func GenerateSettlementBatchesWorkflow(
	ctx workflow.Context,
) (*GenerateSettlementBatchesResult, error) {
	ctx = workflow.WithActivityOptions(ctx, settlementActivityOptions)

	var a *Activities
	result := new(GenerateSettlementBatchesResult)
	if err := workflow.ExecuteActivity(
		ctx,
		a.GenerateSettlementBatchesActivity,
	).Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("Settlement batch generation workflow failed", "error", err)
		return nil, err
	}

	workflow.GetLogger(ctx).Info("Settlement batch generation workflow completed",
		"organizationsChecked", result.OrganizationsChecked,
		"batchesGenerated", result.BatchesGenerated,
		"failed", result.Failed,
	)
	return result, nil
}

func AccrueEscrowInterestWorkflow(
	ctx workflow.Context,
) (*AccrueEscrowInterestResult, error) {
	ctx = workflow.WithActivityOptions(ctx, settlementActivityOptions)

	var a *Activities
	result := new(AccrueEscrowInterestResult)
	if err := workflow.ExecuteActivity(
		ctx,
		a.AccrueEscrowInterestActivity,
	).Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("Escrow interest accrual workflow failed", "error", err)
		return nil, err
	}

	workflow.GetLogger(ctx).Info("Escrow interest accrual workflow completed",
		"accountsAccrued", result.AccountsAccrued,
		"failed", result.Failed,
	)
	return result, nil
}
