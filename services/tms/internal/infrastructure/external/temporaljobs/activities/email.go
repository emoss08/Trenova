/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package activities

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/external/temporaljobs"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rs/zerolog"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
)

// EmailActivitiesParams defines dependencies for email activities
type EmailActivitiesParams struct {
	fx.In

	Logger       *logger.Logger
	EmailService services.EmailService
}

// EmailActivities contains email-related activities
type EmailActivities struct {
	logger       *zerolog.Logger
	emailService services.EmailService
}

// NewEmailActivities creates a new email activities instance
func NewEmailActivities(p EmailActivitiesParams) *EmailActivities {
	log := p.Logger.With().
		Str("component", "temporal-email-activities").
		Logger()

	return &EmailActivities{
		logger:       &log,
		emailService: p.EmailService,
	}
}

// SendEmail sends a single email
func (ea *EmailActivities) SendEmail(
	ctx context.Context,
	payload *temporaljobs.EmailPayload,
) (string, error) {
	logger := activity.GetLogger(ctx)
	info := activity.GetInfo(ctx)

	logger.Info("sending email",
		"activityID", info.ActivityID,
		"workflowID", info.WorkflowExecution.ID,
		"to", payload.To,
		"subject", payload.Subject,
	)

	activity.RecordHeartbeat(ctx, "preparing email")

	emailReq := &services.SendEmailRequest{
		OrganizationID: payload.OrganizationID,
		BusinessUnitID: payload.BusinessUnitID,
		UserID:         payload.UserID,
		To:             payload.To,
		Subject:        payload.Subject,
		TextBody:       payload.Body,
		HTMLBody:       payload.BodyHTML,
	}

	response, err := ea.emailService.SendEmail(ctx, emailReq)
	if err != nil {
		logger.Error("failed to send email via service", "error", err)
		return "", fmt.Errorf("send email via service: %w", err)
	}

	return response.MessageID, nil
}

// FetchEmailsFromQueue fetches emails from a queue for batch processing
func (ea *EmailActivities) FetchEmailsFromQueue(
	ctx context.Context,
	payload *temporaljobs.BasePayload,
) ([]temporaljobs.EmailPayload, error) {
	logger := activity.GetLogger(ctx)
	info := activity.GetInfo(ctx)

	logger.Info("fetching emails from queue",
		"activityID", info.ActivityID,
		"workflowID", info.WorkflowExecution.ID,
		"organizationId", payload.OrganizationID.String(),
	)

	// Placeholder implementation
	// In a real implementation, you would:
	// 1. Connect to your message queue (Redis, RabbitMQ, etc.)
	// 2. Fetch pending emails
	// 3. Transform them to EmailPayload format
	// 4. Return the batch

	// For demo purposes, return an empty batch
	emails := []temporaljobs.EmailPayload{}

	// Example of how you might populate this:
	// emails = append(emails, temporaljobs.EmailPayload{
	//     BasePayload: *payload,
	//     To:          []string{"user@example.com"},
	//     Subject:     "Test Email",
	//     Body:        "This is a test email",
	// })

	logger.Info("fetched emails from queue", "count", len(emails))
	return emails, nil
}

// ValidateEmailAddress validates email addresses before sending
func (ea *EmailActivities) ValidateEmailAddress(ctx context.Context, email string) (bool, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("validating email address", "email", email)

	// Simple validation - in production, use a proper email validation library
	if len(email) < 3 || len(email) > 254 {
		return false, nil
	}

	// Check for @ symbol
	atIndex := -1
	for i, c := range email {
		if c == '@' {
			if atIndex != -1 {
				return false, nil // Multiple @ symbols
			}
			atIndex = i
		}
	}

	if atIndex == -1 || atIndex == 0 || atIndex == len(email)-1 {
		return false, nil
	}

	return true, nil
}
