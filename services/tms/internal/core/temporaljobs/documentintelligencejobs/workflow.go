package documentintelligencejobs

import (
	"time"

	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var defaultRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    2 * time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    4,
	MaximumInterval:    1 * time.Minute,
	NonRetryableErrorTypes: []string{
		temporaltype.ErrorTypeInvalidInput.String(),
	},
}

var defaultActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 5 * time.Minute,
	HeartbeatTimeout:    30 * time.Second,
	RetryPolicy:         defaultRetryPolicy,
}

var asyncAIExtractionActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 20 * time.Minute,
	RetryPolicy:         defaultRetryPolicy,
}

var pollPendingAIExtractionsActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 2 * time.Minute,
	RetryPolicy:         defaultRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        "ProcessDocumentIntelligenceWorkflow",
			Fn:          ProcessDocumentIntelligenceWorkflow,
			TaskQueue:   temporaltype.DocumentIntelligenceTaskQueue,
			Description: "Extract and classify document content",
		},
		{
			Name:        "ReconcileDocumentIntelligenceWorkflow",
			Fn:          ReconcileDocumentIntelligenceWorkflow,
			TaskQueue:   temporaltype.DocumentIntelligenceTaskQueue,
			Description: "Re-drive stale document intelligence jobs",
		},
		{
			Name:        "ProcessDocumentAIExtractionWorkflow",
			Fn:          ProcessDocumentAIExtractionWorkflow,
			TaskQueue:   temporaltype.DocumentIntelligenceTaskQueue,
			Description: "Run asynchronous AI extraction for an indexed document",
		},
		{
			Name:        "PollPendingDocumentAIExtractionsWorkflow",
			Fn:          PollPendingDocumentAIExtractionsWorkflow,
			TaskQueue:   temporaltype.DocumentIntelligenceTaskQueue,
			Description: "Poll pending OpenAI background extraction jobs",
		},
	}
}

