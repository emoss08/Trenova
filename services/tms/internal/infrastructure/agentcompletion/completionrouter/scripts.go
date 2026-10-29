package completionrouter

import (
	"context"
	"errors"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) ScriptsOffered(ctx context.Context, tenant pagination.TenantInfo) (bool, error) {
	usable, err := s.usableFor(ctx, aitask.TaskAssistantChat, tenant)
	if err != nil {
		if errors.Is(err, llmports.ErrNoProviderConfigured) {
			return false, nil
		}
		return false, err
	}

	return slices.ContainsFunc(usable, func(provider *aiprovider.Provider) bool {
		return provider.ScriptSettings().Scripts
	}), nil
}

// ScriptSettings reads the provider as it is now. A provider that no longer
// serves assistant chat, or is disabled, runs no script whatever it holds.
func (s *Service) ScriptSettings(
	ctx context.Context,
	providerID pulid.ID,
	tenant pagination.TenantInfo,
) (aiprovider.ScriptSettings, error) {
	if providerID.IsNil() {
		return aiprovider.ScriptSettings{}, nil
	}
	provider, err := s.repo.GetByID(ctx, aiproviderports.GetAIProviderByIDRequest{
		ID:         providerID,
		TenantInfo: tenant,
	})
	if err != nil {
		return aiprovider.ScriptSettings{}, err
	}
	if ok, _ := provider.CanServeTask(aitask.TaskAssistantChat); !ok {
		return aiprovider.ScriptSettings{}, nil
	}

	return provider.ScriptSettings(), nil
}
