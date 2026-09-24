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

type toolTrustLister interface {
	ListByDefinitionIDs(
		ctx context.Context,
		req repositories.ListToolTrustByDefinitionsRequest,
	) (map[pulid.ID][]*agent.ToolTrust, error)
}

type ToolTrustByAgentIDLoaderFactoryParams struct {
	fx.In

	Trust repositories.AgentToolTrustRepository
}

type ToolTrustByAgentIDLoaderFactory struct {
	trust toolTrustLister
}

func NewToolTrustByAgentIDLoaderFactory(
	p ToolTrustByAgentIDLoaderFactoryParams,
) *ToolTrustByAgentIDLoaderFactory {
	return &ToolTrustByAgentIDLoaderFactory{trust: p.Trust}
}

func (f *ToolTrustByAgentIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*agent.ToolTrust] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *ToolTrustByAgentIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[[]*agent.ToolTrust] {
	return batchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*agent.ToolTrust, error) {
			return f.trust.ListByDefinitionIDs(ctx, repositories.ListToolTrustByDefinitionsRequest{
				TenantInfo:         tenantInfo,
				AgentDefinitionIDs: ids,
			})
		},
	)
}
