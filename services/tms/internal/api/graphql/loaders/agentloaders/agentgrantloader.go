package agentloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type agentGrantLister interface {
	ListRolesByAgents(
		ctx context.Context,
		req permissionports.ListGrantsByAgentsRequest,
	) (map[pulid.ID][]*rbac.Role, error)
	ListAgentsByRoles(
		ctx context.Context,
		req permissionports.ListGrantsByRolesRequest,
	) (map[pulid.ID][]*agentdefinition.Definition, error)
}

type AgentGrantLoaderFactoryParams struct {
	fx.In

	Grants permissionports.RoleAgentGrantRepository
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

func newAccessRolesByAgentIDLoader(f *AccessRolesByAgentIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*rbac.Role] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *AccessRolesByAgentIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[[]*rbac.Role] {
	return loadercore.BatchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*rbac.Role, error) {
			return f.grants.ListRolesByAgents(ctx, permissionports.ListGrantsByAgentsRequest{
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

func newAgentsByRoleIDLoader(f *AgentsByRoleIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*agentdefinition.Definition] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *AgentsByRoleIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[[]*agentdefinition.Definition] {
	return loadercore.BatchGroupFunc(
		func(
			ctx context.Context,
			ids []pulid.ID,
		) (map[pulid.ID][]*agentdefinition.Definition, error) {
			return f.grants.ListAgentsByRoles(ctx, permissionports.ListGrantsByRolesRequest{
				TenantInfo: tenantInfo,
				RoleIDs:    ids,
			})
		},
	)
}

var (
	accessRolesByAgentIDKey = loadercore.NewKey(newAccessRolesByAgentIDLoader)
	agentsByRoleIDKey       = loadercore.NewKey(newAgentsByRoleIDLoader)
)

func AccessRolesByAgentID(ctx context.Context, key string) ([]*rbac.Role, error) {
	return loadercore.Load(ctx, accessRolesByAgentIDKey, key)
}

func (f *AccessRolesByAgentIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}

func AgentsByRoleID(ctx context.Context, key string) ([]*agentdefinition.Definition, error) {
	return loadercore.Load(ctx, agentsByRoleIDKey, key)
}

func (f *AgentsByRoleIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
