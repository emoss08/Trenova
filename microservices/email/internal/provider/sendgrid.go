package provider

import (
	"context"
	"encoding/base64"

	"github.com/emoss08/trenova/microservices/email/internal/config"
	"github.com/emoss08/trenova/microservices/email/internal/model"
	"github.com/rotisserie/eris"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// SendGridProvider implements the EmailProvider interface for SendGrid
type SendGridProvider struct {
	cfg *config.SendGridConfig
}

// NewSendGridProvider creates a new SendGrid provider
func NewSendGridProvider(cfg *config.AppConfig) *SendGridProvider {
	return &SendGridProvider{
		cfg: &cfg.SendGrid,
	}
}

// Send sends an email using SendGrid
func (p *SendGridProvider) Send(ctx context.Context, email *model.Email) error {
	if !p.IsConfigured() {
		return eris.New("SendGrid provider is not configured")
	}

	// Create new SendGrid message
	message := mail.NewV3Mail()

	// Set from address
	from := mail.NewEmail(p.cfg.Name, p.cfg.From)
	message.SetFrom(from)

	// Set subject
	message.Subject = email.Subject

	// Add recipients
	personalization := mail.NewPersonalization()
	for _, to := range email.To {
		personalization.AddTos(mail.NewEmail("", to))
	}

	// Add CC recipients if present
	for _, cc := range email.Cc {
		personalization.AddCCs(mail.NewEmail("", cc))
	}

	// Add BCC recipients if present
	for _, bcc := range email.Bcc {
		personalization.AddBCCs(mail.NewEmail("", bcc))
	}

	message.AddPersonalizations(personalization)

	// Add HTML content
	htmlContent := mail.NewContent("text/html", email.GetHTMLBody())
	message.AddContent(htmlContent)

	// Add attachments if any
	if data, ok := email.Data.(map[string]any); ok {
		if attachments, aOk := data["attachments"].([]model.EmailAttachment); aOk {
			for _, attachment := range attachments {
				// Decode base64 content
				content, err := base64.StdEncoding.DecodeString(attachment.Content)
				if err != nil {
					return eris.Wrapf(err, "failed to decode attachment content for %s", attachment.Filename)
				}

				// Create attachment
				sgAttachment := mail.NewAttachment()
				sgAttachment.SetContent(base64.StdEncoding.EncodeToString(content))
				sgAttachment.SetType(attachment.ContentType)
				sgAttachment.SetFilename(attachment.Filename)
				sgAttachment.SetDisposition("attachment")

				message.AddAttachment(sgAttachment)
			}
		}
	}

	// Send the email
	client := sendgrid.NewSendClient(p.cfg.APIKey)
	response, err := client.Send(message)
	if err != nil {
		return eris.Wrap(err, "failed to send email via SendGrid")
	}

	// Check for errors in the response
	if response.StatusCode >= 400 {
		return eris.Errorf("SendGrid API error: %d - %s", response.StatusCode, response.Body)
	}

	return nil
}

// GetName returns the name of the provider
func (p *SendGridProvider) GetName() string {
	return "SendGrid"
}

// IsConfigured returns true if the provider is configured
func (p *SendGridProvider) IsConfigured() bool {
	return p.cfg.APIKey != "" && p.cfg.From != ""
}
