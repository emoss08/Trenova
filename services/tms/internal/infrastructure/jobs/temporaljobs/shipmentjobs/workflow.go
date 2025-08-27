package shipmentjobs

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func DuplicateShipmentWorkflow(
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

	return duplicateShipment(ctx, payload)
}

func duplicateShipment(
	ctx workflow.Context,
	payload *DuplicateShipmentPayload,
) (result *DuplicateShipmentResult, err error) {
	so := &workflow.SessionOptions{
		CreationTimeout:  10 * time.Second,
		ExecutionTimeout: 10 * time.Second,
	}

	sessionCtx, err := workflow.CreateSession(ctx, so)
	if err != nil {
		return nil, err
	}
	defer workflow.CompleteSession(sessionCtx)

	var a *ShipmentJobsActivities
	err = workflow.
		ExecuteActivity(sessionCtx, a.DuplicateShipmentActivity, &payload).
		Get(sessionCtx, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
