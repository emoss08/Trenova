package emailjobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	EmailTaskQueue = "email-queue"
)

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        "SendEmailWorkflow",
			Fn:          SendEmailWorkflow,
			TaskQueue:   temporaltype.EmailTaskQueue,
			Description: "Send an email",
		},
		{
			Name:        "SendTemplatedEmailWorkflow",
			Fn:          SendTemplatedEmailWorkflow,
			TaskQueue:   temporaltype.EmailTaskQueue,
			Description: "Send a templated email",
		},
	}
}

func SendEmailWorkflow(
	ctx workflow.Context,
	payload *temporaltype.SendEmailPayload,
) (*temporaltype.EmailResult, error) {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		HeartbeatTimeout:    5 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    5,
			MaximumInterval:    5 * time.Minute,
		},
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	so := &workflow.SessionOptions{
		CreationTimeout:  10 * time.Second,
		ExecutionTimeout: 60 * time.Second,
	}

	sessionCtx, err := workflow.CreateSession(ctx, so)
	if err != nil {
		return nil, err
	}
	defer workflow.CompleteSession(sessionCtx)

	var a *Activities
	var result temporaltype.EmailResult

	err = workflow.
		ExecuteActivity(sessionCtx, a.SendEmailActivity, payload).
		Get(sessionCtx, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func SendTemplatedEmailWorkflow(
	ctx workflow.Context,
	payload *temporaltype.SendTemplatedEmailPayload,
) (*temporaltype.EmailResult, error) {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		HeartbeatTimeout:    5 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    5,
			MaximumInterval:    5 * time.Minute,
		},
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	so := &workflow.SessionOptions{
		CreationTimeout:  10 * time.Second,
		ExecutionTimeout: 60 * time.Second,
	}

	sessionCtx, err := workflow.CreateSession(ctx, so)
	if err != nil {
		return nil, err
	}
	defer workflow.CompleteSession(sessionCtx)

	var a *Activities

	var renderedEmail temporaltype.SendEmailPayload
	err = workflow.
		ExecuteActivity(sessionCtx, a.RenderTemplateActivity, payload).
		Get(sessionCtx, &renderedEmail)
	if err != nil {
		return nil, err
	}

	var result temporaltype.EmailResult
	err = workflow.
		ExecuteActivity(sessionCtx, a.SendEmailActivity, &renderedEmail).
		Get(sessionCtx, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
