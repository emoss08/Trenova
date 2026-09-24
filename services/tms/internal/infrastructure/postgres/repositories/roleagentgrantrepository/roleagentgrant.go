package roleagentgrantrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.RoleAgentGrantRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.roleagentgrant-repository"),
	}
}

func (r *repository) ListGrantedAgentIDs(
	ctx context.Context,
	req repositories.ListGrantedAgentIDsRequest,
) ([]pulid.ID, error) {
	if len(req.RoleIDs) == 0 {
		return []pulid.ID{}, nil
	}

	cols := buncolgen.RoleAgentGrantColumns
	ids := make([]pulid.ID, 0, len(req.RoleIDs))
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*permission.RoleAgentGrant)(nil)).
		Distinct().
		Column(cols.AgentDefinitionID.Bare()).
		Where(cols.OrganizationID.Eq(), req.OrganizationID).
		Where(cols.RoleID.In(), bun.List(req.RoleIDs)).
		Scan(ctx, &ids)
	if err != nil {
		r.l.Error("failed to list granted agents", zap.Error(err))
		return nil, fmt.Errorf("list agents granted to roles: %w", err)
	}

	return ids, nil
}

func (r *repository) listGrants(
	ctx context.Context,
	tenant pagination.TenantInfo,
	key buncolgen.Column,
	ids []pulid.ID,
) ([]*permission.RoleAgentGrant, error) {
	grants := make([]*permission.RoleAgentGrant, 0, len(ids))
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&grants).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RoleAgentGrantScopeTenant(sq, tenant).
				Where(key.In(), bun.List(ids))
		}).
		Order(buncolgen.RoleAgentGrantColumns.GrantedAt.OrderAsc()).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list role agent grants: %w", err)
	}

	return grants, nil
}

func (r *repository) ListRolesByAgents(
	ctx context.Context,
	req repositories.ListGrantsByAgentsRequest,
) (map[pulid.ID][]*permission.Role, error) {
	out := make(map[pulid.ID][]*permission.Role, len(req.AgentIDs))
	if len(req.AgentIDs) == 0 {
		return out, nil
	}

	grants, err := r.listGrants(
		ctx,
		req.TenantInfo,
		buncolgen.RoleAgentGrantColumns.AgentDefinitionID,
		req.AgentIDs,
	)
	if err != nil {
		r.l.Error("failed to list grants by agent", zap.Error(err))
		return nil, err
	}
	if len(grants) == 0 {
		return out, nil
	}

	roleIDs := make([]pulid.ID, 0, len(grants))
	for _, grant := range grants {
		roleIDs = append(roleIDs, grant.RoleID)
	}

	roleCols := buncolgen.RoleColumns
	roles := make([]*permission.Role, 0, len(roleIDs))
	err = r.db.DBForContext(ctx).
		NewSelect().
		Model(&roles).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RoleScopeTenant(sq, req.TenantInfo).
				Where(roleCols.ID.In(), bun.List(sliceutils.Dedupe(roleIDs)))
		}).
		Order(roleCols.Name.OrderAsc()).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list granted roles", zap.Error(err))
		return nil, fmt.Errorf("list roles granted agents: %w", err)
	}

	byRole := make(map[pulid.ID][]pulid.ID, len(roles))
	for _, grant := range grants {
		byRole[grant.RoleID] = append(byRole[grant.RoleID], grant.AgentDefinitionID)
	}
	for _, role := range roles {
		for _, agentID := range byRole[role.ID] {
			out[agentID] = append(out[agentID], role)
		}
	}

	return out, nil
}

func (r *repository) ListAgentsByRoles(
	ctx context.Context,
	req repositories.ListGrantsByRolesRequest,
) (map[pulid.ID][]*agentdefinition.Definition, error) {
	out := make(map[pulid.ID][]*agentdefinition.Definition, len(req.RoleIDs))
	if len(req.RoleIDs) == 0 {
		return out, nil
	}

	grants, err := r.listGrants(
		ctx,
		req.TenantInfo,
		buncolgen.RoleAgentGrantColumns.RoleID,
		req.RoleIDs,
	)
	if err != nil {
		r.l.Error("failed to list grants by role", zap.Error(err))
		return nil, err
	}
	if len(grants) == 0 {
		return out, nil
	}

	agentIDs := make([]pulid.ID, 0, len(grants))
	for _, grant := range grants {
		agentIDs = append(agentIDs, grant.AgentDefinitionID)
	}

	definitionCols := buncolgen.DefinitionColumns
	definitions := make([]*agentdefinition.Definition, 0, len(agentIDs))
	err = r.db.DBForContext(ctx).
		NewSelect().
		Model(&definitions).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DefinitionScopeTenant(sq, req.TenantInfo).
				Where(definitionCols.ID.In(), bun.List(sliceutils.Dedupe(agentIDs)))
		}).
		Order(definitionCols.Name.OrderAsc()).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list granted agents", zap.Error(err))
		return nil, fmt.Errorf("list agents granted to roles: %w", err)
	}

	byAgent := make(map[pulid.ID][]pulid.ID, len(definitions))
	for _, grant := range grants {
		byAgent[grant.AgentDefinitionID] = append(byAgent[grant.AgentDefinitionID], grant.RoleID)
	}
	for _, definition := range definitions {
		for _, roleID := range byAgent[definition.ID] {
			out[roleID] = append(out[roleID], definition)
		}
	}

	return out, nil
}

