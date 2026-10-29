package rolerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission/permissiondb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/resultcache"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	roleClosureName          = "role_closure"
	parentRoleAlias          = "parent_role"
	directAssignment         = "direct"
	orphanedScopeReason      = "find role assignments in organizations their holders no longer belong to, across every organization"
	roleClosureParentColumn  = "parent_role_ids"
	roleClosureRoleColumn    = "id"
	alreadyAssignedMessage   = "Role is already assigned to this user"
	alreadyAssignedFieldName = "roleId"
	heldRolesName            = "held"
	heldUserColumn           = "user_id"
	heldRoleColumn           = "role_id"
	heldParentColumn         = "parent_role_ids"
)

var (
	roleErrors           = dberror.Mapping{Entity: "Role"}
	assignmentErrors     = dberror.Mapping{Entity: "Role assignment"}
	rolePermissionErrors = dberror.Mapping{Entity: "Role permission"}
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

func New(p Params) permissionports.RoleRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.role-repository"),
	}
}

func forTenant[T any](
	ctx context.Context,
	q *keel.Query[*T],
	preds ...keel.Pred,
) *keel.Query[*T] {
	if dbscope.IsSystem(ctx) {
		q = q.AllTenants()
	}

	return q.Where(preds...)
}

func (r *repository) List(
	ctx context.Context,
	req *permissionports.ListRolesRequest,
) (*pagination.ListResult[*rbac.Role], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*rbac.Role], error) {
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				permissiondb.Roles.Query(),
				&listquery.Request[rbac.Role]{Filter: req.Filter},
			)
			if err != nil {
				r.l.Error("failed to list roles", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *permissionports.ListRoleConnectionRequest,
) (*pagination.CursorListResult[*rbac.Role], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*rbac.Role], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				permissiondb.Roles.Query(),
				&listquery.Request[rbac.Role]{
					Filter: req.Filter,
					Cursor: req.Cursor,
				},
			)
			if err != nil {
				r.l.Error("failed to list roles", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*rbac.Role], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*rbac.Role], error) {
			ro := permissiondb.Roles
			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				ro.Pick(ro.ID, ro.Name, ro.Description),
				req,
				func(s string) keel.Pred {
					return keel.Or(ro.Name.ContainsFold(s), ro.Description.ContainsFold(s))
				},
			)
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req permissionports.GetRoleByIDRequest,
) (*rbac.Role, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*rbac.Role, error) {
		ro := permissiondb.Roles
		role, err := forTenant(
			ctx,
			ro.Query(),
			ro.ID.Eq(req.ID),
			ro.OrganizationID.Eq(req.TenantInfo.OrgID),
		).
			Load(ro.Permissions()).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error(
				"failed to get role",
				zap.String("id", req.ID.String()),
				zap.Error(err),
			)
			return nil, roleErrors.Map(err, req.ID.String())
		}

		return role, nil
	})
}

func rolesWithInheritance(ctx context.Context, roleIDs []pulid.ID) *keel.Query[*rbac.Role] {
	ro := permissiondb.Roles
	parent := ro.As(parentRoleAlias)
	closure := keel.RecursiveCTE(
		roleClosureName,
		[]string{roleClosureRoleColumn, roleClosureParentColumn},
		forTenant(ctx, ro.Pick(ro.ID, ro.ParentRoleIDs), ro.ID.In(roleIDs...)),
	)
	closure.Recurse(forTenant(
		ctx,
		parent.Pick(parent.ID, parent.ParentRoleIDs).
			Join(
				closure,
				parent.ID.InArray(keel.StringArrayOf(closure.Col(roleClosureParentColumn))),
			),
	)).Distinct()

	return forTenant(
		ctx,
		ro.With(closure),
		ro.ID.InQuery(closure.Query().Pick(closure.Col(roleClosureRoleColumn))),
	).Load(ro.Permissions())
}

func (r *repository) GetRolesWithInheritance(
	ctx context.Context,
	roleIDs []pulid.ID,
) ([]*rbac.Role, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*rbac.Role, error) {
		if len(roleIDs) == 0 {
			return []*rbac.Role{}, nil
		}

		roles, err := rolesWithInheritance(ctx, roleIDs).
			Cache(resultcache.Access()...).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error(
				"failed to get roles with inheritance",
				zap.Int("roleCount", len(roleIDs)),
				zap.Error(err),
			)
			return nil, err
		}

		return roles, nil
	})
}

