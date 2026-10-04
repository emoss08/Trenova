package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type agentMemoriesByIDsLister interface {
	ListByIDs(
		ctx context.Context,
		req repositories.ListAgentMemoriesByIDsRequest,
	) ([]*agent.Memory, error)
}

type AgentMemoryByIDLoaderFactoryParams struct {
	fx.In

	Memories repositories.AgentMemoryRepository
}

type AgentMemoryByIDLoaderFactory struct {
	memories agentMemoriesByIDsLister
}

func NewAgentMemoryByIDLoaderFactory(
	p AgentMemoryByIDLoaderFactoryParams,
) *AgentMemoryByIDLoaderFactory {
	return &AgentMemoryByIDLoaderFactory{memories: p.Memories}
}

func (f *AgentMemoryByIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *agent.Memory] {
	return newLoader(batchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*agent.Memory, error) {
			return f.memories.ListByIDs(ctx, repositories.ListAgentMemoriesByIDsRequest{
				TenantInfo: tenantInfo,
				IDs:        ids,
			})
		},
		"Agent memory not found",
	))
}

type agentMemoryReplacementsLister interface {
	ListReplacements(
		ctx context.Context,
		req repositories.ListAgentMemoryReplacementsRequest,
	) ([]*agent.Memory, error)
}

type AgentMemoryReplacementLoaderFactoryParams struct {
	fx.In

	Memories repositories.AgentMemoryRepository
}

type AgentMemoryReplacementLoaderFactory struct {
	memories agentMemoryReplacementsLister
}

func NewAgentMemoryReplacementLoaderFactory(
	p AgentMemoryReplacementLoaderFactoryParams,
) *AgentMemoryReplacementLoaderFactory {
	return &AgentMemoryReplacementLoaderFactory{memories: p.Memories}
}

func (f *AgentMemoryReplacementLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *agent.Memory] {
	return newLoader(batchReplacementFunc(f.memories, tenantInfo))
}

func batchReplacementFunc(
	memories agentMemoryReplacementsLister,
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[*agent.Memory] {
	return func(ctx context.Context, keys []string) ([]*agent.Memory, []error) {
		values := make([]*agent.Memory, len(keys))
		errs := make([]error, len(keys))

		ids, indexesByID := parseBatchKeys(keys, errs)
		if len(ids) == 0 {
			return values, errs
		}

		replacements, err := memories.ListReplacements(
			ctx,
			repositories.ListAgentMemoryReplacementsRequest{
				TenantInfo:  tenantInfo,
				ReplacedIDs: ids,
			},
		)
		if err != nil {
			fillMissingErrors(errs, err)
			return values, errs
		}

		newest := agent.NewestReplacements(replacements)
		for _, id := range ids {
			for _, idx := range indexesByID[id] {
				values[idx] = newest[id]
			}
		}

		return values, errs
	}
}
