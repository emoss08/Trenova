/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package email

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/external/temporaljobs/payloads"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// SendEmailWorkflow sends a single email
func SendEmailWorkflow(ctx workflow.Context, payload *payloads.EmailPayload) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("starting email workflow",
		"organizationId", payload.OrganizationID.String(),
		"to", payload.To,
	)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		HeartbeatTimeout:    5 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var result string
	err := workflow.ExecuteActivity(ctx, SendEmailActivity, payload).Get(ctx, &result)
	if err != nil {
		logger.Error("failed to send email", "error", err)
		return err
	}

	logger.Info("email sent successfully", "result", result)
	return nil
}

// ProcessEmailQueueWorkflow processes a batch of emails from queue
func ProcessEmailQueueWorkflow(ctx workflow.Context, payload *payloads.BasePayload) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("processing email queue",
		"organizationId", payload.OrganizationID.String(),
	)
	
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var emailBatch []payloads.EmailPayload
	err := workflow.ExecuteActivity(ctx, FetchEmailsFromQueueActivity, payload).Get(ctx, &emailBatch)
	if err != nil {
		return err
	}

	for _, email := range emailBatch {
		var result string
		_ = workflow.ExecuteActivity(ctx, SendEmailActivity, &email).Get(ctx, &result)
	}

	return nil
}

// WorkflowDefinition defines a workflow with its configuration
type WorkflowDefinition struct {
	Name        string
	Fn          any
	TaskQueue   string
	Description string
}

// RegisterWorkflows registers all email-related workflows
func RegisterWorkflows() []WorkflowDefinition {
	return []WorkflowDefinition{
		{
			Name:        "SendEmailWorkflow",
			Fn:          SendEmailWorkflow,
			TaskQueue:   "email-tasks",
			Description: "Sends individual emails",
		},
		{
			Name:        "ProcessEmailQueueWorkflow",
			Fn:          ProcessEmailQueueWorkflow,
			TaskQueue:   "email-tasks",
			Description: "Processes batches of emails from queue",
		},
	}
}