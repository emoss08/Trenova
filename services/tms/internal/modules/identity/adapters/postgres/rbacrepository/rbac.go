package rbacrepository

import (
	"context"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission/permissiondb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// rolePermissionCountSQL counts the distinct resource/operation grants a role carries,
// following the same inheritance edge the permission engine follows when it merges a
// role's permissions: parent_role_ids, walked transitively. Counting only a role's own
// resource_permissions rows would report 0 for a role that exists purely to inherit.
const rolePermissionCountSQL = `
WITH RECURSIVE role_closure AS (
	SELECT r.id AS root_id, r.id AS role_id, r.parent_role_ids
	FROM roles r
	WHERE r.id = ANY($1::text[])
	UNION
	SELECT c.root_id, p.id, p.parent_role_ids
	FROM role_closure c
	JOIN roles p ON p.id = ANY(c.parent_role_ids)
)
SELECT c.root_id AS role_id,
	COUNT(DISTINCT rp.resource || ':' || op) AS permission_count
FROM role_closure c
LEFT JOIN resource_permissions rp ON rp.role_id = c.role_id
LEFT JOIN LATERAL unnest(rp.operations) AS op ON TRUE
GROUP BY c.root_id`

var (
	edgeErrors       = dberror.Mapping{Entity: "Role hierarchy edge"}
	roleErrors       = dberror.Mapping{Entity: "Role"}
	constraintErrors = dberror.Mapping{Entity: "Role constraint"}
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

func New(p Params) permissionports.RBACRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.rbac-repository"),
	}
}

func forTenant[T any](
	ctx context.Context,
	q *keel.Query[T],
	preds ...keel.Pred,
) *keel.Query[T] {
	if dbscope.IsSystem(ctx) {
		q = q.AllTenants()
	}

	return q.Where(preds...)
}

type rolePermissionCount struct {
	RoleID          pulid.ID
	PermissionCount int
}

func (r *repository) ListRoleHierarchyEdges(
	ctx context.Context,
	orgID pulid.ID,
) ([]*permission.RoleHierarchyEdge, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*permission.RoleHierarchyEdge, error) {
		rhe := permissiondb.RoleHierarchyEdges
		return forTenant(ctx, rhe.Query(), rhe.OrganizationID.Eq(orgID)).
			Load(rhe.SeniorRole(), rhe.JuniorRole()).
			Order(rhe.CreatedAt.Asc()).
			All(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) UpsertRoleHierarchyEdge(
	ctx context.Context,
	req *permissionports.UpsertRoleHierarchyEdgeRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if req.SeniorRoleID == req.JuniorRoleID {
			return permissionports.ErrCircularRoleHierarchy
		}

		closure, err := r.GetRoleClosure(ctx, []pulid.ID{req.JuniorRoleID})
		if err != nil {
			return err
		}
		if slices.Contains(closure, req.SeniorRoleID) {
			return permissionports.ErrCircularRoleHierarchy
		}

		rhe := permissiondb.RoleHierarchyEdges
		if _, err = rhe.CreateOrSkip(
			ctx,
			r.db.Keel(ctx),
			[]*permission.RoleHierarchyEdge{{
				SeniorRoleID:   req.SeniorRoleID,
				JuniorRoleID:   req.JuniorRoleID,
				OrganizationID: req.OrganizationID,
				BusinessUnitID: req.BusinessUnitID,
				CreatedBy:      req.ActorID,
			}},
			rhe.SeniorRoleID,
			rhe.JuniorRoleID,
		); err != nil {
			return edgeErrors.Map(err, "")
		}

		return r.syncParentRoleIDs(ctx, req.SeniorRoleID)
	})
}

