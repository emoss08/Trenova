package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type AgentDefinitionStatsLoaderFactoryParams struct {
	fx.In

	DefinitionRepo agentdefinitionports.AgentDefinitionRepository
}

type AgentDefinitionStatsLoaderFactory struct {
	definitionRepo agentdefinitionports.AgentDefinitionRepository
}

func NewAgentDefinitionStatsLoaderFactory(
	p AgentDefinitionStatsLoaderFactoryParams,
) *AgentDefinitionStatsLoaderFactory {
	return &AgentDefinitionStatsLoaderFactory{definitionRepo: p.DefinitionRepo}
}

func (f *AgentDefinitionStatsLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, agentdefinitionports.AgentDefinitionStats] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *AgentDefinitionStatsLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[agentdefinitionports.AgentDefinitionStats] {
	return func(ctx context.Context, keys []string) ([]agentdefinitionports.AgentDefinitionStats, []error) {
		values := make([]agentdefinitionports.AgentDefinitionStats, len(keys))
		errs := make([]error, len(keys))

		ids, indexesByID := parseBatchKeys(keys, errs)
		if len(ids) == 0 {
			return values, errs
		}

		stats, err := f.definitionRepo.StatsByIDs(ctx, tenantInfo, ids)
		if err != nil {
			fillMissingErrors(errs, err)
			return values, errs
		}

		for _, id := range ids {
			for _, idx := range indexesByID[id] {
				values[idx] = stats[id]
			}
		}

		return values, errs
	}
}
