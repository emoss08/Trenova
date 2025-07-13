package email

import (
	"time"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/services/email/providers"
	"github.com/emoss08/trenova/internal/infrastructure/encryption"
	"github.com/samber/oops"
	"go.uber.org/fx"
)

// ClientFactory creates mail clients based on provider configuration
type ClientFactory interface {
	GetProvider(profile *email.Profile) (providers.Provider, error)
	BuildConfig(profile *email.Profile) (*providers.ProviderConfig, error)
}

type clientFactory struct {
	encryptionService encryption.Service
	providerRegistry  providers.Registry
}

type ClientFactoryParams struct {
	fx.In

	EncryptionService encryption.Service
	ProviderRegistry  providers.Registry
}

// NewClientFactory creates a new mail client factory
func NewClientFactory(p ClientFactoryParams) ClientFactory {
	return &clientFactory{
		encryptionService: p.EncryptionService,
		providerRegistry:  p.ProviderRegistry,
	}
}

// GetProvider returns the appropriate provider for the profile
func (f *clientFactory) GetProvider(profile *email.Profile) (providers.Provider, error) {
	return f.providerRegistry.Get(profile.ProviderType)
}

// BuildConfig builds provider configuration from email profile
func (f *clientFactory) BuildConfig(profile *email.Profile) (*providers.ProviderConfig, error) {
	config := &providers.ProviderConfig{
		Host:           profile.Host,
		Port:           profile.Port,
		Username:       profile.Username,
		AuthType:       profile.AuthType,
		EncryptionType: profile.EncryptionType,
		TimeoutSeconds: profile.TimeoutSeconds,
		MaxConnections: profile.MaxConnections,
		Metadata:       profile.Metadata,
		OAuth2ClientID: profile.OAuth2ClientID,
		OAuth2TenantID: profile.OAuth2TenantID,
	}

	// Decrypt password if present
	if profile.EncryptedPassword != "" {
		password, err := f.encryptionService.Decrypt(profile.EncryptedPassword)
		if err != nil {
			return nil, oops.In("client_factory").
				Tags("operation", "decrypt_password").
				Tags("profile_id", profile.ID.String()).
				Time(time.Now()).
				Wrapf(err, "failed to decrypt password")
		}
		config.Password = password
	}

	// Decrypt API key if present
	if profile.EncryptedAPIKey != "" {
		apiKey, err := f.encryptionService.Decrypt(profile.EncryptedAPIKey)
		if err != nil {
			return nil, oops.In("client_factory").
				Tags("operation", "decrypt_api_key").
				Tags("profile_id", profile.ID.String()).
				Time(time.Now()).
				Wrapf(err, "failed to decrypt API key")
		}
		config.APIKey = apiKey
	}

	// Decrypt OAuth2 client secret if present
	if profile.OAuth2ClientSecret != "" {
		secret, err := f.encryptionService.Decrypt(profile.OAuth2ClientSecret)
		if err != nil {
			return nil, oops.In("client_factory").
				Tags("operation", "decrypt_oauth_secret").
				Tags("profile_id", profile.ID.String()).
				Time(time.Now()).
				Wrapf(err, "failed to decrypt OAuth2 client secret")
		}
		config.OAuth2ClientSecret = secret
	}

	return config, nil
}
