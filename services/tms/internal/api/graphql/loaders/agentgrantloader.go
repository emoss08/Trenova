package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type agentGrantLister interface {
	ListRolesByAgents(
		ctx context.Context,
		req repositories.ListGrantsByAgentsRequest,
	) (map[pulid.ID][]*permission.Role, error)
	ListAgentsByRoles(
		ctx context.Context,
		req repositories.ListGrantsByRolesRequest,
	) (map[pulid.ID][]*agentdefinition.Definition, error)
}

type AgentGrantLoaderFactoryParams struct {
	fx.In

	Grants repositories.RoleAgentGrantRepository
}

// AccessRolesByAgentIDLoaderFactory reads the roles granted a page of agents
// in one query.
type AccessRolesByAgentIDLoaderFactory struct {
	grants agentGrantLister
}

func NewAccessRolesByAgentIDLoaderFactory(
	p AgentGrantLoaderFactoryParams,
) *AccessRolesByAgentIDLoaderFactory {
	return &AccessRolesByAgentIDLoaderFactory{grants: p.Grants}
}

func (f *AccessRolesByAgentIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*permission.Role] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *AccessRolesByAgentIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[[]*permission.Role] {
	return batchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*permission.Role, error) {
			return f.grants.ListRolesByAgents(ctx, repositories.ListGrantsByAgentsRequest{
				TenantInfo: tenantInfo,
				AgentIDs:   ids,
			})
		},
	)
}

// AgentsByRoleIDLoaderFactory reads the agents granted a page of roles in one
// query.
type AgentsByRoleIDLoaderFactory struct {
	grants agentGrantLister
}

func NewAgentsByRoleIDLoaderFactory(p AgentGrantLoaderFactoryParams) *AgentsByRoleIDLoaderFactory {
	return &AgentsByRoleIDLoaderFactory{grants: p.Grants}
}

func (f *AgentsByRoleIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*agentdefinition.Definition] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *AgentsByRoleIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[[]*agentdefinition.Definition] {
	return batchGroupFunc(
		func(
			ctx context.Context,
			ids []pulid.ID,
		) (map[pulid.ID][]*agentdefinition.Definition, error) {
			return f.grants.ListAgentsByRoles(ctx, repositories.ListGrantsByRolesRequest{
				TenantInfo: tenantInfo,
				RoleIDs:    ids,
			})
		},
	)
}
