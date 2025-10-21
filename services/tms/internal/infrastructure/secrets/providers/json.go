package providers

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/ports"
)

var _ ports.SecretProvider = (*JSONProvider)(nil)

type JSONProvider struct {
	filePath string
	secrets  map[string]any
	mu       sync.RWMutex
}

func NewJSONProvider(config map[string]string) (*JSONProvider, error) {
	filePath := config["file_path"]
	if filePath == "" {
		return nil, ErrJSONProviderFilePathRequired
	}

	provider := &JSONProvider{
		filePath: filePath,
	}

	if err := provider.loadSecrets(); err != nil {
		return nil, err
	}

	return provider, nil
}

func (p *JSONProvider) loadSecrets() error {
	data, err := os.ReadFile(p.filePath)
	if err != nil {
		return fmt.Errorf("failed to read JSON secrets file %s: %w", p.filePath, err)
	}

	var secrets map[string]any
	if err = sonic.Unmarshal(data, &secrets); err != nil {
		return fmt.Errorf("failed to parse JSON secrets: %w", err)
	}

	p.mu.Lock()
	p.secrets = secrets
	p.mu.Unlock()

	return nil
}

func (p *JSONProvider) GetSecret(_ context.Context, key string) (string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	value := p.getNestedValue(key)
	if value == nil {
		return "", fmt.Errorf("secret not found: %s", key)
	}

	switch v := value.(type) {
	case string:
		return v, nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(v), nil
	default:
		jsonBytes, err := sonic.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("failed to marshal secret value: %w", err)
		}
		return string(jsonBytes), nil
	}
}

func (p *JSONProvider) GetSecrets(ctx context.Context, keys []string) (map[string]string, error) {
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

func (p *JSONProvider) GetBinarySecret(ctx context.Context, key string) ([]byte, error) {
	value, err := p.GetSecret(ctx, key)
	if err != nil {
		return nil, err
	}
	return []byte(value), nil
}

func (p *JSONProvider) Close() error {
	return nil
}

func (p *JSONProvider) Reload() error {
	return p.loadSecrets()
}

func (p *JSONProvider) getNestedValue(key string) any {
	parts := strings.Split(key, ".")
	current := p.secrets

	for i, part := range parts {
		if current == nil {
			return nil
		}

		if i == len(parts)-1 {
			return current[part]
		}

		next, ok := current[part]
		if !ok {
			return nil
		}

		nextMap, ok := next.(map[string]any)
		if !ok {
			return nil
		}

		current = nextMap
	}

	return nil
}
