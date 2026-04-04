package documentintelligencejobs

import (
	"time"

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
	if err := workflow.ExecuteActivity(ctx, a.ProcessDocumentIntelligenceActivity, payload).Get(ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func ReconcileDocumentIntelligenceWorkflow(
	ctx workflow.Context,
) (*ReconcileDocumentIntelligenceResult, error) {
	ctx = workflow.WithActivityOptions(ctx, defaultActivityOptions)

	var a *Activities
	var result *ReconcileDocumentIntelligenceResult
	if err := workflow.ExecuteActivity(ctx, a.ReconcileDocumentIntelligenceActivity, &ReconcileDocumentIntelligencePayload{
		BasePayload: temporaltype.BasePayload{
			Timestamp: workflow.Now(ctx).Unix(),
		},
		OlderThanSeconds: int64((10 * time.Minute).Seconds()),
	}).Get(ctx, &result); err != nil {
		return nil, err
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
	if err := workflow.ExecuteActivity(ctx, a.SubmitAndAwaitDocumentAIExtractionActivity, payload).Get(ctx, &completion); err != nil {
		return nil, err
	}

	var result *ProcessDocumentAIExtractionResult
	if err := workflow.ExecuteActivity(ctx, a.ApplyDocumentAIExtractionResultActivity, &ApplyDocumentAIExtractionPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: payload.OrganizationID,
			BusinessUnitID: payload.BusinessUnitID,
			UserID:         payload.UserID,
		},
		DocumentID:  payload.DocumentID,
		ExtractedAt: payload.ExtractedAt,
		Completion:  completion,
	}).Get(ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func PollPendingDocumentAIExtractionsWorkflow(
	ctx workflow.Context,
	payload *PollPendingDocumentAIExtractionsPayload,
) (*PollPendingDocumentAIExtractionsResult, error) {
	ctx = workflow.WithActivityOptions(ctx, pollPendingAIExtractionsActivityOptions)

	var a *Activities
	var result *PollPendingDocumentAIExtractionsResult
	if err := workflow.ExecuteActivity(ctx, a.PollPendingDocumentAIExtractionsActivity, payload).Get(ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}