func ProcessDocumentIntelligenceWorkflow(
	ctx workflow.Context,
	payload *ProcessDocumentIntelligencePayload,
) (*ProcessDocumentIntelligenceResult, error) {
	ctx = workflow.WithActivityOptions(ctx, defaultActivityOptions)

	var a *Activities
	var result *ProcessDocumentIntelligenceResult
	if err := workflow.ExecuteActivity(
		ctx, a.ProcessDocumentIntelligenceActivity, payload,
	).Get(ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func ReconcileDocumentIntelligenceWorkflow(
	ctx workflow.Context,
) (*ReconcileDocumentIntelligenceResult, error) {
	activityCtx := workflow.WithActivityOptions(ctx, defaultActivityOptions)
	logger := workflow.GetLogger(ctx)

	var a *Activities
	payload := &ReconcileDocumentIntelligencePayload{
		BasePayload: temporaltype.BasePayload{
			Timestamp: workflow.Now(ctx).Unix(),
		},
		OlderThanSeconds: int64((10 * time.Minute).Seconds()),
		Limit:            temporaljobs.DefaultTenantRecordLimit,
	}

	var tenantsResult *ListDocumentIntelligenceTenantsResult
	if err := workflow.ExecuteActivity(
		activityCtx,
		a.ListDocumentIntelligenceTenantsActivity,
		payload,
	).Get(ctx, &tenantsResult); err != nil {
		return nil, err
	}

	result := &ReconcileDocumentIntelligenceResult{}
	result.TenantsScanned = len(tenantsResult.Tenants)
	for _, tenant := range tenantsResult.Tenants {
		tenantPayload := *payload
		tenantPayload.OrganizationID = tenant.OrganizationID
		tenantPayload.BusinessUnitID = tenant.BusinessUnitID
		tenantPayload.Limit = temporaljobs.NormalizeLimit(
			tenant.Limit,
			temporaljobs.DefaultTenantRecordLimit,
		)

		var tenantResult *ReconcileDocumentIntelligenceResult
		if err := workflow.ExecuteActivity(
			activityCtx,
			a.ReconcileDocumentIntelligenceActivity,
			&tenantPayload,
		).Get(ctx, &tenantResult); err != nil {
			logger.Error("Document intelligence reconciliation tenant failed",
				"orgId", tenant.OrganizationID.String(),
				"buId", tenant.BusinessUnitID.String(),
				"error", err,
			)
			result.AddFailure(tenant, err)
			continue
		}

		result.AddTenantResult(tenantResult.Queued, 0)
		result.Queued += tenantResult.Queued
	}

	return result, nil
}

func ProcessDocumentAIExtractionWorkflow(
	ctx workflow.Context,
	payload *ProcessDocumentAIExtractionPayload,
) (*ProcessDocumentAIExtractionResult, error) {
	ctx = workflow.WithActivityOptions(ctx, asyncAIExtractionActivityOptions)

	var a *Activities
	var completion *AsyncAIExtractionCompletion
	if err := workflow.ExecuteActivity(
		ctx, a.SubmitAndAwaitDocumentAIExtractionActivity, payload,
	).Get(ctx, &completion); err != nil {
		return nil, err
	}

	var result *ProcessDocumentAIExtractionResult
	if err := workflow.ExecuteActivity(
		ctx,
		a.ApplyDocumentAIExtractionResultActivity,
		&ApplyDocumentAIExtractionPayload{
			BasePayload: temporaltype.BasePayload{
				OrganizationID: payload.OrganizationID,
				BusinessUnitID: payload.BusinessUnitID,
				UserID:         payload.UserID,
			},
			DocumentID:  payload.DocumentID,
			ExtractedAt: payload.ExtractedAt,
			Completion:  completion,
		},
	).Get(ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func PollPendingDocumentAIExtractionsWorkflow(
	ctx workflow.Context,
	payload *PollPendingDocumentAIExtractionsPayload,
) (*PollPendingDocumentAIExtractionsResult, error) {
	activityCtx := workflow.WithActivityOptions(ctx, pollPendingAIExtractionsActivityOptions)
	logger := workflow.GetLogger(ctx)
	if payload == nil {
		payload = &PollPendingDocumentAIExtractionsPayload{}
	}

	var a *Activities
	var tenantsResult *ListDocumentIntelligenceTenantsResult
	if err := workflow.ExecuteActivity(
		activityCtx,
		a.ListPollableDocumentAIExtractionTenantsActivity,
		payload,
	).Get(ctx, &tenantsResult); err != nil {
		return nil, err
	}

	result := &PollPendingDocumentAIExtractionsResult{}
	result.TenantsScanned = len(tenantsResult.Tenants)
	for _, tenant := range tenantsResult.Tenants {
		tenantPayload := *payload
		tenantPayload.OrganizationID = tenant.OrganizationID
		tenantPayload.BusinessUnitID = tenant.BusinessUnitID
		tenantPayload.Limit = temporaljobs.NormalizeLimit(
			tenant.Limit,
			temporaljobs.DefaultTenantRecordLimit,
		)

		var tenantResult *PollPendingDocumentAIExtractionsResult
		if err := workflow.ExecuteActivity(
			activityCtx,
			a.PollPendingDocumentAIExtractionsActivity,
			&tenantPayload,
		).Get(ctx, &tenantResult); err != nil {
			logger.Error("Document AI extraction polling tenant failed",
				"orgId", tenant.OrganizationID.String(),
				"buId", tenant.BusinessUnitID.String(),
				"error", err,
			)
			result.AddFailure(tenant, err)
			continue
		}

		processed := tenantResult.Completed + tenantResult.Pending + tenantResult.Failed
		result.AddTenantResult(processed, 0)
		result.Completed += tenantResult.Completed
		result.Pending += tenantResult.Pending
		result.Failed += tenantResult.Failed
	}

	return result, nil
}
