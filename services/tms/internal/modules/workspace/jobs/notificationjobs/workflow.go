package notificationjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const NotificationDigestWorkflowName = "NotificationDigestWorkflow"

var digestActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 15 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    10 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    3,
		MaximumInterval:    time.Minute,
	},
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        NotificationDigestWorkflowName,
			Fn:          NotificationDigestWorkflow,
			TaskQueue:   temporaltype.EmailTaskQueue,
			Description: "Email each person who bundles notifications their daily digest",
		},
	}
}

func NotificationDigestWorkflow(ctx workflow.Context) (*notificationservice.DigestRunResult, error) {
	activityCtx := workflow.WithActivityOptions(ctx, digestActivityOptions)

	var a *Activities
	var result *notificationservice.DigestRunResult
	if err := workflow.ExecuteActivity(
		activityCtx,
		a.SendDueDigestsActivity,
		&SendDueDigestsPayload{At: workflow.Now(ctx).Unix()},
	).Get(ctx, &result); err != nil {
		return nil, err
	}

	workflow.GetLogger(ctx).Info("Notification digest workflow completed",
		"subscribers", result.Subscribers,
		"due", result.Due,
		"sent", result.Sent,
		"empty", result.Empty,
		"failed", result.Failed,
	)

	return result, nil
}