func (r *repository) GetUsersWithRole(
	ctx context.Context,
	roleID pulid.ID,
) ([]permissionports.ImpactedUser, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]permissionports.ImpactedUser, error) {
		ura := permissiondb.UserRoleAssignments
		u := tenantdb.Users
		o := tenantdb.Organizations
		results, err := keel.Into[permissionports.ImpactedUser](ctx, r.db.Keel(ctx), forTenant(
			ctx,
			ura.Join(u, u.ID.EqCol(ura.UserID)).Join(o, o.ID.EqCol(ura.OrganizationID)),
			ura.RoleID.Eq(roleID),
		).Pick(
			ura.UserID,
			u.Name,
			ura.OrganizationID,
			o.Name,
			keel.Text(directAssignment),
		))
		if err != nil {
			r.l.Error(
				"failed to get users with role",
				zap.String("roleId", roleID.String()),
				zap.Error(err),
			)
			return nil, err
		}

		return results, nil
	})
}

func (r *repository) GetUserRoleAssignments(
	ctx context.Context,
	userID, orgID pulid.ID,
) ([]*rbac.UserRoleAssignment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*rbac.UserRoleAssignment, error) {
			ura := permissiondb.UserRoleAssignments
			assignments, err := forTenant(
				ctx,
				ura.Query(),
				ura.UserID.Eq(userID),
				ura.OrganizationID.Eq(orgID),
			).
				Load(ura.Role()).
				Cache(resultcache.Access()...).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error(
					"failed to get user role assignments",
					zap.String("userId", userID.String()),
					zap.String("orgId", orgID.String()),
					zap.Error(err),
				)
				return nil, err
			}

			return assignments, nil
		},
	)
}

func (r *repository) Create(ctx context.Context, role *rbac.Role) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.Keel(ctx)
		if err := permissiondb.Roles.Create(ctx, db, role); err != nil {
			r.l.Error("failed to create role", zap.String("name", role.Name), zap.Error(err))
			return roleErrors.Map(err, role.ID.String())
		}

		if len(role.Permissions) == 0 {
			return nil
		}

		for _, rp := range role.Permissions {
			rp.RoleID = role.ID
		}
		if err := permissiondb.ResourcePermissions.CreateMany(
			ctx,
			db,
			role.Permissions,
		); err != nil {
			r.l.Error("failed to create resource permissions", zap.Error(err))
			return rolePermissionErrors.Map(err, role.ID.String())
		}

		return nil
	})
}

func (r *repository) Update(ctx context.Context, role *rbac.Role) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		ro := permissiondb.Roles
		if err := ro.Save(
			ctx,
			r.db.Keel(ctx),
			role,
			ro.Name,
			ro.Description,
			ro.CoreResponsibility,
			ro.MaxSensitivity,
		); err != nil {
			r.l.Error(
				"failed to update role",
				zap.String("id", role.ID.String()),
				zap.Error(err),
			)
			return roleErrors.Map(err, role.ID.String())
		}

		if role.Permissions == nil {
			return nil
		}

		if err := r.replaceResourcePermissions(ctx, role.ID, role.Permissions); err != nil {
			r.l.Error("failed to replace role permissions", zap.Error(err))
			return err
		}

		return nil
	})
}

func (r *repository) Delete(ctx context.Context, req permissionports.DeleteRoleRequest) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.Keel(ctx)
		ro := permissiondb.Roles
		deleted, err := forTenant(
			ctx,
			ro.Query(),
			ro.ID.Eq(req.RoleID),
			ro.OrganizationID.Eq(req.OrganizationID),
			ro.IsSystem.IsFalse(),
		).Delete(ctx, db)
		if err != nil {
			r.l.Error(
				"failed to delete role",
				zap.String("id", req.RoleID.String()),
				zap.Error(err),
			)
			return roleErrors.Map(err, req.RoleID.String())
		}
		if err = dberror.CheckFoundCount(deleted, roleErrors.Entity); err != nil {
			return err
		}

		if _, err = forTenant(
			ctx,
			ro.Query(),
			ro.ParentRoleIDs.Has(req.RoleID.String()),
			ro.OrganizationID.Eq(req.OrganizationID),
		).Update(
			ctx,
			db,
			ro.ParentRoleIDs.Remove(req.RoleID.String()),
		); err != nil {
			r.l.Error("failed to drop the deleted role from its seniors", zap.Error(err))
			return err
		}

		return nil
	})
}

