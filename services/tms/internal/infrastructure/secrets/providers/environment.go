package providers

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/emoss08/trenova/internal/core/ports"
)

var _ ports.SecretProvider = (*EnvironmentProvider)(nil)

type EnvironmentProvider struct {
	prefix string
}

func NewEnvironmentProvider(config map[string]string) *EnvironmentProvider {
	prefix := config["prefix"]
	if prefix == "" {
		prefix = "TRENOVA_SECRET"
	}
	return &EnvironmentProvider{
		prefix: prefix,
	}
}

func (p *EnvironmentProvider) GetSecret(_ context.Context, key string) (string, error) {
	envKey := p.formatKey(key)

	value := os.Getenv(envKey)
	if value == "" {
		value = os.Getenv(key)
		if value == "" {
			return "", fmt.Errorf("secret not found in environment: %s or %s", envKey, key)
		}
	}

	return value, nil
}

func (p *EnvironmentProvider) GetSecrets(
	ctx context.Context,
	keys []string,
) (map[string]string, error) {
	secrets := make(map[string]string)
	var errors []string

	for _, key := range keys {
		value, err := p.GetSecret(ctx, key)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", key, err))
			continue
		}
		secrets[key] = value
	}

	if len(errors) > 0 {
		return secrets, fmt.Errorf("failed to get some secrets: %s", strings.Join(errors, "; "))
	}

	return secrets, nil
}

func (p *EnvironmentProvider) GetBinarySecret(ctx context.Context, key string) ([]byte, error) {
	value, err := p.GetSecret(ctx, key)
	if err != nil {
		return nil, err
	}
	return []byte(value), nil
}

func (p *EnvironmentProvider) Close() error {
	return nil
}

func (p *EnvironmentProvider) formatKey(key string) string {
	formatted := strings.ReplaceAll(key, ".", "_")
	formatted = strings.ReplaceAll(formatted, "-", "_")
	formatted = strings.ToUpper(formatted)

	if p.prefix != "" {
		return fmt.Sprintf("%s_%s", p.prefix, formatted)
	}

	return formatted
}
