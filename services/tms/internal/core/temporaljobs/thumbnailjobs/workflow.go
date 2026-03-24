package thumbnailjobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var defaultRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    2 * time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    3,
	MaximumInterval:    30 * time.Second,
	NonRetryableErrorTypes: []string{
		temporaltype.ErrorTypeInvalidInput.String(),
		temporaltype.ErrorTypeDataIntegrity.String(),
	},
}

var thumbnailActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 2 * time.Minute,
	HeartbeatTimeout:    30 * time.Second,
	RetryPolicy:         defaultRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        "GenerateThumbnailWorkflow",
			Fn:          GenerateThumbnailWorkflow,
			TaskQueue:   temporaltype.ThumbnailTaskQueue,
			Description: "Generate thumbnail for uploaded document",
		},
	}
}

func GenerateThumbnailWorkflow(
	ctx workflow.Context,
	payload *GenerateThumbnailPayload,
) (*GenerateThumbnailResult, error) {
	ctx = workflow.WithActivityOptions(ctx, thumbnailActivityOptions)

	var a *Activities
	var result *GenerateThumbnailResult

	err := workflow.ExecuteActivity(ctx, a.GenerateThumbnailActivity, payload).Get(ctx, &result)
	if err != nil {
		workflow.GetLogger(ctx).Error(
			"Failed to generate thumbnail",
			"documentId", payload.DocumentID.String(),
			"error", err,
		)
		return nil, err
	}

	workflow.GetLogger(ctx).Info(
		"Thumbnail workflow completed",
		"documentId", payload.DocumentID.String(),
		"success", result.Success,
	)

	return result, nil
}
