package providers

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/email"
)

type Provider interface {
	GetType() email.ProviderType
	Send(ctx context.Context, config *ProviderConfig, message *Message) (string, error)
	ValidateConfig(config *ProviderConfig) error
	TestConnection(ctx context.Context, config *ProviderConfig) error
}

type Registry interface {
	Register(provider Provider)
	Get(providerType email.ProviderType) (Provider, error)
	List() []Provider
}

type ProviderConfig struct {
	Host               string
	Port               int
	Username           string
	Password           string
	APIKey             string
	OAuth2ClientID     string
	OAuth2ClientSecret string
	OAuth2TenantID     string
	AuthType           email.AuthType
	EncryptionType     email.EncryptionType
	TLSPolicy          email.TLSPolicy // * Important: This is required for SMTP authentication
	TimeoutSeconds     int
	MaxConnections     int
	Metadata           map[string]any
}

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

type EmailAddress struct {
	Email string
	Name  string
}

type Attachment struct {
	FileName    string
	ContentType string
	Data        []byte
	ContentID   string
}
