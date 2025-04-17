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
func (s *SenderService) SendEmail(ctx context.Context, payload *model.EmailPayload, tenantID string) (string, error) {
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
	payload, ok := msg.Payload.(*model.EmailPayload)
	if !ok {
		// Try to convert the payload to an EmailPayload
		mapPayload, ok := msg.Payload.(map[string]any)
		if !ok {
			return eris.New("invalid email payload format")
		}

		// Create a new payload from the map
		payload = &model.EmailPayload{
			Template: mapPayload["template"].(string),
			Subject:  mapPayload["subject"].(string),
		}

		// Handle recipient lists
		if to, ok := mapPayload["to"].([]any); ok {
			payload.To = make([]string, len(to))
			for i, v := range to {
				if email, ok := v.(string); ok && email != "" {
					payload.To[i] = email
				}
			}
			// Filter out empty strings
			validEmails := make([]string, 0, len(payload.To))
			for _, email := range payload.To {
				if email != "" {
					validEmails = append(validEmails, email)
				}
			}
			payload.To = validEmails
		}

		if cc, ok := mapPayload["cc"].([]any); ok {
			payload.Cc = make([]string, len(cc))
			for i, v := range cc {
				if email, ok := v.(string); ok && email != "" {
					payload.Cc[i] = email
				}
			}
			// Filter out empty strings
			validEmails := make([]string, 0, len(payload.Cc))
			for _, email := range payload.Cc {
				if email != "" {
					validEmails = append(validEmails, email)
				}
			}
			payload.Cc = validEmails
		}

		if bcc, ok := mapPayload["bcc"].([]any); ok {
			payload.Bcc = make([]string, len(bcc))
			for i, v := range bcc {
				if email, ok := v.(string); ok && email != "" {
					payload.Bcc[i] = email
				}
			}
			// Filter out empty strings
			validEmails := make([]string, 0, len(payload.Bcc))
			for _, email := range payload.Bcc {
				if email != "" {
					validEmails = append(validEmails, email)
				}
			}
			payload.Bcc = validEmails
		}

		// Handle data and attachments
		if data, ok := mapPayload["data"].(map[string]any); ok {
			payload.Data = data
		}

		if attachments, ok := mapPayload["attachments"].([]any); ok {
			payload.Attachments = make([]model.EmailAttachment, len(attachments))
			for i, v := range attachments {
				attachment := v.(map[string]any)
				payload.Attachments[i] = model.EmailAttachment{
					Filename:    attachment["filename"].(string),
					Content:     attachment["content"].(string),
					ContentType: attachment["contentType"].(string),
				}
			}
		}
	} else {
		// Filter out empty emails from the payload if it's already the correct type
		if payload.To != nil {
			validEmails := make([]string, 0, len(payload.To))
			for _, email := range payload.To {
				if email != "" {
					validEmails = append(validEmails, email)
				}
			}
			payload.To = validEmails
		}
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
