package ports

import (
	"context"
	"time"
)

type SecretProviderType string

const (
	SecretProviderTypeEnvironment SecretProviderType = "env"
	SecretProviderTypeFile        SecretProviderType = "file"
	SecretProviderTypeJSON        SecretProviderType = "json"
	SecretProviderTypeAWS         SecretProviderType = "aws"
	SecretProviderTypeVault       SecretProviderType = "vault"
	SecretProviderTypeAzure       SecretProviderType = "azure"
	SecretProviderTypeGCP         SecretProviderType = "gcp"
	SecretProviderTypeKubernetes  SecretProviderType = "kubernetes"
)

type SecretProvider interface {
	GetSecret(ctx context.Context, key string) (string, error)
	GetSecrets(ctx context.Context, keys []string) (map[string]string, error)
	GetBinarySecret(ctx context.Context, key string) ([]byte, error)
	Close() error
}

type Config struct {
	Provider       SecretProviderType `mapstructure:"provider"       validate:"required,oneof=env file json aws vault azure gcp kubernetes"`
	CacheTTL       time.Duration      `mapstructure:"cache_ttl"`
	RetryAttempts  int                `mapstructure:"retry_attempts"`
	RetryDelay     time.Duration      `mapstructure:"retry_delay"`
	ProviderConfig map[string]string  `mapstructure:"config"`
}
