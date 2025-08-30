/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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
		StartToCloseTimeout: 2 * time.Minute,  // Allow up to 2 minutes for duplication (up to 100 shipments)
		HeartbeatTimeout:    10 * time.Second, // Heartbeat every 10 seconds
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
	// Configure activity options for bulk cancellation across multiple organizations
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,  // Allow up to 5 minutes for processing all organizations
		HeartbeatTimeout:    15 * time.Second, // Heartbeat every 15 seconds during bulk operations
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
