package providers

import (
	"bytes"
	"context"
	"crypto/tls"
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/samber/oops"
	"github.com/wneessen/go-mail"
)

// SMTPProvider implements the Provider interface for SMTP
type SMTPProvider struct{}

// NewSMTPProvider creates a new SMTP provider
func NewSMTPProvider() Provider {
	return &SMTPProvider{}
}

// GetType returns the provider type
func (p *SMTPProvider) GetType() email.ProviderType {
	return email.ProviderTypeSMTP
}

// Send sends an email using SMTP
func (p *SMTPProvider) Send(
	_ context.Context,
	config *ProviderConfig,
	message *Message,
) (string, error) {
	// Create mail client
	client, err := p.createClient(config)
	if err != nil {
		return "", oops.In("smtp_provider").
			Tags("operation", "create_client").
			Time(time.Now()).
			Wrapf(err, "failed to create SMTP client")
	}

	// Create message
	msg := mail.NewMsg()

	// Set from
	if message.From.Name != "" {
		if err = msg.FromFormat(message.From.Name, message.From.Email); err != nil {
			return "", oops.In("smtp_provider").
				Tags("operation", "set_from_format").
				Tags("from_email", message.From.Email).
				Time(time.Now()).
				Wrapf(err, "failed to set from address")
		}
	} else {
		if err = msg.From(message.From.Email); err != nil {
			return "", oops.In("smtp_provider").
				Tags("operation", "set_from").
				Tags("from_email", message.From.Email).
				Time(time.Now()).
				Wrapf(err, "failed to set from address")
		}
	}

	// Set reply-to
	if message.ReplyTo != nil {
		if err = msg.ReplyTo(message.ReplyTo.Email); err != nil {
			return "", oops.In("smtp_provider").
				Tags("operation", "set_reply_to").
				Tags("reply_to_email", message.ReplyTo.Email).
				Time(time.Now()).
				Wrapf(err, "failed to set reply-to")
		}
	}

	// Set recipients
	if err = p.setRecipients(msg, message); err != nil {
		return "", err
	}

	// Set subject
	msg.Subject(message.Subject)

	// Set body
	if message.HTMLBody != "" {
		msg.SetBodyString(mail.TypeTextHTML, message.HTMLBody)
	}
	if message.TextBody != "" {
		msg.AddAlternativeString(mail.TypeTextPlain, message.TextBody)
	}

	// Set priority
	p.setPriority(msg, message.Priority)

	// Set custom headers
	for key, value := range message.Headers {
		// SetGenHeader requires a Header type, so we use SetHeaderPreformatted for custom headers
		msg.SetGenHeaderPreformatted(mail.Header(key), value)
	}

	// Add attachments
	if len(message.Attachments) > 0 {
		if err = p.addAttachments(msg, message.Attachments); err != nil {
			return "", oops.In("smtp_provider").
				Tags("operation", "add_attachments").
				Time(time.Now()).
				Wrapf(err, "failed to add attachments")
		}
	}

	// Send the email
	if err = client.DialAndSend(msg); err != nil {
		return "", oops.In("smtp_provider").
			Tags("operation", "send_email").
			Tags("subject", message.Subject).
			Time(time.Now()).
			Wrapf(err, "failed to send email")
	}

	// Extract message ID
	messageID := msg.GetGenHeader(mail.HeaderMessageID)
	if len(messageID) > 0 {
		return messageID[0], nil
	}

	return "", nil
}

// ValidateConfig validates the SMTP configuration
func (p *SMTPProvider) ValidateConfig(config *ProviderConfig) error {
	if config.Host == "" {
		return oops.In("smtp_provider").
			Tags("operation", "validate_config").
			Time(time.Now()).
			Errorf("SMTP host is required")
	}
	if config.Port <= 0 || config.Port > 65535 {
		return oops.In("smtp_provider").
			Tags("operation", "validate_config").
			Tags("port", strconv.Itoa(config.Port)).
			Time(time.Now()).
			Errorf("invalid SMTP port: %d", config.Port)
	}
	return nil
}

