/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package email

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/external/temporaljobs/payloads"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rs/zerolog"
	"go.temporal.io/sdk/activity"
)

// ActivityProvider provides email activities with dependencies
type ActivityProvider struct {
	logger       *zerolog.Logger
	emailService services.EmailService
}

// NewActivityProvider creates a new email activity provider
func NewActivityProvider(
	logger *logger.Logger,
	emailService services.EmailService,
) *ActivityProvider {
	log := logger.With().
		Str("component", "temporal-email-activities").
		Logger()

	return &ActivityProvider{
		logger:       &log,
		emailService: emailService,
	}
}

// SendEmailActivity sends an email
func SendEmailActivity(ctx context.Context, payload *payloads.EmailPayload) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("sending email activity",
		"to", payload.To,
		"subject", payload.Subject,
	)

	activity.RecordHeartbeat(ctx, "sending email")

	// In production, this would call the actual email service
	result := "Email sent successfully to " + payload.To[0]
	return result, nil
}

// SendEmailWithService sends an email using the injected email service
func (p *ActivityProvider) SendEmailWithService(
	ctx context.Context,
	payload *payloads.EmailPayload,
) (string, error) {
	if p.emailService == nil {
		return SendEmailActivity(ctx, payload)
	}

	activity.RecordHeartbeat(ctx, "preparing email")

	emailReq := &services.SendEmailRequest{
		OrganizationID: payload.OrganizationID,
		BusinessUnitID: payload.BusinessUnitID,
		To:             payload.To,
		Subject:        payload.Subject,
		TextBody:       payload.Body,
		HTMLBody:       payload.BodyHTML,
	}

	if payload.UserID != "" {
		emailReq.UserID = payload.UserID
	}

	activity.RecordHeartbeat(ctx, "sending email via service")
	response, err := p.emailService.SendEmail(ctx, emailReq)
	if err != nil {
		return "", err
	}

	return "Email sent with ID: " + response.MessageID, nil
}

// FetchEmailsFromQueueActivity fetches emails from queue
func FetchEmailsFromQueueActivity(
	ctx context.Context,
	payload *payloads.BasePayload,
) ([]payloads.EmailPayload, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("fetching emails from queue",
		"organizationId", payload.OrganizationID.String(),
	)

	activity.RecordHeartbeat(ctx, "fetching emails")

	// Return empty batch for now - would fetch from queue in production
	return []payloads.EmailPayload{}, nil
}

// ActivityDefinition defines an activity with its configuration
type ActivityDefinition struct {
	Name        string
	Fn          any
	Description string
}

// RegisterActivities registers all email-related activities
func RegisterActivities() []ActivityDefinition {
	return []ActivityDefinition{
		{
			Name:        "SendEmailActivity",
			Fn:          SendEmailActivity,
			Description: "Sends an individual email",
		},
		{
			Name:        "FetchEmailsFromQueueActivity",
			Fn:          FetchEmailsFromQueueActivity,
			Description: "Fetches emails from queue for batch processing",
		},
	}
}
