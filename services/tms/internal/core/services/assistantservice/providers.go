package assistantservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
)

/*
What a person may choose from, and nothing else.

A provider row carries an encrypted API key and the endpoint it belongs to.
Neither is any use to a picker and both are worth keeping off a response that
every assistant user can call, so this projects rather than returning the
entity. The administrative endpoint under ResourceAIProvider still serves the
full record to the people who manage them.

The candidate list is the router's own: tenant-scoped, enabled, and assigned
the assistant task. A provider absent from here cannot be reached by naming its
id either, because the router applies a preference to this same filtered set —
so this endpoint decides what is offered, never what is permitted.
*/
func (s *Service) SelectableProviders(
	ctx context.Context,
	tenant serviceports.RequestActor,
) ([]serviceports.AssistantProviderOption, error) {
	providers, err := s.providers.ListForTask(ctx, repositories.ListAIProvidersForTaskRequest{
		Task:       aiprovider.TaskAssistantChat,
		TenantInfo: tenant.TenantInfo(),
	})
	if err != nil {
		return nil, err
	}

	options := make([]serviceports.AssistantProviderOption, 0, len(providers))
	for _, provider := range providers {
		if ok, _ := provider.CanServeTask(aiprovider.TaskAssistantChat); !ok {
			continue
		}

		options = append(options, serviceports.AssistantProviderOption{
			ID:      provider.ID,
			Name:    provider.Name,
			Kind:    string(provider.Kind),
			Model:   provider.Model,
			Trusted: provider.Trusted,
		})
	}

	return options, nil
}

// resolvePreference keeps a choice only while it is still one of the options.
//
// A provider can be deleted, disabled, or unassigned from the assistant after
// somebody picked it. The router would ignore the stale id and answer from the
// priority order, which is the right behaviour — but leaving the dead id on the
// thread would keep showing a model in the picker that nothing will ever use.
// Dropping it here makes the stored preference and the answer agree.
func (s *Service) resolvePreference(
	ctx context.Context,
	chosen pulid.ID,
	tenant serviceports.RequestActor,
) pulid.ID {
	if chosen.IsNil() {
		return pulid.Nil
	}

	options, err := s.SelectableProviders(ctx, tenant)
	if err != nil {
		// A lookup failure is not a reason to lose the person's choice. The
		// router validates it against the same set before using it, so passing
		// it through unverified cannot widen what it reaches.
		return chosen
	}

	for _, option := range options {
		if option.ID == chosen {
			return chosen
		}
	}

	return pulid.Nil
}
