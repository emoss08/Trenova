package documentuploadjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var finalizeRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    2 * time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    5,
	MaximumInterval:    30 * time.Second,
}

var finalizeActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 3 * time.Minute,
	HeartbeatTimeout:    30 * time.Second,
	RetryPolicy:         finalizeRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        "FinalizeDocumentUploadWorkflow",
			Fn:          FinalizeDocumentUploadWorkflow,
			TaskQueue:   temporaltype.UploadTaskQueue,
			Description: "Finalize a direct-to-storage document upload",
		},
		{
			Name:        "ReconcileDocumentUploadsWorkflow",
			Fn:          ReconcileDocumentUploadsWorkflow,
			TaskQueue:   temporaltype.UploadTaskQueue,
			Description: "Reconcile stale document uploads and pending previews",
		},
		{
			Name:        "CleanupDocumentStorageWorkflow",
			Fn:          CleanupDocumentStorageWorkflow,
			TaskQueue:   temporaltype.UploadTaskQueue,
			Description: "Retry cleanup of deleted document storage objects",
		},
	}
}

func FinalizeDocumentUploadWorkflow(
	ctx workflow.Context,
	payload *FinalizeUploadPayload,
) (*FinalizeUploadResult, error) {
	ctx = workflow.WithActivityOptions(ctx, finalizeActivityOptions)

	var a *Activities
	var result *FinalizeUploadResult

	if err := workflow.ExecuteActivity(
		ctx,
		a.FinalizeUploadActivity,
		payload,
	).Get(ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func ReconcileDocumentUploadsWorkflow(
	ctx workflow.Context,
) (*ReconcileUploadsResult, error) {
	activityCtx := workflow.WithActivityOptions(ctx, finalizeActivityOptions)
	logger := workflow.GetLogger(ctx)

	var a *Activities
	now := workflow.Now(ctx)

	payload := &ReconcileUploadsPayload{
		BasePayload: temporaltype.BasePayload{
			Timestamp: now.Unix(),
		},
		StaleAfterSeconds:   int64((10 * time.Minute).Seconds()),
		PendingAfterSeconds: int64((10 * time.Minute).Seconds()),
		Limit:               100,
	}

	var tenantsResult *ListReconcileUploadsTenantsResult
	if err := workflow.ExecuteActivity(
		activityCtx,
		a.ListReconcileUploadsTenantsActivity,
		payload,
	).Get(ctx, &tenantsResult); err != nil {
		return nil, err
	}

	result := &ReconcileUploadsResult{}
	result.TenantsScanned = len(tenantsResult.Tenants)
	for _, tenant := range tenantsResult.Tenants {
		tenantPayload := *payload
		tenantPayload.OrganizationID = tenant.OrganizationID
		tenantPayload.BusinessUnitID = tenant.BusinessUnitID
		tenantPayload.Limit = temporaljobs.NormalizeLimit(
			tenant.Limit,
			temporaljobs.DefaultTenantRecordLimit,
		)

		var tenantResult *ReconcileUploadsResult
		if err := workflow.ExecuteActivity(
			activityCtx,
			a.ReconcileUploadsActivity,
			&tenantPayload,
		).Get(ctx, &tenantResult); err != nil {
			logger.Error("Document upload reconciliation tenant failed",
				"orgId", tenant.OrganizationID.String(),
				"buId", tenant.BusinessUnitID.String(),
				"error", err,
			)
			result.AddFailure(tenant, err)
			continue
		}

		processed := tenantResult.StaleSessionsProcessed + tenantResult.PreviewRetriesStarted
		result.AddTenantResult(processed, 0)
		result.StaleSessionsProcessed += tenantResult.StaleSessionsProcessed
		result.FinalizationsStarted += tenantResult.FinalizationsStarted
		result.SessionsExpired += tenantResult.SessionsExpired
		result.PreviewRetriesStarted += tenantResult.PreviewRetriesStarted
	}

	logger.Info("Document upload reconciliation workflow completed",
		"tenantsScanned", result.TenantsScanned,
		"tenantsProcessed", result.TenantsProcessed,
		"recordsProcessed", result.RecordsProcessed,
		"failureCount", result.FailureCount,
	)

	return result, nil
}

func CleanupDocumentStorageWorkflow(
	ctx workflow.Context,
	payload *CleanupDocumentStoragePayload,
) error {
	ctx = workflow.WithActivityOptions(ctx, finalizeActivityOptions)

	var a *Activities
	return workflow.ExecuteActivity(ctx, a.CleanupDocumentStorageActivity, payload).Get(ctx, nil)
}
