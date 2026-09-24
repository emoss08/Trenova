package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type usableAgentsReader interface {
	AgentsUsable(
		ctx context.Context,
		actor *services.RequestActor,
		operation permission.Operation,
	) (*services.UsableAgents, error)
}

type UsableAgentByIDLoaderFactoryParams struct {
	fx.In

	Definitions repositories.AgentDefinitionRepository
	Permissions services.PermissionEngine
}

// UsableAgentByIDLoaderFactory reads, for the person making the request, the
// agents among a page's ids they may ask in a conversation: enabled, talked
// to, and open to them. One query for the agents and one read of what the
// person may use cover the whole page; an agent they may not ask is not
// found. It is what a picker's delegate marks are read through, so a
// delegate is never shown to a person who could not use it.
type UsableAgentByIDLoaderFactory struct {
	definitions agentDefinitionsByIDsLister
	permissions usableAgentsReader
}

func NewUsableAgentByIDLoaderFactory(
	p UsableAgentByIDLoaderFactoryParams,
) *UsableAgentByIDLoaderFactory {
	return &UsableAgentByIDLoaderFactory{definitions: p.Definitions, permissions: p.Permissions}
}

func (f *UsableAgentByIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *agentdefinition.Definition] {
	return newLoader(batchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*agentdefinition.Definition, error) {
			return f.usable(ctx, tenantInfo, ids)
		},
		"Agent not found",
	))
}

func (f *UsableAgentByIDLoaderFactory) usable(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
) ([]*agentdefinition.Definition, error) {
	if tenantInfo.UserID.IsNil() || f.permissions == nil {
		return []*agentdefinition.Definition{}, nil
	}

	usable, err := f.permissions.AgentsUsable(
		ctx,
		services.UserActor(tenantInfo),
		permission.OpCreate,
	)
	if err != nil {
		return nil, err
	}
	if usable == nil || !usable.Assistant {
		return []*agentdefinition.Definition{}, nil
	}

	found, err := f.definitions.ListByIDs(ctx, repositories.ListAgentDefinitionsByIDsRequest{
		IDs:        ids,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	allowed := make([]*agentdefinition.Definition, 0, len(found))
	for _, definition := range found {
		if definition.Enabled && !definition.IsBackground() && usable.Allows(definition) {
			allowed = append(allowed, definition)
		}
	}

	return allowed, nil
}
