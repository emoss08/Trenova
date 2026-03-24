package analyticsservice

import (
	"sync"

	"github.com/emoss08/trenova/internal/core/ports/services"
)

type Registry struct {
	providers map[services.AnalyticsPage]services.AnalyticsPageProvider
	mu        sync.RWMutex
}

func NewRegistry() services.AnalyticsRegistry {
	return &Registry{
		providers: make(map[services.AnalyticsPage]services.AnalyticsPageProvider),
	}
}

func (r *Registry) RegisterProvider(provider services.AnalyticsPageProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.providers[provider.GetPage()] = provider
}

func (r *Registry) GetProvider(page services.AnalyticsPage) (services.AnalyticsPageProvider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[page]
	return provider, exists
}
