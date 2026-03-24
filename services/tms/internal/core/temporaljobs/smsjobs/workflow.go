package smsjobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var smsRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    3,
	MaximumInterval:    30 * time.Second,
	NonRetryableErrorTypes: []string{
		temporaltype.ErrorTypeInvalidInput.String(),
	},
}

var smsActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 30 * time.Second,
	HeartbeatTimeout:    10 * time.Second,
	RetryPolicy:         smsRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        "SendSMSWorkflow",
			Fn:          SendSMSWorkflow,
			TaskQueue:   temporaltype.SMSTaskQueue,
			Description: "Send SMS message via Twilio",
		},
	}
}

func SendSMSWorkflow(
	ctx workflow.Context,
	payload *SendSMSPayload,
) (*SendSMSResult, error) {
	ctx = workflow.WithActivityOptions(ctx, smsActivityOptions)

	var a *Activities
	var result *SendSMSResult

	err := workflow.ExecuteActivity(ctx, a.SendSMSActivity, payload).Get(ctx, &result)
	if err != nil {
		workflow.GetLogger(ctx).Error(
			"Failed to send SMS",
			"organizationId", payload.OrganizationID.String(),
			"error", err,
		)
		return nil, err
	}

	workflow.GetLogger(ctx).Info(
		"SMS workflow completed",
		"organizationId", payload.OrganizationID.String(),
		"success", result.Success,
	)

	return result, nil
}
