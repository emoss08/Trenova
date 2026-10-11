package analyticsservice

import (
	"sync"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

type Registry struct {
	providers map[kernelports.AnalyticsPage]kernelports.AnalyticsPageProvider
	mu        sync.RWMutex
}

func NewRegistry() kernelports.AnalyticsRegistry {
	return &Registry{
		providers: make(map[kernelports.AnalyticsPage]kernelports.AnalyticsPageProvider),
	}
}

func (r *Registry) RegisterProvider(provider kernelports.AnalyticsPageProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.providers[provider.GetPage()] = provider
}

func (r *Registry) GetProvider(
	page kernelports.AnalyticsPage,
) (kernelports.AnalyticsPageProvider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[page]
	return provider, exists
}
