package shipmentjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
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
	logger := workflow.GetLogger(ctx)
	activityCtx := workflow.WithActivityOptions(ctx, autoDelayActivityOptions)

	var a *Activities
	var tenantsResult *ListShipmentTenantsResult
	if err := workflow.ExecuteActivity(
		activityCtx,
		a.ListAutoDelayShipmentTenantsActivity,
		&ListShipmentTenantsPayload{Limit: temporaljobs.DefaultTenantScanLimit},
	).Get(ctx, &tenantsResult); err != nil {
		logger.Error("Shipment auto delay tenant discovery failed", "error", err)
		return nil, err
	}

	result := &AutoDelayShipmentsResult{
		ShipmentIDs: make([]pulid.ID, 0),
		CompletedAt: workflow.Now(ctx).Unix(),
	}
	result.TenantsScanned = len(tenantsResult.Tenants)

	for _, tenant := range tenantsResult.Tenants {
		payload := &ShipmentTenantWorkPayload{TenantWorkItem: tenant}

		var tenantResult *AutoDelayShipmentsResult
		if err := workflow.ExecuteActivity(
			activityCtx,
			a.AutoDelayTenantShipmentsActivity,
			payload,
		).Get(ctx, &tenantResult); err != nil {
			logger.Error("Shipment auto delay tenant failed",
				"orgId", tenant.OrganizationID.String(),
				"buId", tenant.BusinessUnitID.String(),
				"error", err,
			)
			result.AddFailure(tenant, err)
			continue
		}

		result.AddTenantResult(tenantResult.DelayedCount, 0)
		result.DelayedCount += tenantResult.DelayedCount
		result.ShipmentIDs = append(result.ShipmentIDs, tenantResult.ShipmentIDs...)
	}

	logger.Info("Shipment auto delay workflow completed",
		"tenantsScanned", result.TenantsScanned,
		"tenantsProcessed", result.TenantsProcessed,
		"recordsProcessed", result.DelayedCount,
		"failureCount", result.FailureCount,
	)
	return result, nil
}

func AutoCancelShipmentsWorkflow(ctx workflow.Context) (*AutoCancelShipmentsResult, error) {
	logger := workflow.GetLogger(ctx)
	activityCtx := workflow.WithActivityOptions(ctx, autoCancelActivityOptions)

	var a *Activities
	var tenantsResult *ListShipmentTenantsResult
	if err := workflow.ExecuteActivity(
		activityCtx,
		a.ListAutoCancelShipmentTenantsActivity,
		&ListShipmentTenantsPayload{Limit: temporaljobs.DefaultTenantScanLimit},
	).Get(ctx, &tenantsResult); err != nil {
		logger.Error("Shipment auto cancel tenant discovery failed", "error", err)
		return nil, err
	}

	result := &AutoCancelShipmentsResult{
		ShipmentIDs: make([]pulid.ID, 0),
		CompletedAt: workflow.Now(ctx).Unix(),
	}
	result.TenantsScanned = len(tenantsResult.Tenants)

	for _, tenant := range tenantsResult.Tenants {
		payload := &ShipmentTenantWorkPayload{TenantWorkItem: tenant}

		var tenantResult *AutoCancelShipmentsResult
		if err := workflow.ExecuteActivity(
			activityCtx,
			a.AutoCancelTenantShipmentsActivity,
			payload,
		).Get(ctx, &tenantResult); err != nil {
			logger.Error("Shipment auto cancel tenant failed",
				"orgId", tenant.OrganizationID.String(),
				"buId", tenant.BusinessUnitID.String(),
				"error", err,
			)
			result.AddFailure(tenant, err)
			continue
		}

		result.AddTenantResult(tenantResult.CanceledCount, 0)
		result.CanceledCount += tenantResult.CanceledCount
		result.ShipmentIDs = append(result.ShipmentIDs, tenantResult.ShipmentIDs...)
	}

	logger.Info("Shipment auto cancel workflow completed",
		"tenantsScanned", result.TenantsScanned,
		"tenantsProcessed", result.TenantsProcessed,
		"recordsProcessed", result.CanceledCount,
		"failureCount", result.FailureCount,
	)
	return result, nil
}
