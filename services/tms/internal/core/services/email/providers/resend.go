package providers

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/resend/resend-go/v2"
)

type ResendProvider struct{}

func NewResendProvider() Provider {
	return &ResendProvider{}
}

func (p *ResendProvider) GetType() email.ProviderType {
	return email.ProviderTypeResend
}

func (p *ResendProvider) Send(
	ctx context.Context,
	config *ProviderConfig,
	message *Message,
) (string, error) {
	client := p.createClient(config)

	to := make([]string, 0, len(message.To))
	for _, addr := range message.To {
		to = append(to, addr.Email)
	}

	bcc := make([]string, 0, len(message.BCC))
	for _, addr := range message.BCC {
		bcc = append(bcc, addr.Email)
	}

	cc := make([]string, 0, len(message.CC))
	for _, addr := range message.CC {
		cc = append(cc, addr.Email)
	}

	attachments := make([]*resend.Attachment, 0, len(message.Attachments))
	for _, attachment := range message.Attachments {
		attachments = append(attachments, &resend.Attachment{
			Content:     attachment.Data,
			Filename:    attachment.FileName,
			ContentType: attachment.ContentType,
		})
	}

	params := &resend.SendEmailRequest{
		From:        message.From.Email,
		To:          to,
		Html:        message.HTMLBody,
		Text:        message.TextBody,
		Subject:     message.Subject,
		Headers:     message.Headers,
		Bcc:         bcc,
		Cc:          cc,
		Attachments: attachments,
	}

	if message.ReplyTo != nil {
		params.ReplyTo = message.ReplyTo.Email
	}

	sent, err := client.Emails.SendWithContext(ctx, params)
	if err != nil {
		return "", fmt.Errorf("failed to send email: %w", err)
	}

	return sent.Id, nil
}

func (p *ResendProvider) ValidateConfig(config *ProviderConfig) error {
	if config.APIKey == "" {
		return ErrResendAPIKeyRequired
	}
	return nil
}

func (p *ResendProvider) TestConnection(_ context.Context, config *ProviderConfig) error {
	client := p.createClient(config)

	if _, err := client.ApiKeys.List(); err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	return nil
}

func (p *ResendProvider) createClient(config *ProviderConfig) *resend.Client {
	return resend.NewClient(config.APIKey)
}
