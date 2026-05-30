package emailjobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var emailRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    5,
	MaximumInterval:    30 * time.Second,
	NonRetryableErrorTypes: []string{
		temporaltype.ErrorTypeInvalidInput.String(),
		temporaltype.ErrorTypeNonRetryable.String(),
		temporaltype.ErrorTypePermissionDenied.String(),
		temporaltype.ErrorTypeDataIntegrity.String(),
	},
}

var emailActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: time.Minute,
	HeartbeatTimeout:    15 * time.Second,
	RetryPolicy:         emailRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        SendEmailWorkflowName,
			Fn:          SendEmailWorkflow,
			TaskQueue:   temporaltype.EmailTaskQueue,
			Description: "Send persisted email message via provider",
		},
	}
}

func SendEmailWorkflow(
	ctx workflow.Context,
	payload *SendEmailPayload,
) (*SendEmailResult, error) {
	ctx = workflow.WithActivityOptions(ctx, emailActivityOptions)

	var a *Activities
	var result *SendEmailResult

	err := workflow.ExecuteActivity(ctx, a.SendEmailActivity, payload).Get(ctx, &result)
	if err != nil {
		workflow.GetLogger(ctx).Error(
			"Failed to send email",
			"messageId", payload.MessageID.String(),
			"organizationId", payload.OrganizationID.String(),
			"error", err,
		)
		return nil, err
	}

	workflow.GetLogger(ctx).Info(
		"Email workflow completed",
		"messageId", result.MessageID.String(),
		"status", string(result.Status),
	)

	return result, nil
}
