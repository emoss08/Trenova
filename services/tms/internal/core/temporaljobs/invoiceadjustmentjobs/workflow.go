package invoiceadjustmentjobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var batchItemRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2,
	MaximumAttempts:    5,
	MaximumInterval:    30 * time.Second,
}

var batchItemActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 5 * time.Minute,
	HeartbeatTimeout:    30 * time.Second,
	RetryPolicy:         batchItemRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        InvoiceAdjustmentBatchWorkflowName,
			Fn:          InvoiceAdjustmentBatchWorkflow,
			TaskQueue:   temporaltype.TaskQueueBilling.String(),
			Description: "Process invoice adjustment batches with per-item retries and progress tracking",
		},
	}
}

func InvoiceAdjustmentBatchWorkflow(
	ctx workflow.Context,
	payload *BatchWorkflowPayload,
) error {
	ctx = workflow.WithActivityOptions(ctx, batchItemActivityOptions)

	var a *Activities
	for _, itemID := range payload.ItemIDs {
		itemPayload := &ProcessBatchItemPayload{
			BatchID:        payload.BatchID,
			ItemID:         itemID,
			OrganizationID: payload.OrganizationID,
			BusinessUnitID: payload.BusinessUnitID,
			UserID:         payload.UserID,
			PrincipalType:  payload.PrincipalType,
			PrincipalID:    payload.PrincipalID,
			APIKeyID:       payload.APIKeyID,
		}
		if err := workflow.ExecuteActivity(ctx, a.ProcessBatchItemActivity, itemPayload).Get(ctx, nil); err != nil {
			workflow.GetLogger(ctx).Error("Invoice adjustment batch item failed", "batchId", payload.BatchID.String(), "itemId", itemID.String(), "error", err)
			return err
		}
	}

	return nil
}