func (r *repository) replaceResourcePermissions(
	ctx context.Context,
	roleID pulid.ID,
	permissions []*rbac.ResourcePermission,
) error {
	db := r.db.Keel(ctx)
	rp := permissiondb.ResourcePermissions
	if _, err := rp.Where(rp.RoleID.Eq(roleID)).Delete(ctx, db); err != nil {
		return err
	}

	if len(permissions) == 0 {
		return nil
	}

	for _, p := range permissions {
		p.ID = pulid.Nil
		p.RoleID = roleID
	}

	return rp.CreateMany(ctx, db, permissions)
}

func (r *repository) CreateAssignment(
	ctx context.Context,
	assignment *rbac.UserRoleAssignment,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.Keel(ctx)
		ura := permissiondb.UserRoleAssignments

		existing, err := forTenant(
			ctx,
			ura.Query(),
			ura.UserID.Eq(assignment.UserID),
			ura.RoleID.Eq(assignment.RoleID),
			ura.OrganizationID.Eq(assignment.OrganizationID),
		).Lock(keel.ForUpdate).First(ctx, db)
		switch {
		case dberror.IsNotFoundError(err):
			inserted, createErr := ura.CreateOrSkip(
				ctx,
				db,
				[]*rbac.UserRoleAssignment{assignment},
				ura.UserID,
				ura.RoleID,
				ura.OrganizationID,
			)
			if createErr != nil {
				r.l.Error("failed to create role assignment", zap.Error(createErr))
				return assignmentErrors.Map(createErr, assignment.ID.String())
			}
			if inserted == 0 {
				return alreadyAssigned()
			}
			return nil
		case err != nil:
			return err
		}

		assignment.AssignedAt = timeutils.NowUnix()
		if existing.ExpiresAt == nil || *existing.ExpiresAt > assignment.AssignedAt {
			return alreadyAssigned()
		}

		expiresAt := ura.ExpiresAt.ToNull()
		if assignment.ExpiresAt != nil {
			expiresAt = ura.ExpiresAt.To(*assignment.ExpiresAt)
		}
		if _, err = forTenant(ctx, ura.Query(), ura.ID.Eq(existing.ID)).Update(
			ctx,
			db,
			expiresAt,
			ura.AssignedBy.To(assignment.AssignedBy),
			ura.AssignedAt.To(assignment.AssignedAt),
		); err != nil {
			r.l.Error("failed to renew role assignment", zap.Error(err))
			return err
		}

		assignment.ID = existing.ID
		return nil
	})
}

func alreadyAssigned() error {
	return errortypes.NewValidationError(
		alreadyAssignedFieldName,
		errortypes.ErrDuplicate,
		alreadyAssignedMessage,
	)
}

func (r *repository) DeleteAssignment(
	ctx context.Context,
	req permissionports.DeleteRoleAssignmentRequest,
) (*rbac.UserRoleAssignment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*rbac.UserRoleAssignment, error) {
		db := r.db.Keel(ctx)
		ura := permissiondb.UserRoleAssignments
		assignment, err := forTenant(
			ctx,
			ura.Query(),
			ura.ID.Eq(req.AssignmentID),
			ura.OrganizationID.Eq(req.OrganizationID),
		).Lock(keel.ForUpdate).One(ctx, db)
		if err != nil {
			return nil, assignmentErrors.Map(err, req.AssignmentID.String())
		}

		if _, err = forTenant(
			ctx,
			ura.Query(),
			ura.ID.Eq(assignment.ID),
		).Delete(ctx, db); err != nil {
			r.l.Error(
				"failed to delete role assignment",
				zap.String("id", req.AssignmentID.String()),
				zap.Error(err),
			)
			return nil, err
		}

		return assignment, nil
	})
}

