package providers

import (
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/samber/oops"
	"go.uber.org/fx"
)

// registry manages email providers
type registry struct {
	mu        sync.RWMutex
	providers map[email.ProviderType]Provider
}

// RegistryParams defines the dependencies for creating a registry
type RegistryParams struct {
	fx.In

	Providers []Provider `group:"email_providers"`
}

// NewRegistry creates a new provider registry
func NewRegistry(p RegistryParams) Registry {
	r := &registry{
		providers: make(map[email.ProviderType]Provider),
	}

	// Register all providers
	for _, provider := range p.Providers {
		r.Register(provider)
	}

	return r
}

// Register registers a provider
func (r *registry) Register(provider Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.providers[provider.GetType()] = provider
}

// Get returns a provider by type
func (r *registry) Get(providerType email.ProviderType) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, ok := r.providers[providerType]
	if !ok {
		return nil, oops.In("provider_registry").
			Tags("operation", "get_provider").
			Tags("provider_type", string(providerType)).
			Time(time.Now()).
			Errorf("provider not found: %s", providerType)
	}

	return provider, nil
}

// List returns all registered providers
func (r *registry) List() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]Provider, 0, len(r.providers))
	for _, provider := range r.providers {
		providers = append(providers, provider)
	}

	return providers
}
