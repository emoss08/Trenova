package documentextractionjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/jobs/modelcall"
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

var considerShadowActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    2 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    3,
		MaximumInterval:    30 * time.Second,
		NonRetryableErrorTypes: []string{
			temporaltype.ErrorTypeInvalidInput.String(),
		},
	},
}

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

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        "ProcessDocumentAIExtractionWorkflow",
			Fn:          ProcessDocumentAIExtractionWorkflow,
			TaskQueue:   temporaltype.DocumentIntelligenceTaskQueue,
			Description: "Run asynchronous AI extraction for an indexed document",
		},
	}
}

// ProcessDocumentAIExtractionWorkflow runs a document's AI extraction: it
// submits the extraction, waits for the model on a durable timer, polling
// until it answers or the longest wait passes, and applies what came back.
//
// Waiting here puts the whole extraction in one execution's history: the
// submit, every poll, and the apply, each visible in the Temporal UI.
func ProcessDocumentAIExtractionWorkflow(
	ctx workflow.Context,
	payload *ProcessDocumentAIExtractionPayload,
) (*ProcessDocumentAIExtractionResult, error) {
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

	considerShadow(ctx, payload)

	return result, nil
}

func considerShadow(ctx workflow.Context, payload *ProcessDocumentAIExtractionPayload) {
	ctx = workflow.WithActivityOptions(ctx, considerShadowActivityOptions)

	var a *Activities
	if err := workflow.ExecuteActivity(
		ctx,
		a.ConsiderDocumentAIExtractionShadowActivity,
		&ConsiderDocumentAIExtractionShadowInput{
			BasePayload: temporaltype.BasePayload{
				OrganizationID: payload.OrganizationID,
				BusinessUnitID: payload.BusinessUnitID,
				UserID:         payload.UserID,
			},
			DocumentID:  payload.DocumentID,
			ExtractedAt: payload.ExtractedAt,
		},
	).Get(ctx, nil); err != nil {
		workflow.GetLogger(ctx).Warn("could not consider a shadow extraction",
			"documentId", payload.DocumentID.String(),
			"error", err,
		)
	}
}
