package ptojobs

import (
	"time"

	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var ptoRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    3,
	MaximumInterval:    30 * time.Second,
}

var ptoActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 15 * time.Minute,
	HeartbeatTimeout:    2 * time.Minute,
	RetryPolicy:         ptoRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        PTOAccrualWorkflowName,
			Fn:          PTOAccrualWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Post scheduled PTO accruals, carryover caps, and expiries for every assigned worker",
		},
	}
}

func PTOAccrualWorkflow(
	ctx workflow.Context,
	input PTOAccrualWorkflowInput,
) (*PTOAccrualResult, error) {
	ctx = workflow.WithActivityOptions(ctx, ptoActivityOptions)
	logger := workflow.GetLogger(ctx)

	var a *Activities
	var tenants []temporaljobs.TenantWorkItem
	if err := workflow.ExecuteActivity(ctx, a.ListAccrualTenantsActivity, input).Get(ctx, &tenants); err != nil {
		logger.Error("PTO accrual workflow failed to list tenants", "error", err)
		return nil, err
	}

	result := &PTOAccrualResult{}
	result.TenantsScanned = len(tenants)

	for _, tenant := range tenants {
		afterID := pulid.Nil
		processed, skipped := 0, 0
		var tenantErr error
		for {
			var page AccrualTenantPageResult
			pageInput := AccrualTenantPageInput{
				TenantWorkItem: tenant,
				WorkerID:       input.WorkerID,
				AsOf:           input.AsOf,
				AfterID:        afterID,
			}
			if tenantErr = workflow.ExecuteActivity(ctx, a.AccrueTenantPageActivity, pageInput).
				Get(ctx, &page); tenantErr != nil {
				break
			}
			processed += page.WorkersProcessed
			skipped += page.WorkersSkipped
			result.EntriesPosted += page.EntriesPosted
			result.EntriesCapped += page.EntriesCapped
			result.EntriesSkipped += page.EntriesSkipped
			if page.NextAfterID.IsNil() || !input.WorkerID.IsNil() {
				break
			}
			afterID = page.NextAfterID
		}
		if tenantErr != nil {
			logger.Error("PTO accrual failed for tenant",
				"organizationId", tenant.OrganizationID.String(), "error", tenantErr)
			result.AddFailure(tenant, tenantErr)
			continue
		}
		result.AddTenantResult(processed, skipped)
	}

	result.CompletedAt = workflow.Now(ctx).Unix()
	logger.Info("PTO accrual workflow completed",
		"tenants", result.TenantsProcessed,
		"workers", result.RecordsProcessed,
		"posted", result.EntriesPosted,
		"capped", result.EntriesCapped,
		"failed", result.FailureCount,
	)

	return result, nil
}
