package providers

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/email"
)

// Provider defines the interface for email providers
type Provider interface {
	// GetType returns the provider type
	GetType() email.ProviderType

	// Send sends an email using this provider
	Send(ctx context.Context, config *ProviderConfig, message *Message) (string, error)

	// ValidateConfig validates the provider configuration
	ValidateConfig(config *ProviderConfig) error

	// TestConnection tests the connection to the provider
	TestConnection(ctx context.Context, config *ProviderConfig) error
}

// ProviderConfig contains provider-specific configuration
type ProviderConfig struct {
	// Common fields
	Host               string
	Port               int
	Username           string
	Password           string
	APIKey             string
	OAuth2ClientID     string
	OAuth2ClientSecret string
	OAuth2TenantID     string

	// Authentication
	AuthType       email.AuthType
	EncryptionType email.EncryptionType
	TLSPolicy      email.TLSPolicy // * Important: This is required for SMTP authentication

	// Additional settings
	TimeoutSeconds int
	MaxConnections int
	Metadata       map[string]any
}

// Message represents an email message
type Message struct {
	From        EmailAddress
	To          []EmailAddress
	CC          []EmailAddress
	BCC         []EmailAddress
	ReplyTo     *EmailAddress
	Subject     string
	HTMLBody    string
	TextBody    string
	Attachments []Attachment
	Headers     map[string]string
	Priority    email.Priority
}

// EmailAddress represents an email address with optional name
type EmailAddress struct {
	Email string
	Name  string
}

// Attachment represents an email attachment
type Attachment struct {
	FileName    string
	ContentType string
	Data        []byte
	ContentID   string // For inline attachments
}

// Registry manages available providers
type Registry interface {
	// Register registers a provider
	Register(provider Provider)

	// Get returns a provider by type
	Get(providerType email.ProviderType) (Provider, error)

	// List returns all registered providers
	List() []Provider
}
