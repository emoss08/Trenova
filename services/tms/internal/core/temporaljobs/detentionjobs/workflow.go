package detentionjobs

import (
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/documentuploadjobs"
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

// storeNoticePDFRetryPolicy retries the filing generously. The notice has
// already reached the customer by the time this runs, so there is no user waiting
// on it — only the claim file, which is worth getting right eventually.
var storeNoticePDFRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    5 * time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    5,
	MaximumInterval:    2 * time.Minute,
}

var storeNoticePDFActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 2 * time.Minute,
	RetryPolicy:         storeNoticePDFRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        SweepDetentionNoticesWorkflowName,
			Fn:          SweepDetentionNoticesWorkflow,
			TaskQueue:   temporaltype.TaskQueueBilling.String(),
			Description: "Send due detention notices for tenants whose policies auto-send",
		},
		{
			Name:        StoreNoticePDFWorkflowName,
			Fn:          StoreNoticePDFWorkflow,
			TaskQueue:   temporaltype.TaskQueueBilling.String(),
			Description: "File a sent detention notice PDF as a shipment document",
		},
	}
}

// StoreNoticePDFWorkflow turns the uploaded notice PDF into a filed document and
// points the notice at it.
//
// It runs after the notice has been emailed, which is why it is a workflow at all
// rather than part of the send: the customer must not wait on document
// finalization, and the filing must not be lost if it fails once.
func StoreNoticePDFWorkflow(
	ctx workflow.Context,
	payload *StoreNoticePDFPayload,
) (*StoreNoticePDFResult, error) {
	activityCtx := workflow.WithActivityOptions(ctx, storeNoticePDFActivityOptions)

	childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID: fmt.Sprintf("document-upload-finalize-%s", payload.SessionID.String()),
		TaskQueue:  temporaltype.UploadTaskQueue,
	})
	finalized := new(documentuploadjobs.FinalizeUploadResult)
	if err := workflow.ExecuteChildWorkflow(
		childCtx,
		documentuploadjobs.FinalizeDocumentUploadWorkflow,
		&documentuploadjobs.FinalizeUploadPayload{
			BasePayload:   payload.BasePayload,
			SessionID:     payload.SessionID,
			PrincipalType: payload.PrincipalType,
			PrincipalID:   payload.PrincipalID,
			APIKeyID:      payload.APIKeyID,
		},
	).Get(ctx, finalized); err != nil {
		workflow.GetLogger(ctx).Error("Detention notice PDF finalization failed", "error", err)
		return nil, err
	}
	if finalized.DocumentID == nil || finalized.DocumentID.IsNil() {
		return nil, temporal.NewNonRetryableApplicationError(
			"Detention notice PDF finalization did not produce a document",
			"detention-notice-pdf-document-missing",
			nil,
		)
	}

	var a *Activities
	result := new(StoreNoticePDFResult)
	if err := workflow.ExecuteActivity(
		activityCtx,
		a.AttachNoticePDFDocumentActivity,
		payload,
		*finalized.DocumentID,
	).Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("Detention notice PDF filing failed", "error", err)
		return nil, err
	}

	return result, nil
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
