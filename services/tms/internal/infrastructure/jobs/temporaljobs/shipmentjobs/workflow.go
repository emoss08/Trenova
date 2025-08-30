package shipmentjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/jobs/temporaljobs/notificationjobs"
	"github.com/emoss08/trenova/pkg/types/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        "DuplicateShipmentWorkflow",
			Fn:          DuplicateShipmentWorkflow,
			TaskQueue:   temporaltype.ShipmentTaskQueue,
			Description: "Duplicate a shipment",
		},
		{
			Name:        "CancelShipmentsByCreatedAtWorkflow",
			Fn:          CancelShipmentsByCreatedAtWorkflow,
			TaskQueue:   temporaltype.ShipmentTaskQueue,
			Description: "Cancel shipments by created at",
		},
	}
}

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
			NonRetryableErrorTypes: []string{
				temporaltype.ErrorTypeInvalidInput.String(),
				temporaltype.ErrorTypePermissionDenied.String(),
				temporaltype.ErrorTypeResourceNotFound.String(),
			},
		},
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	result, err := duplicateShipment(ctx, payload)
	if err != nil {
		return nil, err
	}

	cwo := workflow.ChildWorkflowOptions{
		TaskQueue: temporaltype.NotificationTaskQueue,
	}
	childCtx := workflow.WithChildOptions(ctx, cwo)
	err = workflow.ExecuteChildWorkflow(
		childCtx,
		notificationjobs.SendJobCompleteNotificationWorkflow,
		&notificationjobs.SendNotificationPayload{
			UserID:         payload.UserID,
			OrganizationID: payload.OrganizationID,
			BusinessUnitID: payload.BusinessUnitID,
			JobID:          result.JobID,
			JobType:        "duplicate_shipment",
			Success:        true,
			Result:         result.Result,
			Data:           result.Data,
		},
	).Get(ctx, nil)
	if err != nil {
		return nil, err
	}

	return result, nil
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

	var a *Activities
	err = workflow.
		ExecuteActivity(sessionCtx, a.DuplicateShipmentActivity, &payload).
		Get(sessionCtx, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func CancelShipmentsByCreatedAtWorkflow(
	ctx workflow.Context,
) (*CancelShipmentsByCreatedAtResult, error) {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second, // Longer timeout for bulk operations
		HeartbeatTimeout:    5 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
			MaximumInterval:    time.Minute,
		},
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	var a *Activities
	var result *CancelShipmentsByCreatedAtResult

	err := workflow.ExecuteActivity(ctx, a.CancelShipmentsByCreatedAtActivity).Get(ctx, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}