package loaders

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type agentDefinitionsForEventsLister interface {
	ListEnabledForEvents(
		ctx context.Context,
		req agentdefinitionports.ListAgentDefinitionsForEventsRequest,
	) ([]*agentdefinition.Definition, error)
}

type SuggestedAgentByEventKindLoaderFactoryParams struct {
	fx.In

	Definitions agentdefinitionports.AgentDefinitionRepository
}

type SuggestedAgentByEventKindLoaderFactory struct {
	definitions agentDefinitionsForEventsLister
}

func NewSuggestedAgentByEventKindLoaderFactory(
	p SuggestedAgentByEventKindLoaderFactoryParams,
) *SuggestedAgentByEventKindLoaderFactory {
	return &SuggestedAgentByEventKindLoaderFactory{definitions: p.Definitions}
}

func newSuggestedAgentByEventKindLoader(
	f *SuggestedAgentByEventKindLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *agentdefinition.Definition] {
	return newLoader(
		func(ctx context.Context, keys []string) ([]*agentdefinition.Definition, []error) {
			values := make([]*agentdefinition.Definition, len(keys))
			errs := make([]error, len(keys))

			kinds := make([]agent.EventKind, 0, len(keys))
			seen := make(map[string]struct{}, len(keys))
			for _, key := range keys {
				key = strings.TrimSpace(key)
				if key == "" {
					continue
				}
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				kinds = append(kinds, agent.EventKind(key))
			}
			if len(kinds) == 0 {
				return values, errs
			}

			definitions, err := f.definitions.ListEnabledForEvents(
				ctx,
				agentdefinitionports.ListAgentDefinitionsForEventsRequest{
					TenantInfo: tenantInfo,
					EventKinds: kinds,
				},
			)
			if err != nil {
				fillMissingErrors(errs, err)
				return values, errs
			}

			first := make(map[agent.EventKind]*agentdefinition.Definition, len(kinds))
			for _, definition := range definitions {
				if definition == nil {
					continue
				}
				for _, kind := range definition.EventKinds {
					if _, taken := first[kind]; !taken {
						first[kind] = definition
					}
				}
			}
			for i, key := range keys {
				values[i] = first[agent.EventKind(strings.TrimSpace(key))]
			}

			return values, errs
		},
	)
}
