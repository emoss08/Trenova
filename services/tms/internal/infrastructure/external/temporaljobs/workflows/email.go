/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package workflows

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/external/temporaljobs"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// SendEmailWorkflow sends an email with retry logic and error handling
func SendEmailWorkflow(ctx workflow.Context, payload *temporaljobs.EmailPayload) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("starting email workflow",
		"organizationId", payload.OrganizationID.String(),
		"businessUnitId", payload.BusinessUnitID.String(),
		"to", payload.To,
		"subject", payload.Subject,
	)

	// Set activity options
	ao := workflow.ActivityOptions{
		TaskQueue:           temporaljobs.TaskQueueEmail,
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

	// Check if we need to delay execution
	if metadata := payload.Metadata; metadata != nil {
		if delaySeconds, ok := metadata["delay_seconds"].(float64); ok && delaySeconds > 0 {
			delay := time.Duration(delaySeconds) * time.Second
			logger.Info("delaying email workflow execution", "delay", delay)
			if err := workflow.Sleep(ctx, delay); err != nil {
				return err
			}
		}
	}

	// Execute the send email activity
	var result string
	err := workflow.ExecuteActivity(ctx, temporaljobs.ActivitySendEmail, payload).Get(ctx, &result)
	if err != nil {
		logger.Error("failed to send email", "error", err)
		return err
	}

	logger.Info("email sent successfully", "result", result)
	return nil
}

// // ProcessEmailQueueWorkflow processes a batch of emails from a queue
// func ProcessEmailQueueWorkflow(ctx workflow.Context, payload *temporaljobs.BasePayload) error {
// 	logger := workflow.GetLogger(ctx)
// 	logger.Info("starting email queue processing workflow",
// 		"organizationId", payload.OrganizationID.String(),
// 		"businessUnitId", payload.BusinessUnitID.String(),
// 	)

// 	// Set activity options for longer running batch processing
// 	ao := workflow.ActivityOptions{
// 		TaskQueue:           temporaljobs.TaskQueueEmail,
// 		StartToCloseTimeout: 5 * time.Minute,
// 		HeartbeatTimeout:    30 * time.Second,
// 		RetryPolicy: &temporal.RetryPolicy{
// 			InitialInterval:    2 * time.Second,
// 			BackoffCoefficient: 2.0,
// 			MaximumInterval:    time.Minute,
// 			MaximumAttempts:    2,
// 		},
// 		Summary: "Process Email Queue",
// 	}
// 	ctx = workflow.WithActivityOptions(ctx, ao)

// 	// First, fetch emails from queue
// 	var emailBatch []temporaljobs.EmailPayload
// 	err := workflow.
// 		ExecuteActivity(ctx, temporaljobs.ActivityFetchEmailsFromQueue, payload).
// 		Get(ctx, &emailBatch)
// 	if err != nil {
// 		logger.Error("failed to fetch emails from queue", "error", err)
// 		return err
// 	}

// 	logger.Info("fetched emails from queue", "count", len(emailBatch))

// 	// Process each email in parallel using workflow.Go
// 	var futures []workflow.Future
// 	for i, email := range emailBatch {
// 		email := email // capture loop variable
// 		future := workflow.ExecuteActivity(ctx, temporaljobs.ActivitySendEmail, &email)
// 		futures = append(futures, future)

// 		// Limit parallelism to avoid overwhelming the system
// 		if (i+1)%10 == 0 {
// 			// Wait for current batch before continuing
// 			for _, f := range futures {
// 				var result string
// 				if err := f.Get(ctx, &result); err != nil {
// 					logger.Warn("failed to send email in batch", "error", err)
// 					// Continue processing other emails even if one fails
// 				}
// 			}
// 			futures = nil
// 		}
// 	}

// 	// Wait for remaining futures
// 	for _, f := range futures {
// 		var result string
// 		if err := f.Get(ctx, &result); err != nil {
// 			logger.Warn("failed to send email in batch", "error", err)
// 		}
// 	}

// 	logger.Info("email queue processing completed")
// 	return nil
// }
