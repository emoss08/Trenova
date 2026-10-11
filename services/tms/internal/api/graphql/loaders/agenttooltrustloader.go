package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type toolTrustLister interface {
	ListByDefinitionIDs(
		ctx context.Context,
		req agentports.ListToolTrustByDefinitionsRequest,
	) (map[pulid.ID][]*agent.ToolTrust, error)
}

type ToolTrustByAgentIDLoaderFactoryParams struct {
	fx.In

	Trust agentports.AgentToolTrustRepository
}

type ToolTrustByAgentIDLoaderFactory struct {
	trust toolTrustLister
}

func NewToolTrustByAgentIDLoaderFactory(
	p ToolTrustByAgentIDLoaderFactoryParams,
) *ToolTrustByAgentIDLoaderFactory {
	return &ToolTrustByAgentIDLoaderFactory{trust: p.Trust}
}

func newToolTrustByAgentIDLoader(f *ToolTrustByAgentIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*agent.ToolTrust] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *ToolTrustByAgentIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[[]*agent.ToolTrust] {
	return batchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*agent.ToolTrust, error) {
			return f.trust.ListByDefinitionIDs(ctx, agentports.ListToolTrustByDefinitionsRequest{
				TenantInfo:         tenantInfo,
				AgentDefinitionIDs: ids,
			})
		},
	)
}
