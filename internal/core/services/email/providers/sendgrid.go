package providers

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/samber/oops"
)

// SendGridProvider implements the Provider interface for SendGrid
type SendGridProvider struct{}

// NewSendGridProvider creates a new SendGrid provider
func NewSendGridProvider() Provider {
	return &SendGridProvider{}
}

// GetType returns the provider type
func (p *SendGridProvider) GetType() email.ProviderType {
	return email.ProviderTypeSendGrid
}

// Send sends an email using SendGrid
func (p *SendGridProvider) Send(
	ctx context.Context,
	config *ProviderConfig,
	message *Message,
) (string, error) {
	// TODO: Implement SendGrid API integration
	return "", oops.In("sendgrid_provider").
		Tags("operation", "send").
		Time(time.Now()).
		Errorf("SendGrid provider not yet implemented")
}

// ValidateConfig validates the SendGrid configuration
func (p *SendGridProvider) ValidateConfig(config *ProviderConfig) error {
	if config.APIKey == "" {
		return oops.In("sendgrid_provider").
			Tags("operation", "validate_config").
			Time(time.Now()).
			Errorf("SendGrid API key is required")
	}
	return nil
}

// TestConnection tests the SendGrid connection
func (p *SendGridProvider) TestConnection(ctx context.Context, config *ProviderConfig) error {
	// TODO: Implement SendGrid connection test
	return oops.In("sendgrid_provider").
		Tags("operation", "test_connection").
		Time(time.Now()).
		Errorf("SendGrid connection test not yet implemented")
}
