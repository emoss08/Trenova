/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package shipment

import (
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/external/temporaljobs/payloads"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// WorkflowDefinition defines a workflow with its configuration
type WorkflowDefinition struct {
	Name        string
	Fn          any
	TaskQueue   string
	Description string
}

// RegisterWorkflows registers all shipment-related workflows
func RegisterWorkflows() []WorkflowDefinition {
	return []WorkflowDefinition{
		{
			Name:        "DuplicateShipmentWorkflow",
			Fn:          DuplicateShipmentWorkflow,
			TaskQueue:   "shipment-tasks",
			Description: "Duplicates shipments with configurable options",
		},
	}
}

// DuplicateShipmentWorkflow handles the duplication of shipments
func DuplicateShipmentWorkflow(
	ctx workflow.Context,
	payload *payloads.DuplicateShipmentPayload,
) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("starting duplicate shipment workflow",
		"organizationId", payload.OrganizationID.String(),
		"shipmentId", payload.ShipmentID.String(),
		"count", payload.Count,
	)

	// Configure activity options with appropriate timeouts and retry policy
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute, // Allow enough time for bulk operations
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Execute the duplicate shipment activity
	var result DuplicateShipmentResult
	err := workflow.ExecuteActivity(ctx, "ActivityProvider.DuplicateShipmentActivity", payload).
		Get(ctx, &result)
	if err != nil {
		logger.Error("failed to duplicate shipments", "error", err)

		notificationPayload := &payloads.JobCompletionNotificationPayload{
			BasePayload:    payload.BasePayload,
			JobType:        "duplicate_shipment",
			Success:        false,
			Result:         "Failed to duplicate shipments",
			Error:          err.Error(),
			OriginalEntity: payload.ShipmentID.String(),
		}

		notifyCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 10 * time.Second,
			RetryPolicy: &temporal.RetryPolicy{
				MaximumAttempts: 2,
			},
		})

		_ = workflow.
			ExecuteActivity(
				notifyCtx,
				"ActivityProvider.SendJobCompletionNotificationWithService",
				notificationPayload,
			).Get(notifyCtx, nil)

		return fmt.Errorf("failed to duplicate shipments: %w", err)
	}

	logger.Info("shipments duplicated successfully",
		"count", result.ShipmentCount,
		"proNumbers", result.ProNumbers,
	)

	notificationPayload := &payloads.JobCompletionNotificationPayload{
		BasePayload:    payload.BasePayload,
		JobType:        "duplicate_shipment",
		Success:        true,
		Result:         fmt.Sprintf("Successfully duplicated %d shipments", result.ShipmentCount),
		OriginalEntity: payload.ShipmentID.String(),
		Data: map[string]any{
			"shipmentCount": result.ShipmentCount,
			"proNumbers":    result.ProNumbers,
		},
	}

	notifyCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 2,
		},
	})

	if err := workflow.
		ExecuteActivity(notifyCtx, "ActivityProvider.SendJobCompletionNotificationWithService", notificationPayload).
		Get(notifyCtx, nil); err != nil {
		logger.Warn("failed to send success notification", "error", err)
	}

	return nil
}
