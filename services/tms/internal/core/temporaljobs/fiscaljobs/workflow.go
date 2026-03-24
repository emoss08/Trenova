package fiscaljobs

import (
	"fmt"
	"time"

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

func AutoCloseFiscalPeriodsWorkflow(ctx workflow.Context) error {
	logger := workflow.GetLogger(ctx)

	fetchCtx := workflow.WithActivityOptions(ctx, fetchActivityOptions)

	var a *Activities
	var tenantsResult *GetAutoCloseTenantsResult

	err := workflow.ExecuteActivity(fetchCtx, a.GetAutoCloseTenantsActivity).
		Get(ctx, &tenantsResult)
	if err != nil {
		logger.Error("Failed to get tenants with auto-close enabled", "error", err)
		return err
	}

	if len(tenantsResult.Tenants) == 0 {
		logger.Info("No tenants with auto-close enabled")
		return nil
	}

	closeCtx := workflow.WithActivityOptions(ctx, closeActivityOptions)
	totalClosed := 0
	totalErrors := 0
	activityErrors := 0

	for _, tenant := range tenantsResult.Tenants {
		payload := &AutoClosePeriodsPayload{
			OrganizationID: tenant.OrganizationID,
			BusinessUnitID: tenant.BusinessUnitID,
		}

		var result *AutoClosePeriodsResult
		err = workflow.ExecuteActivity(closeCtx, a.CloseExpiredPeriodsActivity, payload).
			Get(ctx, &result)
		if err != nil {
			logger.Error("Failed to close expired periods for tenant",
				"orgId", tenant.OrganizationID.String(),
				"error", err,
			)
			activityErrors++
			totalErrors++
			continue
		}

		totalClosed += result.ClosedCount
		totalErrors += len(result.Errors)
	}

	logger.Info("Auto-close fiscal periods workflow completed",
		"tenantsProcessed", len(tenantsResult.Tenants),
		"totalClosed", totalClosed,
		"totalErrors", totalErrors,
	)

	if activityErrors > 0 {
		return temporal.NewApplicationError(
			fmt.Sprintf(
				"Workflow completed with %d activity errors out of %d tenants",
				activityErrors,
				len(tenantsResult.Tenants),
			),
			"PartialFailure",
			nil,
		)
	}

	return nil
}

func AutoCreateNextFiscalYearWorkflow(ctx workflow.Context) error {
	logger := workflow.GetLogger(ctx)

	fetchCtx := workflow.WithActivityOptions(ctx, fetchActivityOptions)

	var a *Activities
	var tenantsResult *GetAutoCloseTenantsResult

	err := workflow.ExecuteActivity(fetchCtx, a.GetAutoCloseTenantsActivity).
		Get(ctx, &tenantsResult)
	if err != nil {
		logger.Error("Failed to get tenants", "error", err)
		return err
	}

	if len(tenantsResult.Tenants) == 0 {
		logger.Info("No tenants found")
		return nil
	}

	createCtx := workflow.WithActivityOptions(ctx, createActivityOptions)
	totalCreated := 0
	totalSkipped := 0
	activityErrors := 0

	for _, tenant := range tenantsResult.Tenants {
		payload := &AutoCreateFiscalYearPayload{
			OrganizationID: tenant.OrganizationID,
			BusinessUnitID: tenant.BusinessUnitID,
		}

		var result *AutoCreateFiscalYearResult
		err = workflow.ExecuteActivity(createCtx, a.CheckAndCreateNextFiscalYearActivity, payload).
			Get(ctx, &result)
		if err != nil {
			logger.Error("Failed to check/create next fiscal year",
				"orgId", tenant.OrganizationID.String(),
				"error", err,
			)
			activityErrors++
			continue
		}

		if result.Created {
			totalCreated++
			logger.Info("Created next fiscal year",
				"orgId", tenant.OrganizationID.String(),
				"year", result.FiscalYear,
			)
		} else {
			totalSkipped++
		}
	}

	logger.Info("Auto-create next fiscal year workflow completed",
		"tenantsProcessed", len(tenantsResult.Tenants),
		"created", totalCreated,
		"skipped", totalSkipped,
	)

	if activityErrors > 0 {
		return temporal.NewApplicationError(
			fmt.Sprintf(
				"Workflow completed with %d activity errors out of %d tenants",
				activityErrors,
				len(tenantsResult.Tenants),
			),
			"PartialFailure",
			nil,
		)
	}

	return nil
}
