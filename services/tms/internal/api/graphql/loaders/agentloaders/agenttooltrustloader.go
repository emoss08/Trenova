package agentloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type toolTrustLister interface {
	ListByDefinitionIDs(
		ctx context.Context,
		req agenttoolports.ListToolTrustByDefinitionsRequest,
	) (map[pulid.ID][]*agenttooltrust.ToolTrust, error)
}

type ToolTrustByAgentIDLoaderFactoryParams struct {
	fx.In

	Trust agenttoolports.AgentToolTrustRepository
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
) *dataloadgen.Loader[string, []*agenttooltrust.ToolTrust] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *ToolTrustByAgentIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[[]*agenttooltrust.ToolTrust] {
	return loadercore.BatchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*agenttooltrust.ToolTrust, error) {
			return f.trust.ListByDefinitionIDs(
				ctx,
				agenttoolports.ListToolTrustByDefinitionsRequest{
					TenantInfo:         tenantInfo,
					AgentDefinitionIDs: ids,
				},
			)
		},
	)
}

var toolTrustByAgentIDKey = loadercore.NewKey(newToolTrustByAgentIDLoader)

func ToolTrustByAgentID(ctx context.Context, key string) ([]*agenttooltrust.ToolTrust, error) {
	return loadercore.Load(ctx, toolTrustByAgentIDKey, key)
}

func (f *ToolTrustByAgentIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
