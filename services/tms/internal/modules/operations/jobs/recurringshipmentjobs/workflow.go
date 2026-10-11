package recurringshipmentjobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var dispatchRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    3,
	MaximumInterval:    30 * time.Second,
}

var dispatchActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 10 * time.Minute,
	HeartbeatTimeout:    time.Minute,
	RetryPolicy:         dispatchRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        DispatchDueRecurringShipmentsWorkflowName,
			Fn:          DispatchDueRecurringShipmentsWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Materialize shipments for due recurring shipment series",
		},
	}
}

func DispatchDueRecurringShipmentsWorkflow(
	ctx workflow.Context,
) (*DispatchDueRecurringShipmentsResult, error) {
	ctx = workflow.WithActivityOptions(ctx, dispatchActivityOptions)

	var a *Activities
	result := new(DispatchDueRecurringShipmentsResult)
	if err := workflow.ExecuteActivity(
		ctx,
		a.DispatchDueRecurringShipmentsActivity,
	).Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("Recurring shipment dispatch workflow failed", "error", err)
		return nil, err
	}

	workflow.GetLogger(ctx).Info("Recurring shipment dispatch workflow completed",
		"dispatched", result.Dispatched,
		"skipped", result.Skipped,
		"failed", result.Failed,
	)

	return result, nil
}
