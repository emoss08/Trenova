/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package email

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/microservices/email/internal/model"
	"github.com/emoss08/trenova/microservices/email/internal/provider"
	"github.com/google/uuid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog/log"
)

var (
	ErrInvalidPayloadFormat = eris.New("invalid email payload format")
	ErrTemplateRequired     = eris.New("template is required")
	ErrSubjectRequired      = eris.New("subject is required")
	ErrFileNameRequired     = eris.New("filename is required")
	ErrContentRequired      = eris.New("content is required")
	ErrContentTypeRequired  = eris.New("contentType is required")
)

// SenderService handles sending emails through providers
type SenderService struct {
	templateService *TemplateService
	providerFactory *provider.Factory
	defaultProvider provider.Type
}

// NewSenderService creates a new sender service
func NewSenderService(templates *TemplateService, factory *provider.Factory) *SenderService {
	return &SenderService{
		templateService: templates,
		providerFactory: factory,
		defaultProvider: provider.TypeSMTP, // Default to SMTP
	}
}

// SetDefaultProvider sets the default provider to use
func (s *SenderService) SetDefaultProvider(providerType provider.Type) {
	s.defaultProvider = providerType
}

// SendEmail sends an email using the configured provider
func (s *SenderService) SendEmail(
	ctx context.Context,
	payload *model.EmailPayload,
	tenantID string,
) (string, error) {
	// Generate a unique ID for the email
	emailID := uuid.New().String()

	// Create the email record
	email := &model.Email{
		ID:        emailID,
		TenantID:  tenantID,
		Status:    model.EmailStatusPending,
		Template:  payload.Template,
		Subject:   payload.Subject,
		To:        payload.To,
		Cc:        payload.Cc,
		Bcc:       payload.Bcc,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Data:      payload.Data,
	}

	// Send the email
	if err := s.sendViaProvider(ctx, email); err != nil {
		return emailID, eris.Wrap(err, "failed to send email")
	}

	return emailID, nil
}

// SendEmailFromMessage sends an email from a RabbitMQ message
func (s *SenderService) SendEmailFromMessage(ctx context.Context, msg *model.Message) error {
	// Extract the email payload from the message
	payload, err := s.extractEmailPayload(msg)
	if err != nil {
		return err
	}

	// Validate we have at least one recipient
	if len(payload.To) == 0 {
		return eris.New("no valid recipient email addresses provided")
	}

	// Send the email
	emailID, err := s.SendEmail(ctx, payload, msg.TenantID)
	if err != nil {
		return eris.Wrapf(err, "failed to send email for message %s", msg.ID)
	}

	log.Info().
		Str("emailID", emailID).
		Str("messageID", msg.ID).
		Str("tenantID", msg.TenantID).
		Msg("Email sent successfully")

	return nil
}

// extractEmailPayload extracts and validates an EmailPayload from a Message
func (s *SenderService) extractEmailPayload(msg *model.Message) (*model.EmailPayload, error) {
	// Try direct type assertion first
	if payload, ok := msg.Payload.(*model.EmailPayload); ok {
		return s.sanitizePayload(payload), nil
	}

	// Try to convert from map
	mapPayload, ok := msg.Payload.(map[string]any)
	if !ok {
		return nil, ErrInvalidPayloadFormat
	}

	template, ok := mapPayload["template"].(string)
	if !ok {
		return nil, ErrTemplateRequired
	}

	subject, ok := mapPayload["subject"].(string)
	if !ok {
		return nil, ErrSubjectRequired
	}

	// Create a new payload from the map
	payload := &model.EmailPayload{
		Template: template,
		Subject:  subject,
	}

	// Process recipient lists
	payload.To = s.extractEmailList(mapPayload, "to")
	payload.Cc = s.extractEmailList(mapPayload, "cc")
	payload.Bcc = s.extractEmailList(mapPayload, "bcc")

	// Handle data
	if data, dataOk := mapPayload["data"].(map[string]any); dataOk {
		payload.Data = data
	}

	// Handle attachments
	if attachments, attachmentsOk := mapPayload["attachments"].([]any); attachmentsOk {
		var err error
		payload.Attachments, err = s.extractAttachments(attachments)
		if err != nil {
			return nil, err
		}
	}

	return payload, nil
}

// extractEmailList extracts and filters email lists from payload maps
func (s *SenderService) extractEmailList(mapPayload map[string]any, key string) []string {
	var validEmails []string

	if list, ok := mapPayload[key].([]any); ok {
		for _, v := range list {
			if email, emailOk := v.(string); emailOk && email != "" {
				validEmails = append(validEmails, email)
			}
		}
	}

	return validEmails
}

// sanitizePayload ensures the payload is properly formatted
func (s *SenderService) sanitizePayload(payload *model.EmailPayload) *model.EmailPayload {
	if payload.To != nil {
		validEmails := make([]string, 0, len(payload.To))
		for _, email := range payload.To {
			if email != "" {
				validEmails = append(validEmails, email)
			}
		}
		payload.To = validEmails
	}
	return payload
}

// sendViaProvider sends an email via the configured provider
func (s *SenderService) sendViaProvider(ctx context.Context, email *model.Email) error {
	// Prepare email content if it uses a template
	if email.Template != "" && email.Template != "custom" {
		// Check if we have template data
		if email.Data == nil {
			email.Data = map[string]any{
				"Subject": email.Subject,
				"Body":    fmt.Sprintf("This is a %s email from Trenova.", email.Template),
				"Year":    time.Now().Year(),
			}
		}

		// Add year to data if not present
		dataMap, ok := email.Data.(map[string]any)
		if ok {
			if _, hasYear := dataMap["Year"]; !hasYear {
				dataMap["Year"] = time.Now().Year()
				email.Data = dataMap
			}
		}
	}

	// Get the email provider
	emailProvider, err := s.providerFactory.GetProvider(s.defaultProvider)
	if err != nil {
		return eris.Wrap(err, "failed to get email provider")
	}

	// Send the email
	if err = emailProvider.Send(ctx, email); err != nil {
		return eris.Wrapf(err, "failed to send email via %s", emailProvider.GetName())
	}

	return nil
}

// extractAttachments processes attachment data from a slice of interfaces
func (s *SenderService) extractAttachments(attachments []any) ([]model.EmailAttachment, error) {
	result := make([]model.EmailAttachment, 0, len(attachments))

	for _, v := range attachments {
		attachment, attachmentOk := v.(map[string]any)
		if !attachmentOk {
			continue
		}

		fileName, fileNameOk := attachment["filename"].(string)
		if !fileNameOk {
			return nil, ErrFileNameRequired
		}

		content, contentOk := attachment["content"].(string)
		if !contentOk {
			return nil, ErrContentRequired
		}

		contentType, contentTypeOk := attachment["contentType"].(string)
		if !contentTypeOk {
			return nil, ErrContentTypeRequired
		}

		result = append(result, model.EmailAttachment{
			Filename:    fileName,
			Content:     content,
			ContentType: contentType,
		})
	}

	return result, nil
}