func (r *repository) DeleteRoleHierarchyEdge(
	ctx context.Context,
	req permissionports.DeleteRoleHierarchyEdgeRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		rhe := permissiondb.RoleHierarchyEdges
		db := r.db.Keel(ctx)
		edge, err := forTenant(
			ctx,
			rhe.Query(),
			rhe.ID.Eq(req.EdgeID),
			rhe.OrganizationID.Eq(req.OrganizationID),
		).One(ctx, db)
		if err != nil {
			return edgeErrors.Map(err, req.EdgeID.String())
		}

		if _, err = forTenant(ctx, rhe.Query(), rhe.ID.Eq(edge.ID)).Delete(ctx, db); err != nil {
			return edgeErrors.Map(err, edge.ID.String())
		}

		return r.syncParentRoleIDs(ctx, edge.SeniorRoleID)
	})
}

func (r *repository) GetRoleClosure(ctx context.Context, roleIDs []pulid.ID) ([]pulid.ID, error) {
	if len(roleIDs) == 0 {
		return []pulid.ID{}, nil
	}

	seen := make(map[pulid.ID]struct{}, len(roleIDs))
	queue := make([]pulid.ID, 0, len(roleIDs))
	for _, id := range roleIDs {
		if id.IsNil() {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		queue = append(queue, id)
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		children, err := r.getInheritedRoleIDs(ctx, current)
		if err != nil {
			return nil, err
		}

		for _, id := range children {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			queue = append(queue, id)
		}
	}

	closure := make([]pulid.ID, 0, len(seen))
	for id := range seen {
		closure = append(closure, id)
	}
	return closure, nil
}

func (r *repository) GetAuthorizedRoles(
	ctx context.Context,
	userID, orgID pulid.ID,
) ([]*permission.Role, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*permission.Role, error) {
		ro := permissiondb.Roles
		ura := permissiondb.UserRoleAssignments
		return forTenant(
			ctx,
			ro.Join(ura, ura.RoleID.EqCol(ro.ID)),
			ura.UserID.Eq(userID),
			ura.OrganizationID.Eq(orgID),
			keel.Or(
				ura.ExpiresAt.IsNull(),
				keel.DynOf(ura.ExpiresAt).Gt(
					keel.DynFunc("date_part", keel.DynLit("epoch").Cast(keel.TypeString), keel.DynFunc("now")).
						Cast(keel.TypeInt64),
				),
			),
		).
			Order(ro.Name.Asc()).
			All(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) CountRolePermissions(
	ctx context.Context,
	roleIDs []pulid.ID,
) (map[pulid.ID]int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]int, error) {
		counts := make(map[pulid.ID]int, len(roleIDs))
		if len(roleIDs) == 0 {
			return counts, nil
		}

		ids := make([]string, len(roleIDs))
		for i, id := range roleIDs {
			ids[i] = id.String()
		}

		rows, err := keel.QueryAll(
			ctx,
			r.db.Keel(ctx),
			pgx.RowToStructByPos[rolePermissionCount],
			rolePermissionCountSQL,
			ids,
		)
		if err != nil {
			return nil, err
		}

		for _, row := range rows {
			counts[row.RoleID] = row.PermissionCount
		}
		return counts, nil
	})
}

func (r *repository) ListRoleConstraints(
	ctx context.Context,
	req permissionports.ListRoleConstraintsRequest,
) ([]*permission.RoleConstraint, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*permission.RoleConstraint, error) {
		rc := permissiondb.RoleConstraints
		return withConstraintRoles(forTenant(
			ctx,
			rc.Query(),
			rc.OrganizationID.Eq(req.OrganizationID),
			keel.When(req.Type != "", rc.Type.Eq(req.Type)),
		)).
			Order(rc.CreatedAt.Desc()).
			All(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) GetRoleConstraint(
	ctx context.Context,
	orgID, constraintID pulid.ID,
) (*permission.RoleConstraint, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*permission.RoleConstraint, error) {
		rc := permissiondb.RoleConstraints
		constraint, err := withConstraintRoles(forTenant(
			ctx,
			rc.Query(),
			rc.ID.Eq(constraintID),
			rc.OrganizationID.Eq(orgID),
		)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, constraintErrors.Map(err, constraintID.String())
		}
		return constraint, nil
	})
}

