package fiscaljobs

import (
	"time"

	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var defaultRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    3,
	MaximumInterval:    30 * time.Second,
	NonRetryableErrorTypes: []string{
		temporaltype.ErrorTypeInvalidInput.String(),
		temporaltype.ErrorTypeDataIntegrity.String(),
	},
}

var fetchActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 1 * time.Minute,
	RetryPolicy:         defaultRetryPolicy,
}

var closeActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 2 * time.Minute,
	HeartbeatTimeout:    30 * time.Second,
	RetryPolicy:         defaultRetryPolicy,
}

var createActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 2 * time.Minute,
	HeartbeatTimeout:    30 * time.Second,
	RetryPolicy:         defaultRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        "AutoCloseFiscalPeriodsWorkflow",
			Fn:          AutoCloseFiscalPeriodsWorkflow,
			TaskQueue:   temporaltype.FiscalTaskQueue,
			Description: "Automatically close expired fiscal periods for tenants with auto-close enabled",
		},
		{
			Name:        "AutoCreateNextFiscalYearWorkflow",
			Fn:          AutoCreateNextFiscalYearWorkflow,
			TaskQueue:   temporaltype.FiscalTaskQueue,
			Description: "Automatically create next fiscal year when current one is nearing end",
		},
	}
}

func AutoCloseFiscalPeriodsWorkflow(ctx workflow.Context) (*FiscalTenantRunResult, error) {
	logger := workflow.GetLogger(ctx)

	fetchCtx := workflow.WithActivityOptions(ctx, fetchActivityOptions)

	var a *Activities
	var tenantsResult *GetAutoCloseTenantsResult

	err := workflow.ExecuteActivity(fetchCtx, a.GetAutoCloseTenantsActivity).
		Get(ctx, &tenantsResult)
	if err != nil {
		logger.Error("Failed to get tenants with auto-close enabled", "error", err)
		return nil, err
	}

	summary := &FiscalTenantRunResult{}
	if len(tenantsResult.Tenants) == 0 {
		logger.Info("No tenants with auto-close enabled")
		return summary, nil
	}
	summary.TenantsScanned = len(tenantsResult.Tenants)

	closeCtx := workflow.WithActivityOptions(ctx, closeActivityOptions)
	totalErrors := 0

	for _, tenant := range tenantsResult.Tenants {
		item := temporaljobs.TenantWorkItem{
			OrganizationID: tenant.OrganizationID,
			BusinessUnitID: tenant.BusinessUnitID,
		}
		payload := &AutoClosePeriodsPayload{
			OrganizationID: tenant.OrganizationID,
			BusinessUnitID: tenant.BusinessUnitID,
		}

		var closeResult *AutoClosePeriodsResult
		err = workflow.ExecuteActivity(closeCtx, a.CloseExpiredPeriodsActivity, payload).
			Get(ctx, &closeResult)
		if err != nil {
			logger.Error("Failed to close expired periods for tenant",
				"orgId", tenant.OrganizationID.String(),
				"error", err,
			)
			summary.AddFailure(item, err)
			totalErrors++
			continue
		}

		summary.AddTenantResult(closeResult.ClosedCount, 0)
		summary.Closed += closeResult.ClosedCount
		totalErrors += len(closeResult.Errors)
	}

	logger.Info("Auto-close fiscal periods workflow completed",
		"tenantsProcessed", len(tenantsResult.Tenants),
		"totalClosed", summary.Closed,
		"totalErrors", totalErrors,
		"failureCount", summary.FailureCount,
	)

	return summary, nil
}

func AutoCreateNextFiscalYearWorkflow(ctx workflow.Context) (*FiscalTenantRunResult, error) {
	logger := workflow.GetLogger(ctx)

	fetchCtx := workflow.WithActivityOptions(ctx, fetchActivityOptions)

	var a *Activities
	var tenantsResult *GetAutoCloseTenantsResult

	err := workflow.ExecuteActivity(fetchCtx, a.GetAutoCloseTenantsActivity).
		Get(ctx, &tenantsResult)
	if err != nil {
		logger.Error("Failed to get tenants", "error", err)
		return nil, err
	}

	summary := &FiscalTenantRunResult{}
	if len(tenantsResult.Tenants) == 0 {
		logger.Info("No tenants found")
		return summary, nil
	}
	summary.TenantsScanned = len(tenantsResult.Tenants)

	createCtx := workflow.WithActivityOptions(ctx, createActivityOptions)
	totalSkipped := 0

	for _, tenant := range tenantsResult.Tenants {
		item := temporaljobs.TenantWorkItem{
			OrganizationID: tenant.OrganizationID,
			BusinessUnitID: tenant.BusinessUnitID,
		}
		payload := &AutoCreateFiscalYearPayload{
			OrganizationID: tenant.OrganizationID,
			BusinessUnitID: tenant.BusinessUnitID,
		}

		var createResult *AutoCreateFiscalYearResult
		err = workflow.ExecuteActivity(createCtx, a.CheckAndCreateNextFiscalYearActivity, payload).
			Get(ctx, &createResult)
		if err != nil {
			logger.Error("Failed to check/create next fiscal year",
				"orgId", tenant.OrganizationID.String(),
				"error", err,
			)
			summary.AddFailure(item, err)
			continue
		}

		summary.AddTenantResult(0, 0)
		if createResult.Created {
			summary.Created++
			logger.Info("Created next fiscal year",
				"orgId", tenant.OrganizationID.String(),
				"year", createResult.FiscalYear,
			)
		} else {
			totalSkipped++
		}
	}

	logger.Info("Auto-create next fiscal year workflow completed",
		"tenantsProcessed", len(tenantsResult.Tenants),
		"created", summary.Created,
		"skipped", totalSkipped,
		"failureCount", summary.FailureCount,
	)

	return summary, nil
}
