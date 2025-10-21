package secrets

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/infrastructure/secrets/providers"
)

type Manager struct {
	provider ports.SecretProvider
	cache    *secretCache
	config   *ports.Config
}

type secretCache struct {
	mu    sync.RWMutex
	items map[string]*cachedSecret
	ttl   time.Duration
}

type cachedSecret struct {
	value     string
	binary    []byte
	isBinary  bool
	expiresAt time.Time
}

func NewManager(ctx context.Context, config *ports.Config) (*Manager, error) {
	provider, err := createProvider(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create secret provider: %w", err)
	}

	cacheTTL := config.CacheTTL
	if cacheTTL == 0 {
		cacheTTL = 5 * time.Minute
	}

	return &Manager{
		provider: provider,
		cache: &secretCache{
			items: make(map[string]*cachedSecret),
			ttl:   cacheTTL,
		},
		config: config,
	}, nil
}

func (m *Manager) GetSecret(ctx context.Context, key string) (value string, err error) {
	if cached := m.cache.get(key); cached != nil && !cached.isBinary {
		return cached.value, nil
	}

	for attempt := 0; attempt <= m.config.RetryAttempts; attempt++ {
		value, err = m.provider.GetSecret(ctx, key)
		if err == nil {
			break
		}

		if attempt < m.config.RetryAttempts {
			time.Sleep(m.config.RetryDelay)
		}
	}

	if err != nil {
		return "", fmt.Errorf(
			"failed to get secret %s after %d attempts: %w",
			key,
			m.config.RetryAttempts+1,
			err,
		)
	}

	m.cache.set(key, value, false)

	return value, nil
}

func (m *Manager) GetBinarySecret(ctx context.Context, key string) ([]byte, error) {
	if cached := m.cache.get(key); cached != nil && cached.isBinary {
		return cached.binary, nil
	}

	var value []byte
	var err error

	for attempt := 0; attempt <= m.config.RetryAttempts; attempt++ {
		value, err = m.provider.GetBinarySecret(ctx, key)
		if err == nil {
			break
		}

		if attempt < m.config.RetryAttempts {
			time.Sleep(m.config.RetryDelay)
		}
	}

	if err != nil {
		return nil, fmt.Errorf(
			"failed to get binary secret %s after %d attempts: %w",
			key,
			m.config.RetryAttempts+1,
			err,
		)
	}

	m.cache.setBinary(key, value)

	return value, nil
}

func (m *Manager) GetSecrets(ctx context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string)
	uncachedKeys := []string{}

	for _, key := range keys {
		if cached := m.cache.get(key); cached != nil && !cached.isBinary {
			result[key] = cached.value
		} else {
			uncachedKeys = append(uncachedKeys, key)
		}
	}

	if len(uncachedKeys) > 0 {
		secrets, err := m.provider.GetSecrets(ctx, uncachedKeys)
		if err != nil {
			return nil, fmt.Errorf("failed to get secrets: %w", err)
		}

		for key, value := range secrets {
			m.cache.set(key, value, false)
			result[key] = value
		}
	}

	return result, nil
}

func (m *Manager) Close() error {
	if m.provider != nil {
		return m.provider.Close()
	}
	return nil
}

func (m *Manager) ClearCache() {
	m.cache.clear()
}

func (c *secretCache) get(key string) *cachedSecret {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.items[key]
	if !ok {
		return nil
	}

	if time.Now().After(item.expiresAt) {
		return nil
	}

	return item
}

func (c *secretCache) set(key, value string, isBinary bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &cachedSecret{
		value:     value,
		isBinary:  isBinary,
		expiresAt: time.Now().Add(c.ttl),
	}
}

func (c *secretCache) setBinary(key string, value []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &cachedSecret{
		binary:    value,
		isBinary:  true,
		expiresAt: time.Now().Add(c.ttl),
	}
}

func (c *secretCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*cachedSecret)
}

func createProvider(ctx context.Context, config *ports.Config) (ports.SecretProvider, error) {
	switch config.Provider {
	case ports.SecretProviderTypeEnvironment:
		return providers.NewEnvironmentProvider(config.ProviderConfig), nil

	case ports.SecretProviderTypeFile:
		return providers.NewFileProvider(config.ProviderConfig), nil

	case ports.SecretProviderTypeJSON:
		return providers.NewJSONProvider(config.ProviderConfig)

	case ports.SecretProviderTypeAWS:
		return providers.NewAWSProvider(ctx, config.ProviderConfig)

	case ports.SecretProviderTypeVault:
		return providers.NewHashiCorpVaultProvider(ctx, config.ProviderConfig)

	case ports.SecretProviderTypeAzure:
		return providers.NewAzureProvider(ctx, config.ProviderConfig)

	case ports.SecretProviderTypeGCP:
		return providers.NewGCPProvider(ctx, config.ProviderConfig)

	case ports.SecretProviderTypeKubernetes:
		return providers.NewKubernetesProvider(ctx, config.ProviderConfig)

	default:
		return nil, fmt.Errorf("unsupported secret provider: %s", config.Provider)
	}
}
