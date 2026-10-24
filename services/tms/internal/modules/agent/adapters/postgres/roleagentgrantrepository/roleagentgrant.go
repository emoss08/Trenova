package roleagentgrantrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition/agentdefinitiondb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission/permissiondb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/keelkinds"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
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

func New(p Params) permissionports.RoleAgentGrantRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.roleagentgrant-repository"),
	}
}

func grantedAgentIDsQuery(
	ctx context.Context,
	req permissionports.ListGrantedAgentIDsRequest,
) *keel.Query[permission.RoleAgentGrant] {
	rag := permissiondb.RoleAgentGrants
	q := rag.Group(rag.AgentDefinitionID).Pick(rag.AgentDefinitionID)
	if dbscope.IsSystem(ctx) {
		q = q.AllTenants()
	}

	return q.Where(rag.OrganizationID.Eq(req.OrganizationID), rag.RoleID.In(req.RoleIDs...))
}

func grantsQuery(key keelkinds.PulidIDCol, ids []pulid.ID) *keel.Query[permission.RoleAgentGrant] {
	rag := permissiondb.RoleAgentGrants

	return rag.Where(key.In(ids...)).Order(rag.GrantedAt.Asc())
}

func rolesQuery(ids []pulid.ID) *keel.Query[permission.Role] {
	ro := permissiondb.Roles

	return ro.Where(keel.When(len(ids) > 0, ro.ID.In(ids...))).Order(ro.Name.Asc())
}

func definitionsQuery(ids []pulid.ID) *keel.Query[agentdefinition.Definition] {
	d := agentdefinitiondb.Definitions

	return d.Where(d.ID.In(ids...)).Order(d.Name.Asc())
}

func currentGrantsQuery(p replaceParams) *keel.Query[permission.RoleAgentGrant] {
	rag := permissiondb.RoleAgentGrants

	return rag.Pick(p.varying).Where(p.fixed.Eq(p.fixedID)).Lock(keel.ForUpdate)
}

func (r *repository) ListGrantedAgentIDs(
	ctx context.Context,
	req permissionports.ListGrantedAgentIDsRequest,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		if len(req.RoleIDs) == 0 {
			return []pulid.ID{}, nil
		}

		ids, err := keel.Into[pulid.ID](ctx, r.db.Keel(ctx), grantedAgentIDsQuery(ctx, req))
		if err != nil {
			r.l.Error("failed to list granted agents", zap.Error(err))
			return nil, fmt.Errorf("list agents granted to roles: %w", err)
		}

		return ids, nil
	})
}

func (r *repository) listGrants(
	ctx context.Context,
	key keelkinds.PulidIDCol,
	ids []pulid.ID,
) ([]*permission.RoleAgentGrant, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*permission.RoleAgentGrant, error) {
		grants, err := grantsQuery(key, ids).All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list role agent grants: %w", err)
		}

		return grants, nil
	})
}

func (r *repository) ListRolesByAgents(
	ctx context.Context,
	req permissionports.ListGrantsByAgentsRequest,
) (map[pulid.ID][]*permission.Role, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID][]*permission.Role, error) {
			out := make(map[pulid.ID][]*permission.Role, len(req.AgentIDs))
			if len(req.AgentIDs) == 0 {
				return out, nil
			}

			grants, err := r.listGrants(
				ctx,
				permissiondb.RoleAgentGrants.AgentDefinitionID,
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

			roles, err := rolesQuery(sliceutils.Dedupe(roleIDs)).All(ctx, r.db.Keel(ctx))
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
		},
	)
}

