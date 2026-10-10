package weatheralertjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	PollNWSAlertsWorkflowName = "PollNWSAlertsWorkflow"

	fetchOnceChange = "weather-alert-poll-fetch-once"
)

var pollNWSAlertsRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    2 * time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    3,
	MaximumInterval:    30 * time.Second,
}

var pollNWSAlertsActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 5 * time.Minute,
	HeartbeatTimeout:    30 * time.Second,
	RetryPolicy:         pollNWSAlertsRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        PollNWSAlertsWorkflowName,
			Fn:          PollNWSAlertsWorkflow,
			TaskQueue:   temporaltype.WeatherAlertTaskQueue,
			Description: "Poll active NWS weather alerts and persist them for all tenants",
		},
	}
}

func PollNWSAlertsWorkflow(ctx workflow.Context) (*PollNWSAlertsResult, error) {
	activityCtx := workflow.WithActivityOptions(ctx, pollNWSAlertsActivityOptions)
	if workflow.GetVersion(ctx, fetchOnceChange, workflow.DefaultVersion, 1) ==
		workflow.DefaultVersion {
		return pollPerTenant(ctx, activityCtx)
	}

	var a *Activities
	var result *PollNWSAlertsResult
	if err := workflow.ExecuteActivity(
		activityCtx,
		a.PollNWSAlertsActivity,
	).Get(ctx, &result); err != nil {
		return nil, err
	}

	if err := workflow.ExecuteActivity(
		activityCtx,
		a.ExpireStaleWeatherAlertsActivity,
	).Get(ctx, nil); err != nil {
		return nil, err
	}

	return result, nil
}

func pollPerTenant(
	ctx workflow.Context,
	activityCtx workflow.Context,
) (*PollNWSAlertsResult, error) {
	logger := workflow.GetLogger(ctx)

	var a *Activities
	var tenantsResult *ListWeatherAlertTenantsResult
	if err := workflow.ExecuteActivity(
		activityCtx,
		a.ListWeatherAlertTenantsActivity,
		&ListWeatherAlertTenantsPayload{Limit: temporaljobs.DefaultTenantScanLimit},
	).Get(ctx, &tenantsResult); err != nil {
		return nil, err
	}

	result := &PollNWSAlertsResult{}
	result.TenantsScanned = len(tenantsResult.Tenants)
	for _, tenant := range tenantsResult.Tenants {
		if err := workflow.ExecuteActivity(
			activityCtx,
			a.PollNWSAlertsForTenantActivity,
			&PollNWSAlertsTenantPayload{TenantWorkItem: tenant},
		).Get(ctx, nil); err != nil {
			logger.Error("Weather alert tenant poll failed",
				"orgId", tenant.OrganizationID.String(),
				"buId", tenant.BusinessUnitID.String(),
				"error", err,
			)
			result.AddFailure(tenant, err)
			continue
		}

		result.AddTenantResult(1, 0)
	}

	if err := workflow.ExecuteActivity(
		activityCtx,
		a.ExpireStaleWeatherAlertsActivity,
	).Get(ctx, nil); err != nil {
		return nil, err
	}

	return result, nil
}
