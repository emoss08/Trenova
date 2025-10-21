package providers

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/wneessen/go-mail"
)

type SMTPProvider struct{}

func NewSMTPProvider() Provider {
	return &SMTPProvider{}
}

func (p *SMTPProvider) GetType() email.ProviderType {
	return email.ProviderTypeSMTP
}

func (p *SMTPProvider) Send(
	_ context.Context,
	config *ProviderConfig,
	message *Message,
) (string, error) {
	client, err := p.createClient(config)
	if err != nil {
		return "", fmt.Errorf("failed to create SMTP client: %w", err)
	}

	msg := mail.NewMsg()

	if message.From.Name != "" {
		if err = msg.FromFormat(message.From.Name, message.From.Email); err != nil {
			return "", fmt.Errorf("failed to set from address: %w", err)
		}
	} else {
		if err = msg.From(message.From.Email); err != nil {
			return "", fmt.Errorf("failed to set from address: %w", err)
		}
	}

	if message.ReplyTo != nil {
		if err = msg.ReplyTo(message.ReplyTo.Email); err != nil {
			return "", fmt.Errorf("failed to set reply-to: %w", err)
		}
	}

	if err = p.setRecipients(msg, message); err != nil {
		return "", err
	}

	msg.Subject(message.Subject)

	if message.HTMLBody != "" {
		msg.SetBodyString(mail.TypeTextHTML, message.HTMLBody)
	}
	if message.TextBody != "" {
		msg.AddAlternativeString(mail.TypeTextPlain, message.TextBody)
	}

	p.setPriority(msg, message.Priority)

	for key, value := range message.Headers {
		msg.SetGenHeaderPreformatted(mail.Header(key), value)
	}

	if len(message.Attachments) > 0 {
		if err = p.addAttachments(msg, message.Attachments); err != nil {
			return "", fmt.Errorf("failed to add attachments: %w", err)
		}
	}

	if err = client.DialAndSend(msg); err != nil {
		return "", fmt.Errorf("failed to send email: %w", err)
	}

	messageID := msg.GetGenHeader(mail.HeaderMessageID)
	if len(messageID) > 0 {
		return messageID[0], nil
	}

	return "", nil
}

func (p *SMTPProvider) ValidateConfig(config *ProviderConfig) error {
	if config.Host == "" {
		return ErrSMTPHostRequired
	}
	if config.Port <= 0 || config.Port > 65535 {
		return fmt.Errorf("invalid SMTP port: %d", config.Port)
	}
	return nil
}

func (p *SMTPProvider) TestConnection(_ context.Context, config *ProviderConfig) error {
	client, err := p.createClient(config)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}

	if err = client.DialAndSend(); err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	return nil
}

func (p *SMTPProvider) createClient(config *ProviderConfig) (*mail.Client, error) {
	options := []mail.Option{
		mail.WithPort(config.Port),
		mail.WithTimeout(time.Duration(config.TimeoutSeconds) * time.Second),
	}

	p.configureAuth(config, &options)

	p.configureEncryption(config, &options)

	p.configureDevOptions(config, &options)

	return mail.NewClient(config.Host, options...)
}

func (p *SMTPProvider) configureAuth(config *ProviderConfig, options *[]mail.Option) {
	if config.Username == "" || config.Password == "" {
		*options = append(*options, mail.WithSMTPAuth(mail.SMTPAuthNoAuth))
	}

	*options = append(*options,
		mail.WithUsername(config.Username),
		mail.WithPassword(config.Password),
	)

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

	// ! For localhost remove the authentication and TLS policy (this should be mailhog)
	if p.isLocalhost(config) {
		*options = append(*options,
			mail.WithSMTPAuth(mail.SMTPAuthNoAuth),
			mail.WithTLSPolicy(mail.NoTLS),
		)
		return
	}
}

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

func (p *SMTPProvider) configureDevOptions(config *ProviderConfig, options *[]mail.Option) {
	if config.Metadata == nil {
		return
	}

	if insecure, ok := config.Metadata["allow_insecure"].(bool); ok && insecure {
		*options = append(*options, mail.WithTLSConfig(&tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // we are in development mode
		}))
	}
}

func (p *SMTPProvider) setRecipients(msg *mail.Msg, message *Message) error {
	toAddresses := make([]string, len(message.To))
	for i, addr := range message.To {
		toAddresses[i] = addr.Email
	}
	if err := msg.To(toAddresses...); err != nil {
		return fmt.Errorf("failed to set to addresses: %w", err)
	}

	// CC addresses
	if len(message.CC) > 0 {
		ccAddresses := make([]string, len(message.CC))
		for i, addr := range message.CC {
			ccAddresses[i] = addr.Email
		}
		if err := msg.Cc(ccAddresses...); err != nil {
			return fmt.Errorf("failed to set cc addresses: %w", err)
		}
	}

	// BCC addresses
	if len(message.BCC) > 0 {
		bccAddresses := make([]string, len(message.BCC))
		for i, addr := range message.BCC {
			bccAddresses[i] = addr.Email
		}
		if err := msg.Bcc(bccAddresses...); err != nil {
			return fmt.Errorf("failed to set bcc addresses: %w", err)
		}
	}

	return nil
}

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

func (p *SMTPProvider) addAttachments(msg *mail.Msg, attachments []Attachment) error {
	for i, attachment := range attachments {
		if attachment.FileName == "" {
			return fmt.Errorf("attachment %d has empty filename", i)
		}

		if len(attachment.Data) == 0 {
			return fmt.Errorf("attachment '%s' has no data", attachment.FileName)
		}

		if attachment.ContentID != "" {
			err := msg.EmbedReader(
				attachment.FileName,
				bytes.NewReader(attachment.Data),
				mail.WithFileContentType(mail.ContentType(attachment.ContentType)),
			)
			if err != nil {
				return fmt.Errorf("failed to add attachment: %w", err)
			}
		} else {
			err := msg.AttachReader(attachment.FileName, bytes.NewReader(attachment.Data), mail.WithFileContentType(mail.ContentType(attachment.ContentType)))
			if err != nil {
				return fmt.Errorf("failed to add attachment: %w", err)
			}
		}
	}

	return nil
}

func (p *SMTPProvider) isLocalhost(config *ProviderConfig) bool {
	if config.Host == "localhost" || config.Host == "127.0.0.1" {
		return true
	}

	return false
}