// TestConnection tests the SMTP connection
func (p *SMTPProvider) TestConnection(_ context.Context, config *ProviderConfig) error {
	client, err := p.createClient(config)
	if err != nil {
		return oops.In("smtp_provider").
			Tags("operation", "test_connection_create_client").
			Tags("host", config.Host).
			Tags("port", strconv.Itoa(config.Port)).
			Time(time.Now()).
			Wrapf(err, "failed to create SMTP client")
	}

	// Test the connection
	if err = client.DialAndSend(); err != nil {
		return oops.In("smtp_provider").
			Tags("operation", "test_connection_dial").
			Tags("host", config.Host).
			Tags("port", strconv.Itoa(config.Port)).
			Time(time.Now()).
			Wrapf(err, "connection test failed")
	}

	return nil
}

// createClient creates an SMTP client
func (p *SMTPProvider) createClient(config *ProviderConfig) (*mail.Client, error) {
	options := []mail.Option{
		mail.WithPort(config.Port),
		mail.WithTimeout(time.Duration(config.TimeoutSeconds) * time.Second),
	}

	// Configure authentication
	p.configureAuth(config, &options)

	// Configure encryption
	p.configureEncryption(config, &options)

	// Configure development options
	p.configureDevOptions(config, &options)

	return mail.NewClient(config.Host, options...)
}

// configureAuth configures SMTP authentication
func (p *SMTPProvider) configureAuth(config *ProviderConfig, options *[]mail.Option) {
	if config.Username == "" || config.Password == "" {
		*options = append(*options, mail.WithSMTPAuth(mail.SMTPAuthNoAuth))
	}

	// Set credentials
	*options = append(*options,
		mail.WithUsername(config.Username),
		mail.WithPassword(config.Password),
	)

	// Set auth type
	switch config.AuthType { //nolint:exhaustive // we only explicity handle the 4 auth types others are handled by the auto-discover
	case email.AuthTypePlain:
		*options = append(*options, mail.WithSMTPAuth(mail.SMTPAuthPlain))
	case email.AuthTypeLogin:
		*options = append(*options, mail.WithSMTPAuth(mail.SMTPAuthLogin))
	case email.AuthTypeCRAMMD5:
		*options = append(*options, mail.WithSMTPAuth(mail.SMTPAuthCramMD5))
	default:
		// Use auto-discover to find the best auth method
		*options = append(*options, mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover))
	}
}

// configureEncryption configures SMTP encryption
func (p *SMTPProvider) configureEncryption(config *ProviderConfig, options *[]mail.Option) {
	switch config.EncryptionType { //nolint:exhaustive // default handles none type
	case email.EncryptionTypeSSLTLS:
		*options = append(*options,
			mail.WithSSLPort(true),
			mail.WithTLSPortPolicy(mail.TLSMandatory),
		)
	case email.EncryptionTypeSTARTTLS:
		*options = append(*options, mail.WithTLSPortPolicy(mail.TLSMandatory))
	default:
		*options = append(*options, mail.WithTLSPortPolicy(mail.NoTLS))
	}
}

// configureDevOptions configures development-specific options
func (p *SMTPProvider) configureDevOptions(config *ProviderConfig, options *[]mail.Option) {
	if config.Metadata == nil {
		return
	}

	// Allow insecure TLS for development
	if insecure, ok := config.Metadata["allow_insecure"].(bool); ok && insecure {
		*options = append(*options, mail.WithTLSConfig(&tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // we are in development mode
		}))
	}
}

