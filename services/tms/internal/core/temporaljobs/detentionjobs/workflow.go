package detentionjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var sweepNoticesRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    2 * time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    3,
	MaximumInterval:    30 * time.Second,
}

var sweepNoticesActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 5 * time.Minute,
	HeartbeatTimeout:    30 * time.Second,
	RetryPolicy:         sweepNoticesRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        SweepDetentionNoticesWorkflowName,
			Fn:          SweepDetentionNoticesWorkflow,
			TaskQueue:   temporaltype.TaskQueueBilling.String(),
			Description: "Send due detention notices for tenants whose policies auto-send",
		},
	}
}

// SweepDetentionNoticesWorkflow fans out across every tenant with the
// detention policy engine enabled and delivers the customer notices whose
// contractual window has opened. Missing the window converts collectible
// detention into a write-off, so this runs on a tight cadence and treats a
// per-tenant failure as isolated rather than fatal.
func SweepDetentionNoticesWorkflow(
	ctx workflow.Context,
) (*SweepDetentionNoticesResult, error) {
	activityCtx := workflow.WithActivityOptions(ctx, sweepNoticesActivityOptions)
	logger := workflow.GetLogger(ctx)

	var a *Activities
	var tenantsResult *ListDetentionTenantsResult
	if err := workflow.ExecuteActivity(
		activityCtx,
		a.ListDetentionTenantsActivity,
		&ListDetentionTenantsPayload{Limit: temporaljobs.DefaultTenantScanLimit},
	).Get(ctx, &tenantsResult); err != nil {
		return nil, err
	}

	result := &SweepDetentionNoticesResult{}
	result.TenantsScanned = len(tenantsResult.Tenants)

	for _, tenant := range tenantsResult.Tenants {
		var sweepResult *SweepTenantNoticesResult
		if err := workflow.ExecuteActivity(
			activityCtx,
			a.SweepTenantNoticesActivity,
			&SweepTenantNoticesPayload{TenantWorkItem: tenant},
		).Get(ctx, &sweepResult); err != nil {
			logger.Error("Detention notice sweep tenant failed",
				"orgId", tenant.OrganizationID.String(),
				"buId", tenant.BusinessUnitID.String(),
				"error", err,
			)
			result.AddFailure(tenant, err)
			continue
		}

		result.AddTenantResult(1, sweepResult.Due)
		result.NoticesSent += sweepResult.Sent
		result.NoticesSkipped += sweepResult.Skipped
		result.NoticesFailed += sweepResult.Failed
	}

	logger.Info("Detention notice sweep completed",
		"tenantsScanned", result.TenantsScanned,
		"tenantsProcessed", result.TenantsProcessed,
		"failureCount", result.FailureCount,
		"noticesSent", result.NoticesSent,
		"noticesSkipped", result.NoticesSkipped,
		"noticesFailed", result.NoticesFailed,
	)

	return result, nil
}
