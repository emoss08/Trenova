package rolerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
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

func New(p Params) repositories.RoleRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.role-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListRolesRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(q, "r", req.Filter, (*permission.Role)(nil))

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListRolesRequest,
) (*pagination.ListResult[*permission.Role], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*permission.Role, 0, req.Filter.Pagination.SafeLimit())

	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count roles", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*permission.Role]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*permission.Role], error) {
	return dbhelper.SelectOptions[*permission.Role](
		ctx,
		r.db.DB(),
		req,
		&dbhelper.SelectOptionsConfig{
			Columns:       []string{"id", "name", "description"},
			OrgColumn:     "r.organization_id",
			BuColumn:      "r.business_unit_id",
			SearchColumns: []string{"r.name", "r.description"},
			EntityName:    "Role",
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetRoleByIDRequest,
) (*permission.Role, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.ID.String()),
	)

	role := new(permission.Role)
	err := r.db.DB().
		NewSelect().
		Model(role).
		Relation("Permissions").
		Where("r.id = ?", req.ID).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("r.organization_id = ?", req.TenantInfo.OrgID).
				WhereOr("r.organization_id IS NULL")
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get role", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "Role")
	}

	return role, nil
}

func (r *repository) GetRolesWithInheritance(
	ctx context.Context,
	roleIDs []pulid.ID,
) ([]*permission.Role, error) {
	log := r.l.With(
		zap.String("operation", "GetRolesWithInheritance"),
		zap.Int("roleCount", len(roleIDs)),
	)

	if len(roleIDs) == 0 {
		return []*permission.Role{}, nil
	}

	allRoleIDs := make(map[pulid.ID]bool)
	for _, id := range roleIDs {
		allRoleIDs[id] = true
	}

	roles := make([]*permission.Role, 0)
	err := r.db.DB().
		NewSelect().
		Model(&roles).
		Relation("Permissions").
		Where("r.id IN (?)", bun.List(roleIDs)).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get roles", zap.Error(err))
		return nil, err
	}

	var parentIDs []pulid.ID
	for _, role := range roles {
		for _, parentID := range role.ParentRoleIDs {
			if !allRoleIDs[parentID] {
				parentIDs = append(parentIDs, parentID)
				allRoleIDs[parentID] = true
			}
		}
	}

	if len(parentIDs) > 0 {
		parentRoles, parentErr := r.GetRolesWithInheritance(ctx, parentIDs)
		if parentErr != nil {
			return nil, parentErr
		}
		roles = append(roles, parentRoles...)
	}

	return roles, nil
}

func (r *repository) GetUsersWithRole(
	ctx context.Context,
	roleID pulid.ID,
) ([]repositories.ImpactedUser, error) {
	log := r.l.With(
		zap.String("operation", "GetUsersWithRole"),
		zap.String("roleId", roleID.String()),
	)

	var results []repositories.ImpactedUser
	err := r.db.DB().NewSelect().
		TableExpr("user_role_assignments AS ura").
		ColumnExpr("ura.user_id").
		ColumnExpr("u.name AS user_name").
		ColumnExpr("ura.organization_id").
		ColumnExpr("o.name AS org_name").
		ColumnExpr("'direct' AS assignment_type").
		Join("JOIN users AS u ON u.id = ura.user_id").
		Join("JOIN organizations AS o ON o.id = ura.organization_id").
		Where("ura.role_id = ?", roleID).
		Scan(ctx, &results)
	if err != nil {
		log.Error("failed to get users with role", zap.Error(err))
		return nil, err
	}

	return results, nil
}

func (r *repository) GetUserRoleAssignments(
	ctx context.Context,
	userID, orgID pulid.ID,
) ([]*permission.UserRoleAssignment, error) {
	log := r.l.With(
		zap.String("operation", "GetUserRoleAssignments"),
		zap.String("userId", userID.String()),
		zap.String("orgId", orgID.String()),
	)

	assignments := make([]*permission.UserRoleAssignment, 0)
	err := r.db.DB().
		NewSelect().
		Model(&assignments).
		Where("ura.user_id = ?", userID).
		Where("ura.organization_id = ?", orgID).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get user role assignments", zap.Error(err))
		return nil, err
	}

	return assignments, nil
}

