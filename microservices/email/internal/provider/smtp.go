package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/microservices/email/internal/config"
	"github.com/emoss08/trenova/microservices/email/internal/model"
	"github.com/rotisserie/eris"
	mail "github.com/wneessen/go-mail"
)

// SMTPProvider implements the EmailProvider interface for SMTP
type SMTPProvider struct {
	cfg             *config.SMTPConfig
	templateService model.TemplateRenderer
}

// NewSMTPProvider creates a new SMTP provider
func NewSMTPProvider(cfg *config.AppConfig, templateService model.TemplateRenderer) *SMTPProvider {
	return &SMTPProvider{
		cfg:             &cfg.SMTP,
		templateService: templateService,
	}
}

// Send sends an email using go-mail
func (p *SMTPProvider) Send(ctx context.Context, email *model.Email) error {
	if !p.IsConfigured() {
		return eris.New("SMTP provider is not configured")
	}

	// Create a new go-mail message
	msg := mail.NewMsg()
	if msg == nil {
		return eris.New("failed to create mail message")
	}

	// Set the message headers
	if err := msg.From(fmt.Sprintf("%s <%s>", p.cfg.FromName, p.cfg.From)); err != nil {
		return eris.Wrap(err, "failed to set From address")
	}

	// Add To recipients
	for _, to := range email.To {
		if err := msg.To(to); err != nil {
			return eris.Wrapf(err, "failed to add To recipient: %s", to)
		}
	}

	// Add CC recipients if present
	for _, cc := range email.Cc {
		if err := msg.Cc(cc); err != nil {
			return eris.Wrapf(err, "failed to add CC recipient: %s", cc)
		}
	}

	// Add BCC recipients if present
	for _, bcc := range email.Bcc {
		if err := msg.Bcc(bcc); err != nil {
			return eris.Wrapf(err, "failed to add BCC recipient: %s", bcc)
		}
	}

	// Set subject
	msg.Subject(email.Subject)

	// Set HTML body - use template renderer if available
	var htmlBody string
	if p.templateService != nil {
		renderedBody, err := email.RenderHTMLBody(p.templateService)
		if err != nil {
			// Log the error but continue with the fallback method
			htmlBody = email.GetHTMLBody()
		} else {
			htmlBody = renderedBody
		}
	} else {
		htmlBody = email.GetHTMLBody()
	}
	msg.SetBodyString(mail.TypeTextHTML, htmlBody)

	// Add attachments if any
	if data, ok := email.Data.(map[string]any); ok {
		if attachments, aOk := data["attachments"].([]model.EmailAttachment); aOk {
			for _, attachment := range attachments {
				// Decode base64 content
				content, err := base64.StdEncoding.DecodeString(attachment.Content)
				if err != nil {
					return eris.Wrapf(
						err,
						"failed to decode attachment content for %s",
						attachment.Filename,
					)
				}

				// Add attachment to the email using a reader
				if err = msg.AttachReader(attachment.Filename, strings.NewReader(string(content)), func(f *mail.File) {
					f.ContentType = mail.ContentType(attachment.ContentType)
				}); err != nil {
					return eris.Wrapf(err, "failed to add attachment: %s", attachment.Filename)
				}
			}
		}
	}

	// Determine TLS policy
	var tlsPolicy mail.TLSPolicy
	switch p.cfg.TLSPolicy {
	case config.TLSMandatory:
		tlsPolicy = mail.TLSMandatory
	case config.TLSOpportunistic:
		tlsPolicy = mail.TLSOpportunistic
	case config.TLSNone:
		tlsPolicy = mail.NoTLS
	default:
		tlsPolicy = mail.TLSMandatory
	}

	// Create a mail client
	client, err := mail.NewClient(p.cfg.Host,
		mail.WithPort(p.cfg.Port),
		mail.WithUsername(p.cfg.User),
		mail.WithPassword(p.cfg.Password),
		mail.WithTimeout(p.cfg.Timeout),
		mail.WithTLSPolicy(tlsPolicy),
		mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover),
	)
	if err != nil {
		return eris.Wrap(err, "failed to create mail client")
	}

	// Send the email with context
	if err = client.DialAndSendWithContext(ctx, msg); err != nil {
		return eris.Wrap(err, "failed to send email")
	}

	return nil
}

// GetName returns the name of the provider
func (p *SMTPProvider) GetName() string {
	return "SMTP"
}

// IsConfigured returns true if the provider is configured
func (p *SMTPProvider) IsConfigured() bool {
	return p.cfg.Host != "" && p.cfg.From != ""
}