func (r *repository) ListGrantableRoles(
	ctx context.Context,
	req repositories.ListGrantableRolesRequest,
) ([]*permission.Role, error) {
	cols := buncolgen.RoleColumns
	roles := make([]*permission.Role, 0, len(req.RoleIDs))
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&roles).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.RoleScopeTenant(sq, req.TenantInfo)
			if len(req.RoleIDs) > 0 {
				sq = sq.Where(cols.ID.In(), bun.List(req.RoleIDs))
			}

			return sq
		}).
		Order(cols.Name.OrderAsc()).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list grantable roles", zap.Error(err))
		return nil, fmt.Errorf("list grantable roles: %w", err)
	}

	return roles, nil
}

type replaceParams struct {
	tenant  pagination.TenantInfo
	fixed   buncolgen.Column
	fixedID pulid.ID
	varying buncolgen.Column
	wanted  []pulid.ID
	grant   func(varyingID pulid.ID) *permission.RoleAgentGrant
}

func (r *repository) replace(
	ctx context.Context,
	p replaceParams,
) (repositories.GrantChange, error) {
	db := r.db.DBForContext(ctx)

	previous := make([]pulid.ID, 0, len(p.wanted))
	err := db.NewSelect().
		Model((*permission.RoleAgentGrant)(nil)).
		Column(p.varying.Bare()).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RoleAgentGrantScopeTenant(sq, p.tenant).
				Where(p.fixed.Eq(), p.fixedID)
		}).
		For("UPDATE").
		Scan(ctx, &previous)
	if err != nil {
		return repositories.GrantChange{}, fmt.Errorf("read the current grants: %w", err)
	}

	wanted := sliceutils.Dedupe(p.wanted)
	change := repositories.GrantChange{
		Previous: previous,
		Added:    sliceutils.Difference(wanted, previous),
		Removed:  sliceutils.Difference(previous, wanted),
	}

	if len(change.Removed) > 0 {
		_, err = db.NewDelete().
			Model((*permission.RoleAgentGrant)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return buncolgen.RoleAgentGrantScopeTenantDelete(dq, p.tenant).
					Where(p.fixed.Eq(), p.fixedID).
					Where(p.varying.In(), bun.List(change.Removed))
			}).
			Exec(ctx)
		if err != nil {
			return repositories.GrantChange{}, fmt.Errorf("remove grants: %w", err)
		}
	}

	if len(change.Added) > 0 {
		rows := make([]*permission.RoleAgentGrant, 0, len(change.Added))
		for _, id := range change.Added {
			rows = append(rows, p.grant(id))
		}
		cols := buncolgen.RoleAgentGrantColumns
		_, err = db.NewInsert().
			Model(&rows).
			On("CONFLICT (" + cols.RoleID.Bare() + ", " + cols.AgentDefinitionID.Bare() +
				") DO NOTHING").
			Exec(ctx)
		if err != nil {
			return repositories.GrantChange{}, fmt.Errorf("add grants: %w", err)
		}
	}

	return change, nil
}

func (r *repository) ReplaceForAgent(
	ctx context.Context,
	req repositories.ReplaceAgentGrantsRequest,
) (repositories.GrantChange, error) {
	cols := buncolgen.RoleAgentGrantColumns
	change, err := r.replace(ctx, replaceParams{
		tenant:  req.TenantInfo,
		fixed:   cols.AgentDefinitionID,
		fixedID: req.AgentID,
		varying: cols.RoleID,
		wanted:  req.RoleIDs,
		grant: func(roleID pulid.ID) *permission.RoleAgentGrant {
			return &permission.RoleAgentGrant{
				OrganizationID:    req.TenantInfo.OrgID,
				BusinessUnitID:    req.TenantInfo.BuID,
				RoleID:            roleID,
				AgentDefinitionID: req.AgentID,
				GrantedBy:         req.GrantedBy,
			}
		},
	})
	if err != nil {
		r.l.Error("failed to replace the roles granted an agent", zap.Error(err))
		return repositories.GrantChange{}, err
	}

	return change, nil
}

func (r *repository) ReplaceForRole(
	ctx context.Context,
	req repositories.ReplaceRoleGrantsRequest,
) (repositories.GrantChange, error) {
	cols := buncolgen.RoleAgentGrantColumns
	change, err := r.replace(ctx, replaceParams{
		tenant:  req.TenantInfo,
		fixed:   cols.RoleID,
		fixedID: req.RoleID,
		varying: cols.AgentDefinitionID,
		wanted:  req.AgentIDs,
		grant: func(agentID pulid.ID) *permission.RoleAgentGrant {
			return &permission.RoleAgentGrant{
				OrganizationID:    req.TenantInfo.OrgID,
				BusinessUnitID:    req.TenantInfo.BuID,
				RoleID:            req.RoleID,
				AgentDefinitionID: agentID,
				GrantedBy:         req.GrantedBy,
			}
		},
	})
	if err != nil {
		r.l.Error("failed to replace the agents granted a role", zap.Error(err))
		return repositories.GrantChange{}, err
	}

	return change, nil
}
