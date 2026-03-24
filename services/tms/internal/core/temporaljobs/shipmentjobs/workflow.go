package shipmentjobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var bulkDuplicateRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    3,
	MaximumInterval:    30 * time.Second,
}

var bulkDuplicateActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 10 * time.Minute,
	HeartbeatTimeout:    30 * time.Second,
	RetryPolicy:         bulkDuplicateRetryPolicy,
}

var autoDelayActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 5 * time.Minute,
	HeartbeatTimeout:    30 * time.Second,
	RetryPolicy:         bulkDuplicateRetryPolicy,
}

var autoCancelActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 10 * time.Minute,
	HeartbeatTimeout:    30 * time.Second,
	RetryPolicy:         bulkDuplicateRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        BulkDuplicateShipmentsWorkflowName,
			Fn:          BulkDuplicateShipmentsWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Duplicate a shipment and its move/stop graph in bulk",
		},
		{
			Name:        AutoDelayShipmentsWorkflowName,
			Fn:          AutoDelayShipmentsWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Mark eligible shipments delayed across tenants",
		},
		{
			Name:        AutoCancelShipmentsWorkflowName,
			Fn:          AutoCancelShipmentsWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Automatically cancel eligible shipments across tenants",
		},
	}
}

func BulkDuplicateShipmentsWorkflow(
	ctx workflow.Context,
	payload *BulkDuplicateShipmentsPayload,
) (*BulkDuplicateShipmentsResult, error) {
	ctx = workflow.WithActivityOptions(ctx, bulkDuplicateActivityOptions)

	var a *Activities
	result := new(BulkDuplicateShipmentsResult)
	if err := workflow.ExecuteActivity(
		ctx,
		a.BulkDuplicateShipmentsActivity,
		payload,
	).Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("Shipment bulk duplicate workflow failed", "error", err)
		return nil, err
	}

	workflow.GetLogger(ctx).Info("Shipment bulk duplicate workflow completed")
	return result, nil
}

func AutoDelayShipmentsWorkflow(ctx workflow.Context) (*AutoDelayShipmentsResult, error) {
	ctx = workflow.WithActivityOptions(ctx, autoDelayActivityOptions)

	var a *Activities
	result := new(AutoDelayShipmentsResult)
	if err := workflow.ExecuteActivity(
		ctx,
		a.AutoDelayShipmentsActivity,
	).Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("Shipment auto delay workflow failed", "error", err)
		return nil, err
	}

	workflow.GetLogger(ctx).Info("Shipment auto delay workflow completed")
	return result, nil
}

func AutoCancelShipmentsWorkflow(ctx workflow.Context) (*AutoCancelShipmentsResult, error) {
	ctx = workflow.WithActivityOptions(ctx, autoCancelActivityOptions)

	var a *Activities
	result := new(AutoCancelShipmentsResult)
	if err := workflow.ExecuteActivity(
		ctx,
		a.AutoCancelShipmentsActivity,
	).Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("Shipment auto cancel workflow failed", "error", err)
		return nil, err
	}

	workflow.GetLogger(ctx).Info("Shipment auto cancel workflow completed")
	return result, nil
}