func (r *repository) DeleteUserAssignments(
	ctx context.Context,
	userID, orgID pulid.ID,
) ([]*rbac.UserRoleAssignment, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) ([]*rbac.UserRoleAssignment, error) {
			db := r.db.Keel(ctx)
			ura := permissiondb.UserRoleAssignments
			assignments, err := forTenant(
				ctx,
				ura.Query(),
				ura.UserID.Eq(userID),
				ura.OrganizationID.Eq(orgID),
			).Lock(keel.ForUpdate).All(ctx, db)
			if err != nil {
				return nil, err
			}
			if len(assignments) == 0 {
				return []*rbac.UserRoleAssignment{}, nil
			}

			ids := make([]pulid.ID, len(assignments))
			for i, assignment := range assignments {
				ids[i] = assignment.ID
			}
			if _, err = forTenant(ctx, ura.Query(), ura.ID.In(ids...)).Delete(ctx, db); err != nil {
				r.l.Error(
					"failed to delete user role assignments",
					zap.String("userId", userID.String()),
					zap.String("orgId", orgID.String()),
					zap.Error(err),
				)
				return nil, err
			}

			return assignments, nil
		},
	)
}

func (r *repository) ListOrphanedAssignments(
	ctx context.Context,
	now int64,
	limit int,
) ([]permissionports.OrphanedAssignment, error) {
	ctx = dbscope.WithSystem(ctx, orphanedScopeReason)
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]permissionports.OrphanedAssignment, error) {
			orphans, err := keel.Into[permissionports.OrphanedAssignment](
				ctx,
				r.db.Keel(ctx),
				orphanedAssignmentsQuery(now, limit),
			)
			if err != nil {
				r.l.Error("failed to list orphaned role assignments", zap.Error(err))
				return nil, err
			}
			if orphans == nil {
				return []permissionports.OrphanedAssignment{}, nil
			}

			return orphans, nil
		},
	)
}

func orphanedAssignmentsQuery(now int64, limit int) *keel.Query[*rbac.UserRoleAssignment] {
	ura := permissiondb.UserRoleAssignments
	o := tenantdb.Organizations
	u := tenantdb.Users
	m := tenantdb.OrganizationMemberships

	return ura.Query().
		AllTenants().
		Join(o, o.ID.EqCol(ura.OrganizationID)).
		Join(u, u.ID.EqCol(ura.UserID)).
		DistinctOn(ura.OrganizationID, ura.UserID).
		Pick(ura.UserID, ura.OrganizationID, o.BusinessUnitID).
		Where(
			u.CurrentOrganizationID.NeCol(ura.OrganizationID),
			keel.NotExists(m.Query().AllTenants().Where(
				m.UserID.EqCol(ura.UserID),
				m.OrganizationID.EqCol(ura.OrganizationID),
				keel.Or(m.ExpiresAt.IsNull(), m.ExpiresAt.Gt(now)),
			)),
		).
		Order(ura.OrganizationID.Asc(), ura.UserID.Asc()).
		Limit(limit)
}

func (r *repository) CreateResourcePermission(
	ctx context.Context,
	rp *rbac.ResourcePermission,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if err := permissiondb.ResourcePermissions.Create(ctx, r.db.Keel(ctx), rp); err != nil {
			r.l.Error(
				"failed to create resource permission",
				zap.String("roleId", rp.RoleID.String()),
				zap.String("resource", rp.Resource),
				zap.Error(err),
			)
			return rolePermissionErrors.Map(err, rp.ID.String())
		}

		return nil
	})
}

func (r *repository) UpdateResourcePermission(
	ctx context.Context,
	rp *rbac.ResourcePermission,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		p := permissiondb.ResourcePermissions
		sets := make([]keel.Assignment, 0, 3)
		if rp.Resource != "" {
			sets = append(sets, p.Resource.To(rp.Resource))
		}
		if len(rp.Operations) > 0 {
			sets = append(
				sets,
				p.Operations.To(operationNames(permission.Normalize(rp.Operations))),
			)
		}
		if rp.DataScope != "" {
			sets = append(sets, p.DataScope.To(rp.DataScope))
		}
		if len(sets) == 0 {
			sets = append(sets, p.UpdatedAt.To(timeutils.NowUnix()))
		}

		rows, err := p.Where(p.ID.Eq(rp.ID), p.RoleID.Eq(rp.RoleID)).
			UpdateReturning(ctx, r.db.Keel(ctx), sets...)
		if err != nil {
			r.l.Error(
				"failed to update resource permission",
				zap.String("id", rp.ID.String()),
				zap.Error(err),
			)
			return rolePermissionErrors.Map(err, rp.ID.String())
		}
		if err = dberror.CheckFoundCount(
			int64(len(rows)),
			rolePermissionErrors.Entity,
		); err != nil {
			return err
		}

		*rp = *rows[0]
		return nil
	})
}