func (r *repository) SaveRoleConstraint(
	ctx context.Context,
	req *permissionports.SaveRoleConstraintRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		constraint := req.Constraint
		rc := permissiondb.RoleConstraints
		db := r.db.Keel(ctx)

		if constraint.ID.IsNil() {
			if err := rc.Create(ctx, db, constraint); err != nil {
				return constraintErrors.Map(err, "")
			}
		} else if err := rc.Save(
			ctx,
			db,
			constraint,
			rc.Name,
			rc.Description,
			rc.Type,
			rc.MaxRoles,
			rc.Enabled,
		); err != nil {
			if dberror.IsNotFoundError(err) {
				return dberror.CreateVersionMismatchError(
					constraintErrors.Entity,
					constraint.ID.String(),
				)
			}
			return constraintErrors.Map(err, constraint.ID.String())
		}

		rcr := permissiondb.RoleConstraintRoles
		if _, err := forTenant(
			ctx,
			rcr.Query(),
			rcr.RoleConstraintID.Eq(constraint.ID),
		).Delete(ctx, db); err != nil {
			return err
		}

		if len(req.RoleIDs) == 0 {
			return nil
		}

		members := make([]*permission.RoleConstraintRole, 0, len(req.RoleIDs))
		for _, roleID := range req.RoleIDs {
			members = append(members, &permission.RoleConstraintRole{
				RoleConstraintID: constraint.ID,
				RoleID:           roleID,
				OrganizationID:   constraint.OrganizationID,
				BusinessUnitID:   constraint.BusinessUnitID,
			})
		}

		return rcr.CreateMany(ctx, db, members)
	})
}

func (r *repository) DeleteRoleConstraint(
	ctx context.Context,
	orgID, constraintID pulid.ID,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		rc := permissiondb.RoleConstraints
		_, err := forTenant(
			ctx,
			rc.Query(),
			rc.ID.Eq(constraintID),
			rc.OrganizationID.Eq(orgID),
		).Delete(ctx, r.db.Keel(ctx))
		return err
	})
}

func (r *repository) ValidateStaticSeparationOfDuty(
	ctx context.Context,
	userID, orgID pulid.ID,
	roleIDs []pulid.ID,
) ([]permissionports.ConstraintViolation, error) {
	assigned := make([]pulid.ID, 0, len(roleIDs))
	if len(roleIDs) == 0 {
		roles, err := r.GetAuthorizedRoles(ctx, userID, orgID)
		if err != nil {
			return nil, err
		}
		for _, role := range roles {
			assigned = append(assigned, role.ID)
		}
	} else {
		assigned = append(assigned, roleIDs...)
	}
	return r.validateConstraints(ctx, orgID, permission.RoleConstraintTypeSSD, assigned)
}

func (r *repository) ValidateDynamicSeparationOfDuty(
	ctx context.Context,
	orgID pulid.ID,
	roleIDs []pulid.ID,
) ([]permissionports.ConstraintViolation, error) {
	return r.validateConstraints(ctx, orgID, permission.RoleConstraintTypeDSD, roleIDs)
}

func (r *repository) RunPreflight(
	ctx context.Context,
	orgID pulid.ID,
) (*permissionports.RBACPreflightReport, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*permissionports.RBACPreflightReport, error) {
			report := &permissionports.RBACPreflightReport{}

			ura := permissiondb.UserRoleAssignments
			userIDs, err := keel.Into[pulid.ID](
				ctx,
				r.db.Keel(ctx),
				forTenant(ctx, ura.Query(), ura.OrganizationID.Eq(orgID)).
					Group(ura.UserID).
					Pick(ura.UserID),
			)
			if err != nil {
				return nil, err
			}

			for _, userID := range userIDs {
				violations, vErr := r.ValidateStaticSeparationOfDuty(ctx, userID, orgID, nil)
				if vErr != nil {
					return nil, vErr
				}
				report.SSDViolations = append(report.SSDViolations, violations...)
			}

			return report, nil
		},
	)
}

