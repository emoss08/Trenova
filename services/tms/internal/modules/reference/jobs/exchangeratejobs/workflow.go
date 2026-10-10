package exchangeratejobs

import (
	"time"

	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const RefreshExchangeRatesWorkflowName = "RefreshExchangeRatesWorkflow"

var refreshExchangeRatesRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    2 * time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    3,
	MaximumInterval:    30 * time.Second,
}

var refreshExchangeRatesActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 10 * time.Minute,
	HeartbeatTimeout:    30 * time.Second,
	RetryPolicy:         refreshExchangeRatesRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        RefreshExchangeRatesWorkflowName,
			Fn:          RefreshExchangeRatesWorkflow,
			TaskQueue:   temporaltype.IntegrationTaskQueue,
			Description: "Refresh cached exchange rates from OANDA for all enabled tenants",
		},
	}
}

func RefreshExchangeRatesWorkflow(ctx workflow.Context) (*RefreshExchangeRatesResult, error) {
	activityCtx := workflow.WithActivityOptions(ctx, refreshExchangeRatesActivityOptions)
	logger := workflow.GetLogger(ctx)

	var a *Activities
	var tenantsResult *ListExchangeRateTenantsResult
	if err := workflow.ExecuteActivity(
		activityCtx,
		a.ListExchangeRateTenantsActivity,
		&ListExchangeRateTenantsPayload{Limit: temporaljobs.DefaultTenantScanLimit},
	).Get(ctx, &tenantsResult); err != nil {
		return nil, err
	}

	result := &RefreshExchangeRatesResult{}
	result.TenantsScanned = len(tenantsResult.Tenants)
	for _, tenant := range tenantsResult.Tenants {
		payload := &RefreshExchangeRateTenantPayload{
			TenantWorkItem: tenant,
		}
		if err := workflow.ExecuteActivity(
			activityCtx,
			a.RefreshExchangeRatesForTenantActivity,
			payload,
		).Get(ctx, nil); err != nil {
			logger.Error("Exchange rate refresh tenant failed",
				"orgId", tenant.OrganizationID.String(),
				"buId", tenant.BusinessUnitID.String(),
				"error", err,
			)
			result.AddFailure(tenant, err)
			continue
		}

		result.AddTenantResult(1, 0)
	}

	logger.Info("Exchange rate refresh workflow completed",
		"tenantsScanned", result.TenantsScanned,
		"tenantsProcessed", result.TenantsProcessed,
		"failureCount", result.FailureCount,
	)

	return result, nil
}
