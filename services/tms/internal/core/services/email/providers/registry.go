package providers

import (
	"fmt"
	"sync"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"go.uber.org/fx"
)

type registry struct {
	mu        sync.RWMutex
	providers map[email.ProviderType]Provider
}

type RegistryParams struct {
	fx.In

	Providers []Provider `group:"email_providers"`
}

func NewRegistry(p RegistryParams) Registry {
	r := &registry{
		providers: make(map[email.ProviderType]Provider),
	}

	for _, provider := range p.Providers {
		r.Register(provider)
	}

	return r
}

func (r *registry) Register(provider Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.providers[provider.GetType()] = provider
}

func (r *registry) Get(providerType email.ProviderType) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, ok := r.providers[providerType]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", providerType)
	}

	return provider, nil
}

func (r *registry) List() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]Provider, 0, len(r.providers))
	for _, provider := range r.providers {
		providers = append(providers, provider)
	}

	return providers
}