func (r *repository) getInheritedRoleIDs(ctx context.Context, roleID pulid.ID) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		db := r.db.Keel(ctx)
		juniors, err := r.juniorRoleIDs(ctx, roleID)
		if err != nil {
			return nil, err
		}
		if len(juniors) > 0 {
			return juniors, nil
		}

		ro := permissiondb.Roles
		role, err := forTenant(ctx, ro.Pick(ro.ParentRoleIDs), ro.ID.Eq(roleID)).One(ctx, db)
		if err != nil {
			return nil, roleErrors.Map(err, roleID.String())
		}
		return role.ParentRoleIDs, nil
	})
}

func (r *repository) juniorRoleIDs(ctx context.Context, seniorRoleID pulid.ID) ([]pulid.ID, error) {
	rhe := permissiondb.RoleHierarchyEdges
	return keel.Into[pulid.ID](
		ctx,
		r.db.Keel(ctx),
		forTenant(ctx, rhe.Pick(rhe.JuniorRoleID), rhe.SeniorRoleID.Eq(seniorRoleID)),
	)
}

func (r *repository) syncParentRoleIDs(ctx context.Context, seniorRoleID pulid.ID) error {
	juniors, err := r.juniorRoleIDs(ctx, seniorRoleID)
	if err != nil {
		return err
	}

	parents := make([]string, len(juniors))
	for i, id := range juniors {
		parents[i] = id.String()
	}

	ro := permissiondb.Roles
	_, err = forTenant(ctx, ro.Query(), ro.ID.Eq(seniorRoleID)).
		Update(ctx, r.db.Keel(ctx), ro.ParentRoleIDs.To(parents))
	return err
}

func withConstraintRoles(
	q *keel.Query[permission.RoleConstraint],
) *keel.Query[permission.RoleConstraint] {
	roles := permissiondb.RoleConstraints.Roles()
	return q.Load(roles.Pick(
		roles.ID,
		roles.BusinessUnitID,
		roles.OrganizationID,
		roles.Name,
		roles.Description,
		roles.MaxSensitivity,
		roles.IsSystem,
	))
}

func (r *repository) validateConstraints(
	ctx context.Context,
	orgID pulid.ID,
	constraintType permission.RoleConstraintType,
	roleIDs []pulid.ID,
) ([]permissionports.ConstraintViolation, error) {
	closure, err := r.GetRoleClosure(ctx, roleIDs)
	if err != nil {
		return nil, err
	}

	activeSet := make(map[pulid.ID]struct{}, len(closure))
	for _, id := range closure {
		activeSet[id] = struct{}{}
	}

	constraints, err := r.ListRoleConstraints(ctx, permissionports.ListRoleConstraintsRequest{
		OrganizationID: orgID,
		Type:           constraintType,
	})
	if err != nil {
		return nil, err
	}

	violations := make([]permissionports.ConstraintViolation, 0)
	for _, constraint := range constraints {
		if !constraint.Enabled {
			continue
		}

		matched := make([]pulid.ID, 0, len(constraint.Roles))
		constraintRoleIDs := make([]pulid.ID, 0, len(constraint.Roles))
		for _, role := range constraint.Roles {
			constraintRoleIDs = append(constraintRoleIDs, role.ID)
			if _, ok := activeSet[role.ID]; ok {
				matched = append(matched, role.ID)
			}
		}

		if len(matched) > constraint.MaxRoles {
			violations = append(violations, permissionports.ConstraintViolation{
				ConstraintID:   constraint.ID,
				ConstraintName: constraint.Name,
				Type:           constraint.Type,
				MaxRoles:       constraint.MaxRoles,
				RoleIDs:        constraintRoleIDs,
				MatchedRoleIDs: matched,
			})
		}
	}

	return violations, nil
}
