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
