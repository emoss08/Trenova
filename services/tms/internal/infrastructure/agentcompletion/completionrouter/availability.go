package completionrouter

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/completionengine"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

// NovaReady says whether Nova can run the task for the tenant: AI is on, the
// plan pays for the task and the organization has an enabled provider chosen
// for Nova. When it cannot, the error says why and is
// llmports.ErrNoProviderConfigured; any other error is a failure to find out.
func (s *Service) NovaReady(
	ctx context.Context,
	tenant pagination.TenantInfo,
	task aitask.Task,
) error {
	if !s.AI.AIEnabled() {
		return errortypes.NewBusinessError(completionengine.AIDisabledMessage).
			WithInternal(llmports.ErrNoProviderConfigured)
	}

	_, err := s.NovaProvider(ctx, tenant, task)

	return err
}
