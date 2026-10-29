package agentloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
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

func newAgentDefinitionStatsLoader(f *AgentDefinitionStatsLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, agentdefinitionports.AgentDefinitionStats] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *AgentDefinitionStatsLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[agentdefinitionports.AgentDefinitionStats] {
	return func(ctx context.Context, keys []string) ([]agentdefinitionports.AgentDefinitionStats, []error) {
		values := make([]agentdefinitionports.AgentDefinitionStats, len(keys))
		errs := make([]error, len(keys))

		ids, indexesByID := loadercore.ParseBatchKeys(keys, errs)
		if len(ids) == 0 {
			return values, errs
		}

		stats, err := f.definitionRepo.StatsByIDs(ctx, tenantInfo, ids)
		if err != nil {
			loadercore.FillMissingErrors(errs, err)
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

var agentDefinitionStatsByIDKey = loadercore.NewKey(newAgentDefinitionStatsLoader)

func AgentDefinitionStatsByID(
	ctx context.Context,
	key string,
) (agentdefinitionports.AgentDefinitionStats, error) {
	return loadercore.Load(ctx, agentDefinitionStatsByIDKey, key)
}

func (f *AgentDefinitionStatsLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
