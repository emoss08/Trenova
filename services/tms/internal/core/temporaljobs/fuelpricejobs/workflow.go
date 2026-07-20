package fuelpricejobs

import (
	"time"

	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const RefreshFuelPricesWorkflowName = "RefreshFuelPricesWorkflow"

var refreshFuelPricesRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    2 * time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    3,
	MaximumInterval:    30 * time.Second,
}

var refreshFuelPricesActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 10 * time.Minute,
	HeartbeatTimeout:    30 * time.Second,
	RetryPolicy:         refreshFuelPricesRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        RefreshFuelPricesWorkflowName,
			Fn:          RefreshFuelPricesWorkflow,
			TaskQueue:   temporaltype.IntegrationTaskQueue,
			Description: "Ingest weekly DOE/EIA diesel prices and re-rate fallback shipments for all enabled tenants",
		},
	}
}

func RefreshFuelPricesWorkflow(ctx workflow.Context) (*RefreshFuelPricesResult, error) {
	activityCtx := workflow.WithActivityOptions(ctx, refreshFuelPricesActivityOptions)
	logger := workflow.GetLogger(ctx)

	var a *Activities
	var tenantsResult *ListFuelPriceTenantsResult
	if err := workflow.ExecuteActivity(
		activityCtx,
		a.ListFuelPriceTenantsActivity,
		&ListFuelPriceTenantsPayload{Limit: temporaljobs.DefaultTenantScanLimit},
	).Get(ctx, &tenantsResult); err != nil {
		return nil, err
	}

	result := &RefreshFuelPricesResult{}
	result.TenantsScanned = len(tenantsResult.Tenants)
	for _, tenant := range tenantsResult.Tenants {
		var refreshResult *RefreshFuelPricesTenantResult
		if err := workflow.ExecuteActivity(
			activityCtx,
			a.RefreshFuelPricesForTenantActivity,
			&RefreshFuelPricesTenantPayload{TenantWorkItem: tenant},
		).Get(ctx, &refreshResult); err != nil {
			logger.Error("Fuel price refresh tenant failed",
				"orgId", tenant.OrganizationID.String(),
				"buId", tenant.BusinessUnitID.String(),
				"error", err,
			)
			result.AddFailure(tenant, err)
			continue
		}

		result.AddTenantResult(1, 0)
		result.NewRows += refreshResult.NewRows

		if refreshResult.NewRows == 0 {
			continue
		}

		var reRateResult *ReRateFallbackShipmentsResult
		if err := workflow.ExecuteActivity(
			activityCtx,
			a.ReRateFallbackShipmentsActivity,
			&ReRateFallbackShipmentsPayload{TenantWorkItem: tenant},
		).Get(ctx, &reRateResult); err != nil {
			logger.Error("Fuel surcharge fallback re-rate failed",
				"orgId", tenant.OrganizationID.String(),
				"error", err,
			)
			continue
		}

		result.ShipmentsReRated += reRateResult.ShipmentsReRated
	}

	logger.Info("Fuel price refresh workflow completed",
		"tenantsScanned", result.TenantsScanned,
		"tenantsProcessed", result.TenantsProcessed,
		"failureCount", result.FailureCount,
		"newRows", result.NewRows,
		"shipmentsReRated", result.ShipmentsReRated,
	)

	return result, nil
}