func (r *repository) HasBusinessUnitAdminAccess(
	ctx context.Context,
	userID, orgID pulid.ID,
) (bool, error) {
	log := r.l.With(
		zap.String("operation", "HasBusinessUnitAdminAccess"),
		zap.String("userId", userID.String()),
		zap.String("orgId", orgID.String()),
	)

	count, err := r.db.DB().
		NewSelect().
		TableExpr("user_role_assignments AS ura").
		Join("JOIN roles AS r ON r.id = ura.role_id").
		Join("JOIN organizations AS target_org ON target_org.id = ?", orgID).
		Where("ura.user_id = ?", userID).
		Where("r.is_business_unit_admin = TRUE").
		Where("r.business_unit_id = target_org.business_unit_id").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("ura.expires_at IS NULL").
				WhereOr("ura.expires_at > extract(epoch from current_timestamp)::bigint")
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to check business unit admin access", zap.Error(err))
		return false, err
	}

	return count > 0, nil
}

func (r *repository) Create(ctx context.Context, role *permission.Role) error {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("name", role.Name),
	)

	_, err := r.db.DB().NewInsert().Model(role).Returning("*").Exec(ctx)
	if err != nil {
		log.Error("failed to create role", zap.Error(err))
		return err
	}

	for _, rp := range role.Permissions {
		rp.RoleID = role.ID
		if _, err = r.db.DB().NewInsert().Model(rp).Returning("*").Exec(ctx); err != nil {
			log.Error("failed to create resource permission", zap.Error(err))
			return err
		}
	}

	return nil
}

func (r *repository) Update(ctx context.Context, role *permission.Role) error {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", role.ID.String()),
	)

	_, err := r.db.DB().
		NewUpdate().
		Model(role).
		WherePK().
		OmitZero().
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update role", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) CreateAssignment(
	ctx context.Context,
	assignment *permission.UserRoleAssignment,
) error {
	log := r.l.With(
		zap.String("operation", "CreateAssignment"),
		zap.String("userId", assignment.UserID.String()),
		zap.String("roleId", assignment.RoleID.String()),
	)

	_, err := r.db.DB().NewInsert().Model(assignment).Returning("*").Exec(ctx)
	if err != nil {
		log.Error("failed to create role assignment", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) DeleteAssignment(ctx context.Context, id pulid.ID) error {
	log := r.l.With(
		zap.String("operation", "DeleteAssignment"),
		zap.String("id", id.String()),
	)

	_, err := r.db.DB().
		NewDelete().
		Model((*permission.UserRoleAssignment)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete role assignment", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) CreateResourcePermission(
	ctx context.Context,
	rp *permission.ResourcePermission,
) error {
	log := r.l.With(
		zap.String("operation", "CreateResourcePermission"),
		zap.String("roleId", rp.RoleID.String()),
		zap.String("resource", rp.Resource),
	)

	_, err := r.db.DB().NewInsert().Model(rp).Returning("*").Exec(ctx)
	if err != nil {
		log.Error("failed to create resource permission", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) UpdateResourcePermission(
	ctx context.Context,
	rp *permission.ResourcePermission,
) error {
	log := r.l.With(
		zap.String("operation", "UpdateResourcePermission"),
		zap.String("id", rp.ID.String()),
	)

	_, err := r.db.DB().
		NewUpdate().
		Model(rp).
		WherePK().
		OmitZero().
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update resource permission", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) DeleteResourcePermission(ctx context.Context, id pulid.ID) error {
	log := r.l.With(
		zap.String("operation", "DeleteResourcePermission"),
		zap.String("id", id.String()),
	)

	_, err := r.db.DB().
		NewDelete().
		Model((*permission.ResourcePermission)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete resource permission", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) GetResourcePermissionsByRoleID(
	ctx context.Context,
	roleID pulid.ID,
) ([]*permission.ResourcePermission, error) {
	log := r.l.With(
		zap.String("operation", "GetResourcePermissionsByRoleID"),
		zap.String("roleId", roleID.String()),
	)

	permissions := make([]*permission.ResourcePermission, 0)
	err := r.db.DB().
		NewSelect().
		Model(&permissions).
		Where("rp.role_id = ?", roleID).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get resource permissions", zap.Error(err))
		return nil, err
	}

	return permissions, nil
}