func (r *repository) ListAgentsByRoles(
	ctx context.Context,
	req permissionports.ListGrantsByRolesRequest,
) (map[pulid.ID][]*agentdefinition.Definition, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID][]*agentdefinition.Definition, error) {
			out := make(map[pulid.ID][]*agentdefinition.Definition, len(req.RoleIDs))
			if len(req.RoleIDs) == 0 {
				return out, nil
			}

			grants, err := r.listGrants(ctx, permissiondb.RoleAgentGrants.RoleID, req.RoleIDs)
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

			definitions, err := definitionsQuery(sliceutils.Dedupe(agentIDs)).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list granted agents", zap.Error(err))
				return nil, fmt.Errorf("list agents granted to roles: %w", err)
			}

			byAgent := make(map[pulid.ID][]pulid.ID, len(definitions))
			for _, grant := range grants {
				byAgent[grant.AgentDefinitionID] = append(
					byAgent[grant.AgentDefinitionID],
					grant.RoleID,
				)
			}
			for _, definition := range definitions {
				for _, roleID := range byAgent[definition.ID] {
					out[roleID] = append(out[roleID], definition)
				}
			}

			return out, nil
		},
	)
}

func (r *repository) ListGrantableRoles(
	ctx context.Context,
	req permissionports.ListGrantableRolesRequest,
) ([]*permission.Role, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*permission.Role, error) {
		roles, err := rolesQuery(req.RoleIDs).All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list grantable roles", zap.Error(err))
			return nil, fmt.Errorf("list grantable roles: %w", err)
		}

		return roles, nil
	})
}

type replaceParams struct {
	fixed   keelkinds.PulidIDCol
	fixedID pulid.ID
	varying keelkinds.PulidIDCol
	wanted  []pulid.ID
	grant   func(varyingID pulid.ID) *permission.RoleAgentGrant
}

func (r *repository) replace(
	ctx context.Context,
	p replaceParams,
) (permissionports.GrantChange, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (permissionports.GrantChange, error) {
		db := r.db.Keel(ctx)
		rag := permissiondb.RoleAgentGrants

		previous, err := keel.Into[pulid.ID](ctx, db, currentGrantsQuery(p))
		if err != nil {
			return permissionports.GrantChange{}, fmt.Errorf("read the current grants: %w", err)
		}

		wanted := sliceutils.Dedupe(p.wanted)
		change := permissionports.GrantChange{
			Previous: previous,
			Added:    sliceutils.Difference(wanted, previous),
			Removed:  sliceutils.Difference(previous, wanted),
		}

		if len(change.Removed) > 0 {
			_, err = rag.Where(p.fixed.Eq(p.fixedID), p.varying.In(change.Removed...)).
				Delete(ctx, db)
			if err != nil {
				return permissionports.GrantChange{}, fmt.Errorf("remove grants: %w", err)
			}
		}

		if len(change.Added) > 0 {
			rows := make([]*permission.RoleAgentGrant, 0, len(change.Added))
			for _, id := range change.Added {
				rows = append(rows, p.grant(id))
			}
			if _, err = rag.CreateOrSkip(ctx, db, rows, rag.RoleID, rag.AgentDefinitionID); err != nil {
				return permissionports.GrantChange{}, fmt.Errorf("add grants: %w", err)
			}
		}

		return change, nil
	})
}

func (r *repository) ReplaceForAgent(
	ctx context.Context,
	req permissionports.ReplaceAgentGrantsRequest,
) (permissionports.GrantChange, error) {
	rag := permissiondb.RoleAgentGrants
	change, err := r.replace(ctx, replaceParams{
		fixed:   rag.AgentDefinitionID,
		fixedID: req.AgentID,
		varying: rag.RoleID,
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
		return permissionports.GrantChange{}, err
	}

	return change, nil
}

func (r *repository) ReplaceForRole(
	ctx context.Context,
	req permissionports.ReplaceRoleGrantsRequest,
) (permissionports.GrantChange, error) {
	rag := permissiondb.RoleAgentGrants
	change, err := r.replace(ctx, replaceParams{
		fixed:   rag.RoleID,
		fixedID: req.RoleID,
		varying: rag.AgentDefinitionID,
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
		return permissionports.GrantChange{}, err
	}

	return change, nil
}
