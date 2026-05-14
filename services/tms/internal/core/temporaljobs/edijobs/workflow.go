package edijobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var approveLoadTenderRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    5,
	MaximumInterval:    time.Minute,
}

var approveLoadTenderActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 10 * time.Minute,
	HeartbeatTimeout:    30 * time.Second,
	RetryPolicy:         approveLoadTenderRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        temporaltype.ApproveLoadTenderTransferWorkflowName,
			Fn:          ApproveLoadTenderTransferWorkflow,
			TaskQueue:   temporaltype.EDITaskQueue,
			Description: "Approve an inbound EDI load tender transfer",
		},
	}
}

func ApproveLoadTenderTransferWorkflow(
	ctx workflow.Context,
	payload *ApproveLoadTenderTransferWorkflowPayload,
) (*ApproveLoadTenderTransferWorkflowResult, error) {
	ctx = workflow.WithActivityOptions(ctx, approveLoadTenderActivityOptions)

	var a *Activities
	result := new(ApproveLoadTenderTransferWorkflowResult)
	if err := workflow.ExecuteActivity(
		ctx,
		a.ApproveLoadTenderTransferActivity,
		payload,
	).Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("EDI load tender approval workflow failed", "error", err)
		return nil, err
	}

	workflow.GetLogger(ctx).Info("EDI load tender approval workflow completed")
	return result, nil
}
