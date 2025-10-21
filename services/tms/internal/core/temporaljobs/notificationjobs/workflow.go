package notificationjobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	NotificationTaskQueue = "notification-queue"
)

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        "SendJobCompleteNotificationWorkflow",
			Fn:          SendJobCompleteNotificationWorkflow,
			TaskQueue:   temporaltype.NotificationTaskQueue,
			Description: "Send a job completion notification",
		},
		{
			Name:        "SendConfigurationCopiedNotificationWorkflow",
			Fn:          SendConfigurationCopiedNotificationWorkflow,
			TaskQueue:   temporaltype.NotificationTaskQueue,
			Description: "Send a configuration copied notification",
		},
	}
}

func SendJobCompleteNotificationWorkflow(
	ctx workflow.Context,
	payload *SendNotificationPayload,
) error {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		HeartbeatTimeout:    2 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
			MaximumInterval:    time.Minute,
		},
	}

	ctx = workflow.WithActivityOptions(ctx, ao)
	so := &workflow.SessionOptions{
		CreationTimeout:  10 * time.Second,
		ExecutionTimeout: 10 * time.Second,
	}

	sessionCtx, err := workflow.CreateSession(ctx, so)
	if err != nil {
		return err
	}
	defer workflow.CompleteSession(sessionCtx)

	var a *Activities
	err = workflow.
		ExecuteActivity(sessionCtx, a.SendNotificationActivity, &payload).
		Get(sessionCtx, nil)
	if err != nil {
		return err
	}

	return nil
}

func SendConfigurationCopiedNotificationWorkflow(
	ctx workflow.Context,
	payload *SendConfigurationCopiedNotificationPayload,
) error {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		HeartbeatTimeout:    2 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
			MaximumInterval:    time.Minute,
		},
	}

	ctx = workflow.WithActivityOptions(ctx, ao)
	so := &workflow.SessionOptions{
		CreationTimeout:  10 * time.Second,
		ExecutionTimeout: 10 * time.Second,
	}

	sessionCtx, err := workflow.CreateSession(ctx, so)
	if err != nil {
		return err
	}
	defer workflow.CompleteSession(sessionCtx)

	var a *Activities
	err = workflow.
		ExecuteActivity(sessionCtx, a.SendConfigurationCopiedNotificationActivity, &payload).
		Get(sessionCtx, nil)
	if err != nil {
		return err
	}

	return nil
}
