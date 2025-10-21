package shipmentjobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	ShipmentTaskQueue = "shipment-queue"
)

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        "BulkDuplicateShipmentWorkflow",
			Fn:          SendBulkDuplicateShipmentWorkflow,
			TaskQueue:   temporaltype.ShipmentTaskQueue,
			Description: "Bulk duplicate shipments",
		},
	}
}

func SendBulkDuplicateShipmentWorkflow(
	ctx workflow.Context,
	payload *DuplicateShipmentPayload,
) (*DuplicateShipmentResult, error) {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		HeartbeatTimeout:    2 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
			MaximumInterval:    time.Minute,
		},
	}

	ctx = workflow.WithActivityOptions(ctx, ao)
	so := &workflow.SessionOptions{
		CreationTimeout:  10 * time.Second,
		ExecutionTimeout: 10 * time.Second,
	}

	sessionCtx, err := workflow.CreateSession(ctx, so)
	if err != nil {
		return nil, err
	}
	defer workflow.CompleteSession(sessionCtx)

	var a *Activities
	var result DuplicateShipmentResult
	err = workflow.
		ExecuteActivity(sessionCtx, a.BulkDuplicateShipmentActivity, &payload).
		Get(sessionCtx, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
