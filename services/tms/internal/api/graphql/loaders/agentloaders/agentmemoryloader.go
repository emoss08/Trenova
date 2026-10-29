package agentloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type agentMemoriesByIDsLister interface {
	ListByIDs(
		ctx context.Context,
		req agentmemoryports.ListAgentMemoriesByIDsRequest,
	) ([]*agentmemory.Memory, error)
}

type AgentMemoryByIDLoaderFactoryParams struct {
	fx.In

	Memories agentmemoryports.AgentMemoryRepository
}

type AgentMemoryByIDLoaderFactory struct {
	memories agentMemoriesByIDsLister
}

func NewAgentMemoryByIDLoaderFactory(
	p AgentMemoryByIDLoaderFactoryParams,
) *AgentMemoryByIDLoaderFactory {
	return &AgentMemoryByIDLoaderFactory{memories: p.Memories}
}

func newAgentMemoryByIDLoader(f *AgentMemoryByIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *agentmemory.Memory] {
	return loadercore.NewLoader(loadercore.BatchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*agentmemory.Memory, error) {
			return f.memories.ListByIDs(ctx, agentmemoryports.ListAgentMemoriesByIDsRequest{
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
		req agentmemoryports.ListAgentMemoryReplacementsRequest,
	) ([]*agentmemory.Memory, error)
}

type AgentMemoryReplacementLoaderFactoryParams struct {
	fx.In

	Memories agentmemoryports.AgentMemoryRepository
}

type AgentMemoryReplacementLoaderFactory struct {
	memories agentMemoryReplacementsLister
}

func NewAgentMemoryReplacementLoaderFactory(
	p AgentMemoryReplacementLoaderFactoryParams,
) *AgentMemoryReplacementLoaderFactory {
	return &AgentMemoryReplacementLoaderFactory{memories: p.Memories}
}

func newAgentMemoryReplacementLoader(f *AgentMemoryReplacementLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *agentmemory.Memory] {
	return loadercore.NewLoader(batchReplacementFunc(f.memories, tenantInfo))
}

func batchReplacementFunc(
	memories agentMemoryReplacementsLister,
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[*agentmemory.Memory] {
	return func(ctx context.Context, keys []string) ([]*agentmemory.Memory, []error) {
		values := make([]*agentmemory.Memory, len(keys))
		errs := make([]error, len(keys))

		ids, indexesByID := loadercore.ParseBatchKeys(keys, errs)
		if len(ids) == 0 {
			return values, errs
		}

		replacements, err := memories.ListReplacements(
			ctx,
			agentmemoryports.ListAgentMemoryReplacementsRequest{
				TenantInfo:  tenantInfo,
				ReplacedIDs: ids,
			},
		)
		if err != nil {
			loadercore.FillMissingErrors(errs, err)
			return values, errs
		}

		newest := agentmemory.NewestReplacements(replacements)
		for _, id := range ids {
			for _, idx := range indexesByID[id] {
				values[idx] = newest[id]
			}
		}

		return values, errs
	}
}

var (
	agentMemoryByIDKey        = loadercore.NewKey(newAgentMemoryByIDLoader)
	agentMemoryReplacementKey = loadercore.NewKey(newAgentMemoryReplacementLoader)
)

func AgentMemoryByID(ctx context.Context, key string) (*agentmemory.Memory, error) {
	return loadercore.Load(ctx, agentMemoryByIDKey, key)
}

func (f *AgentMemoryByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}

func AgentMemoryReplacement(ctx context.Context, key string) (*agentmemory.Memory, error) {
	return loadercore.Load(ctx, agentMemoryReplacementKey, key)
}

func (f *AgentMemoryReplacementLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
