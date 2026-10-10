package telematicsjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

//nolint:gosec // Temporal workflow names, not credentials.
const (
	TelematicsPollWorkflowName      = "TelematicsPollWorkflow"
	TelematicsSweepWorkflowName     = "TelematicsSweepWorkflow"
	TelematicsRetentionWorkflowName = "TelematicsRetentionWorkflow"
)

var telematicsRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    2 * time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    2,
	MaximumInterval:    15 * time.Second,
}

var telematicsActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 5 * time.Minute,
	HeartbeatTimeout:    time.Minute,
	RetryPolicy:         telematicsRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        TelematicsPollWorkflowName,
			Fn:          TelematicsPollWorkflow,
			TaskQueue:   temporaltype.IntegrationTaskQueue,
			Description: "Poll Samsara vehicle positions and HOS clocks for all enabled tenants",
		},
		{
			Name:        TelematicsSweepWorkflowName,
			Fn:          TelematicsSweepWorkflow,
			TaskQueue:   temporaltype.IntegrationTaskQueue,
			Description: "Sync telematics vehicle mappings, driver rulesets, and HOS violations for all enabled tenants",
		},
		{
			Name:        TelematicsRetentionWorkflowName,
			Fn:          TelematicsRetentionWorkflow,
			TaskQueue:   temporaltype.IntegrationTaskQueue,
			Description: "Prune aged telematics events and HOS violation history",
		},
	}
}

func TelematicsRetentionWorkflow(ctx workflow.Context) (*RetentionResult, error) {
	activityCtx := workflow.WithActivityOptions(ctx, telematicsActivityOptions)

	var a *Activities
	result := new(RetentionResult)
	if err := workflow.ExecuteActivity(
		activityCtx,
		a.CleanupTelematicsActivity,
	).Get(ctx, result); err != nil {
		return nil, err
	}

	workflow.GetLogger(ctx).Info("Telematics retention workflow completed",
		"rowsDeleted", result.RowsDeleted,
	)
	return result, nil
}

func TelematicsPollWorkflow(ctx workflow.Context) (*temporaljobs.TenantRunResult, error) {
	return runTenantFanOut(ctx, "Telematics poll", func(
		activityCtx workflow.Context,
		a *Activities,
		tenant temporaljobs.TenantWorkItem,
	) (int, error) {
		var pollResult *PollTenantResult
		err := workflow.ExecuteActivity(
			activityCtx,
			a.PollTenantTelematicsActivity,
			&TenantPayload{TenantWorkItem: tenant},
		).Get(activityCtx, &pollResult)
		if err != nil {
			return 0, err
		}
		return pollResult.PositionsUpserted + pollResult.HOSStatesUpserted, nil
	})
}

func TelematicsSweepWorkflow(ctx workflow.Context) (*temporaljobs.TenantRunResult, error) {
	return runTenantFanOut(ctx, "Telematics sweep", func(
		activityCtx workflow.Context,
		a *Activities,
		tenant temporaljobs.TenantWorkItem,
	) (int, error) {
		var sweepResult *SweepTenantResult
		err := workflow.ExecuteActivity(
			activityCtx,
			a.SweepTenantTelematicsActivity,
			&TenantPayload{TenantWorkItem: tenant},
		).Get(activityCtx, &sweepResult)
		if err != nil {
			return 0, err
		}
		return sweepResult.VehiclesMatched + sweepResult.ViolationsUpserted, nil
	})
}

func runTenantFanOut(
	ctx workflow.Context,
	label string,
	runTenant func(workflow.Context, *Activities, temporaljobs.TenantWorkItem) (int, error),
) (*temporaljobs.TenantRunResult, error) {
	activityCtx := workflow.WithActivityOptions(ctx, telematicsActivityOptions)
	logger := workflow.GetLogger(ctx)

	var a *Activities
	var tenantsResult *ListTelematicsTenantsResult
	if err := workflow.ExecuteActivity(
		activityCtx,
		a.ListTelematicsTenantsActivity,
		&ListTelematicsTenantsPayload{Limit: temporaljobs.DefaultTenantScanLimit},
	).Get(ctx, &tenantsResult); err != nil {
		return nil, err
	}

	result := new(temporaljobs.TenantRunResult)
	result.TenantsScanned = len(tenantsResult.Tenants)
	for _, tenant := range tenantsResult.Tenants {
		processed, err := runTenant(activityCtx, a, tenant)
		if err != nil {
			logger.Error(label+" tenant failed",
				"orgId", tenant.OrganizationID.String(),
				"buId", tenant.BusinessUnitID.String(),
				"error", err,
			)
			result.AddFailure(tenant, err)
			continue
		}
		result.AddTenantResult(processed, 0)
	}

	logger.Info(label+" workflow completed",
		"tenantsScanned", result.TenantsScanned,
		"tenantsProcessed", result.TenantsProcessed,
		"recordsProcessed", result.RecordsProcessed,
		"failureCount", result.FailureCount,
	)
	return result, nil
}
