package documentintelligencejobs

import (
	"time"

	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
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

// pollOnTimerChange marks where an extraction stopped waiting in an activity
// the global poller completed. One started before it finishes that way.
const pollOnTimerChange = "document-ai-extraction-timer-poll"

// submitAIExtractionActivityOptions bound a submit, which runs the model inline
// when no provider can defer it, so it heartbeats on a timer rather than on
// progress.
var submitAIExtractionActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 20 * time.Minute,
	HeartbeatTimeout:    modelcall.HeartbeatTimeout,
	RetryPolicy:         defaultRetryPolicy,
}

// pollAIExtractionActivityOptions bound one poll: a record read, one call to
// the provider, and a record write.
var pollAIExtractionActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: time.Minute,
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

// ProcessDocumentAIExtractionWorkflow runs a document's AI extraction: it
// submits the extraction, waits for the model on a durable timer, polling
// until it answers or the longest wait passes, and applies what came back.
//
// Waiting here, rather than in an activity a global poller completes by task
// token, puts the whole extraction in one execution's history: the submit,
// every poll, and the apply, each visible in the Temporal UI.
func ProcessDocumentAIExtractionWorkflow(
	ctx workflow.Context,
	payload *ProcessDocumentAIExtractionPayload,
) (*ProcessDocumentAIExtractionResult, error) {
	if workflow.GetVersion(ctx, pollOnTimerChange, workflow.DefaultVersion, 1) ==
		workflow.DefaultVersion {
		return extractWithTaskToken(ctx, payload)
	}

	var a *Activities
	submitCtx := workflow.WithActivityOptions(ctx, submitAIExtractionActivityOptions)
	var progress AIExtractionProgress
	if err := workflow.ExecuteActivity(submitCtx, a.SubmitDocumentAIExtractionActivity, payload).
		Get(submitCtx, &progress); err != nil {
		return nil, err
	}

	deadline := workflow.Now(ctx).Add(documentAIExtractionMaxWait)
	pollCtx := workflow.WithActivityOptions(ctx, pollAIExtractionActivityOptions)
	for progress.Completion == nil {
		if err := workflow.Sleep(ctx, documentAIExtractionPollInterval); err != nil {
			return nil, err
		}

		input := &PollDocumentAIExtractionInput{
			Payload: payload,
			GiveUp:  !workflow.Now(ctx).Before(deadline),
		}
		if err := workflow.ExecuteActivity(pollCtx, a.PollDocumentAIExtractionActivity, input).
			Get(pollCtx, &progress); err != nil {
			return nil, err
		}
	}

	return applyAIExtraction(ctx, payload, progress.Completion)
}

// extractWithTaskToken is the extraction as it was before the workflow polled
// on its own timer: one activity the global poller completes by task token.
// It runs only executions that started on it, and must not change.
func extractWithTaskToken(
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

	return applyAIExtraction(ctx, payload, completion)
}

func applyAIExtraction(
	ctx workflow.Context,
	payload *ProcessDocumentAIExtractionPayload,
	completion *AsyncAIExtractionCompletion,
) (*ProcessDocumentAIExtractionResult, error) {
	ctx = workflow.WithActivityOptions(ctx, asyncAIExtractionActivityOptions)

	var a *Activities
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
