// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package analytics

import (
	"sync"

	"github.com/emoss08/trenova/internal/core/ports/services"
)

// Registry is an implementation of AnalyticsRegistry
type Registry struct {
	providers map[services.AnalyticsPage]services.AnalyticsPageProvider
	mu        sync.RWMutex
}

// NewRegistry creates a new analytics registry
func NewRegistry() services.AnalyticsRegistry {
	return &Registry{
		providers: make(map[services.AnalyticsPage]services.AnalyticsPageProvider),
	}
}

// RegisterProvider registers an analytics page provider
func (r *Registry) RegisterProvider(provider services.AnalyticsPageProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.providers[provider.GetPage()] = provider
}

// GetProvider returns the provider for a specific page
func (r *Registry) GetProvider(page services.AnalyticsPage) (services.AnalyticsPageProvider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[page]
	return provider, exists
}