func operationNames(ops []permission.Operation) []string {
	names := make([]string, len(ops))
	for i, op := range ops {
		names[i] = string(op)
	}
	return names
}

func (r *repository) DeleteResourcePermission(
	ctx context.Context,
	req permissionports.DeleteResourcePermissionRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		p := permissiondb.ResourcePermissions
		deleted, err := p.Where(p.ID.Eq(req.PermissionID), p.RoleID.Eq(req.RoleID)).
			Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error(
				"failed to delete resource permission",
				zap.String("id", req.PermissionID.String()),
				zap.Error(err),
			)
			return err
		}

		return dberror.CheckFoundCount(deleted, rolePermissionErrors.Entity)
	})
}

func (r *repository) GetResourcePermissionsByRoleID(
	ctx context.Context,
	roleID pulid.ID,
) ([]*rbac.ResourcePermission, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*rbac.ResourcePermission, error) {
			p := permissiondb.ResourcePermissions
			permissions, err := p.Where(p.RoleID.Eq(roleID)).All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error(
					"failed to get resource permissions",
					zap.String("roleId", roleID.String()),
					zap.Error(err),
				)
				return nil, err
			}

			return permissions, nil
		},
	)
}

func (r *repository) ListUsersWithPermission(
	ctx context.Context,
	req permissionports.ListUsersWithPermissionRequest,
) ([]permissionports.PermittedUser, error) {
	if req.OrganizationID.IsNotNil() && req.BusinessUnitID.IsNotNil() {
		ctx = dbscope.WithTenant(ctx, dbscope.Tenant{
			OrganizationID: req.OrganizationID,
			BusinessUnitID: req.BusinessUnitID,
		})
	}

	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]permissionports.PermittedUser, error) {
			users, err := keel.Into[permissionports.PermittedUser](
				ctx,
				r.db.Keel(ctx),
				usersWithPermissionQuery(&req),
			)
			if err != nil {
				r.l.Error(
					"failed to list users with permission",
					zap.String("resource", req.Resource.String()),
					zap.String("permissionOperation", string(req.Operation)),
					zap.Error(err),
				)
				return nil, err
			}
			if users == nil {
				return []permissionports.PermittedUser{}, nil
			}

			return users, nil
		},
	)
}

func usersWithPermissionQuery(
	req *permissionports.ListUsersWithPermissionRequest,
) *keel.Query[*tenant.User] {
	ura := permissiondb.UserRoleAssignments
	ro := permissiondb.Roles
	parent := ro.As(parentRoleAlias)
	rp := permissiondb.ResourcePermissions
	u := tenantdb.Users

	held := keel.RecursiveCTE(
		heldRolesName,
		[]string{heldUserColumn, heldRoleColumn, heldParentColumn},
		ura.Join(ro, ro.ID.EqCol(ura.RoleID)).
			Pick(ura.UserID, ro.ID, ro.ParentRoleIDs).
			Where(keel.Or(ura.ExpiresAt.IsNull(), ura.ExpiresAt.Gt(req.Now))),
	)
	held.Recurse(parent.
		Pick(held.Col(heldUserColumn), parent.ID, parent.ParentRoleIDs).
		Join(held, parent.ID.InArray(keel.StringArrayOf(held.Col(heldParentColumn))))).
		Distinct()

	return u.With(held).
		Pick(u.ID, u.Name, u.Username, u.EmailAddress, u.ProfilePicURL.Coalesce(""), u.Locale).
		Where(
			u.Status.Eq(domaintypes.StatusActive),
			keel.Exists(held.Query().
				Join(rp, rp.RoleID.EqCol(held.Col(heldRoleColumn))).
				Where(
					held.Col(heldUserColumn).EqCol(u.ID),
					rp.Resource.Eq(req.Resource.String()),
					rp.Operations.Overlaps(grantingOperations(req.Operation)...),
				)),
		).
		Order(u.Name.Asc())
}

func grantingOperations(op permission.Operation) []string {
	granting := permission.GrantingOperations(op)
	out := make([]string, len(granting))
	for i, candidate := range granting {
		out[i] = string(candidate)
	}
	return out
}
