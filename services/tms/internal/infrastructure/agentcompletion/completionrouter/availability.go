package completionrouter

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/pagination"
)

// NovaReady reports whether Nova can run for the tenant: AI is on and the
// organization has an enabled provider chosen for Nova.
func (s *Service) NovaReady(ctx context.Context, tenant pagination.TenantInfo) (bool, error) {
	if !s.AI.AIEnabled() {
		return false, nil
	}

	if _, err := s.NovaProvider(ctx, tenant); err != nil {
		if errors.Is(err, llmports.ErrNoProviderConfigured) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
