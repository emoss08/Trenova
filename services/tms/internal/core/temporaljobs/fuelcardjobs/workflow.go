package fuelcardjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const FuelCardSyncWorkflowName = "FuelCardSyncWorkflow"

var fuelCardRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    5 * time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    2,
	MaximumInterval:    time.Minute,
}

// A run reads files off a remote server and posts what resolves, so it is given
// room to finish. The watermark only advances on success, which is what makes a
// timed-out run safe to repeat.
var fuelCardActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 15 * time.Minute,
	HeartbeatTimeout:    2 * time.Minute,
	RetryPolicy:         fuelCardRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        FuelCardSyncWorkflowName,
			Fn:          FuelCardSyncWorkflow,
			TaskQueue:   temporaltype.IntegrationTaskQueue,
			Description: "Read fuel card transactions for all connected tenants and post the ones that resolve",
		},
	}
}

func FuelCardSyncWorkflow(ctx workflow.Context) (*temporaljobs.TenantRunResult, error) {
	activityCtx := workflow.WithActivityOptions(ctx, fuelCardActivityOptions)
	logger := workflow.GetLogger(ctx)

	var a *Activities
	var tenantsResult *ListFuelCardTenantsResult
	if err := workflow.ExecuteActivity(
		activityCtx,
		a.ListFuelCardTenantsActivity,
		&ListFuelCardTenantsPayload{Limit: temporaljobs.DefaultTenantScanLimit},
	).Get(ctx, &tenantsResult); err != nil {
		return nil, err
	}

	result := new(temporaljobs.TenantRunResult)
	result.TenantsScanned = len(tenantsResult.Tenants)

	for _, tenant := range tenantsResult.Tenants {
		var syncResult *SyncTenantResult
		err := workflow.ExecuteActivity(
			activityCtx,
			a.SyncTenantFuelCardsActivity,
			&TenantPayload{TenantWorkItem: tenant},
		).Get(activityCtx, &syncResult)
		if err != nil {
			// One tenant's broken connection must not stop the rest; the failure
			// is recorded on that tenant's feed state for somebody to see.
			logger.Error("Fuel card sync tenant failed",
				"orgId", tenant.OrganizationID.String(),
				"buId", tenant.BusinessUnitID.String(),
				"error", err,
			)
			result.AddFailure(tenant, err)
			continue
		}

		result.AddTenantResult(syncResult.Committed, syncResult.Queued)
	}

	logger.Info("Fuel card sync workflow completed",
		"tenantsScanned", result.TenantsScanned,
		"tenantsProcessed", result.TenantsProcessed,
		"recordsProcessed", result.RecordsProcessed,
		"failureCount", result.FailureCount,
	)

	return result, nil
}