// setRecipients sets the email recipients
func (p *SMTPProvider) setRecipients(msg *mail.Msg, message *Message) error {
	// To addresses
	toAddresses := make([]string, len(message.To))
	for i, addr := range message.To {
		toAddresses[i] = addr.Email
	}
	if err := msg.To(toAddresses...); err != nil {
		return oops.In("smtp_provider").
			Tags("operation", "set_to_addresses").
			Tags("to_count", strconv.Itoa(len(toAddresses))).
			Time(time.Now()).
			Wrapf(err, "failed to set to addresses")
	}

	// CC addresses
	if len(message.CC) > 0 {
		ccAddresses := make([]string, len(message.CC))
		for i, addr := range message.CC {
			ccAddresses[i] = addr.Email
		}
		if err := msg.Cc(ccAddresses...); err != nil {
			return oops.In("smtp_provider").
				Tags("operation", "set_cc_addresses").
				Tags("cc_count", strconv.Itoa(len(ccAddresses))).
				Time(time.Now()).
				Wrapf(err, "failed to set cc addresses")
		}
	}

	// BCC addresses
	if len(message.BCC) > 0 {
		bccAddresses := make([]string, len(message.BCC))
		for i, addr := range message.BCC {
			bccAddresses[i] = addr.Email
		}
		if err := msg.Bcc(bccAddresses...); err != nil {
			return oops.In("smtp_provider").
				Tags("operation", "set_bcc_addresses").
				Tags("bcc_count", strconv.Itoa(len(bccAddresses))).
				Time(time.Now()).
				Wrapf(err, "failed to set bcc addresses")
		}
	}

	return nil
}

// setPriority sets the email priority
func (p *SMTPProvider) setPriority(msg *mail.Msg, priority email.Priority) {
	switch priority { //nolint:exhaustive // no need to handle all priorities
	case email.PriorityHigh:
		msg.SetImportance(mail.ImportanceHigh)
		msg.SetGenHeaderPreformatted(mail.Header("X-Priority"), "1")
	case email.PriorityLow:
		msg.SetImportance(mail.ImportanceLow)
		msg.SetGenHeaderPreformatted(mail.Header("X-Priority"), "5")
	default:
		msg.SetImportance(mail.ImportanceNormal)
		msg.SetGenHeaderPreformatted(mail.Header("X-Priority"), "3")
	}
}

// addAttachments adds attachments to the email message
func (p *SMTPProvider) addAttachments(msg *mail.Msg, attachments []Attachment) error {
	for i, attachment := range attachments {
		// Validate attachment
		if attachment.FileName == "" {
			return oops.In("smtp_provider").
				Tags("operation", "validate_attachment").
				Tags("attachment_index", strconv.Itoa(i)).
				Time(time.Now()).
				Errorf("attachment %d has empty filename", i)
		}

		if len(attachment.Data) == 0 {
			return oops.In("smtp_provider").
				Tags("operation", "validate_attachment").
				Tags("filename", attachment.FileName).
				Time(time.Now()).
				Errorf("attachment '%s' has no data", attachment.FileName)
		}

		// Add attachment to message
		if attachment.ContentID != "" {
			// Inline attachment (for embedded images, etc.)
			err := msg.EmbedReader(
				attachment.FileName,
				bytes.NewReader(attachment.Data),
				mail.WithFileContentType(mail.ContentType(attachment.ContentType)),
			)
			if err != nil {
				return oops.In("smtp_provider").
					Tags("operation", "add_attachments").
					Tags("attachment_index", strconv.Itoa(i)).
					Time(time.Now()).
					Wrapf(err, "failed to add attachment")
			}
		} else {
			// Regular attachment
			err := msg.AttachReader(attachment.FileName, bytes.NewReader(attachment.Data), mail.WithFileContentType(mail.ContentType(attachment.ContentType)))
			if err != nil {
				return oops.In("smtp_provider").
					Tags("operation", "add_attachments").
					Tags("attachment_index", strconv.Itoa(i)).
					Time(time.Now()).
					Wrapf(err, "failed to add attachment")
			}
		}
	}

	return nil
}
