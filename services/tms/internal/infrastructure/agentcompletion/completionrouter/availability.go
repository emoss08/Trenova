package completionrouter

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) CanServe(
	ctx context.Context,
	task aitask.Task,
	tenant pagination.TenantInfo,
) (bool, error) {
	if !s.AI.AIEnabled() {
		return false, nil
	}

	if _, err := s.UsableFor(ctx, task, tenant); err != nil {
		if errors.Is(err, llmports.ErrNoProviderConfigured) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
