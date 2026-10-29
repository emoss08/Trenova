package rbacrepository

import (
	"context"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac/rbacdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

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
	q *keel.Query[*T],
	preds ...keel.Pred,
) *keel.Query[*T] {
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
) ([]*rbac.RoleHierarchyEdge, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*rbac.RoleHierarchyEdge, error) {
		rhe := rbacdb.RoleHierarchyEdges
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

		rhe := rbacdb.RoleHierarchyEdges
		if _, err = rhe.CreateOrSkip(
			ctx,
			r.db.Keel(ctx),
			[]*rbac.RoleHierarchyEdge{{
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
		rhe := rbacdb.RoleHierarchyEdges
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
) ([]*rbac.Role, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*rbac.Role, error) {
		ro := rbacdb.Roles
		ura := rbacdb.UserRoleAssignments
		return forTenant(
			ctx,
			ro.Join(ura, ura.RoleID.EqCol(ro.ID)),
			ura.UserID.Eq(userID),
			ura.OrganizationID.Eq(orgID),
			keel.Or(
				ura.ExpiresAt.IsNull(),
				keel.NowEpoch().LtExpr(ura.ExpiresAt),
			),
		).
			Order(ro.Name.Asc()).
			All(ctx, r.db.Keel(ctx))
	})
}

func rolePermissionCounts(ctx context.Context, roleIDs []pulid.ID) keel.Projection {
	ro := rbacdb.Roles
	parent := ro.As("parent_role")
	closure := keel.RecursiveCTE(
		"role_closure",
		[]string{"root_id", "role_id", "parent_role_ids"},
		forTenant(ctx, ro.Pick(ro.ID, ro.ID, ro.ParentRoleIDs), ro.ID.In(roleIDs...)),
	)
	closure.Recurse(forTenant(
		ctx,
		parent.Pick(closure.Col("root_id"), parent.ID, parent.ParentRoleIDs).
			Join(closure, parent.ID.InArray(keel.StringArrayOf(closure.Col("parent_role_ids")))),
	)).Distinct()

	rp := rbacdb.ResourcePermissions
	op := keel.Rows("op", []string{"op"}, rp.Operations.Unnest())
	root := closure.Col("root_id")

	return forTenant(ctx, closure.Query().
		With(closure).
		LeftJoin(rp, rp.RoleID.EqCol(closure.Col("role_id"))).
		LeftJoinLateral(op).
		Pick(root, keel.CountOf(rp.Resource.Cat(keel.Text(":"), keel.StringOf(op.Col("op")))).Distinct()).
		Group(root))
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

		rows, err := keel.Into[rolePermissionCount](
			ctx,
			r.db.Keel(ctx),
			rolePermissionCounts(ctx, roleIDs),
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
) ([]*rbac.RoleConstraint, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*rbac.RoleConstraint, error) {
		rc := rbacdb.RoleConstraints
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
) (*rbac.RoleConstraint, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*rbac.RoleConstraint, error) {
		rc := rbacdb.RoleConstraints
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
		rc := rbacdb.RoleConstraints
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

		rcr := rbacdb.RoleConstraintRoles
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

		members := make([]*rbac.RoleConstraintRole, 0, len(req.RoleIDs))
		for _, roleID := range req.RoleIDs {
			members = append(members, &rbac.RoleConstraintRole{
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
		rc := rbacdb.RoleConstraints
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
	return r.validateConstraints(ctx, orgID, rbac.RoleConstraintTypeSSD, assigned)
}

func (r *repository) ValidateDynamicSeparationOfDuty(
	ctx context.Context,
	orgID pulid.ID,
	roleIDs []pulid.ID,
) ([]permissionports.ConstraintViolation, error) {
	return r.validateConstraints(ctx, orgID, rbac.RoleConstraintTypeDSD, roleIDs)
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

			ura := rbacdb.UserRoleAssignments
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

		ro := rbacdb.Roles
		role, err := forTenant(ctx, ro.Pick(ro.ParentRoleIDs), ro.ID.Eq(roleID)).One(ctx, db)
		if err != nil {
			return nil, roleErrors.Map(err, roleID.String())
		}
		return role.ParentRoleIDs, nil
	})
}

func (r *repository) juniorRoleIDs(ctx context.Context, seniorRoleID pulid.ID) ([]pulid.ID, error) {
	rhe := rbacdb.RoleHierarchyEdges
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

	ro := rbacdb.Roles
	_, err = forTenant(ctx, ro.Query(), ro.ID.Eq(seniorRoleID)).
		Update(ctx, r.db.Keel(ctx), ro.ParentRoleIDs.To(parents))
	return err
}

func withConstraintRoles(
	q *keel.Query[*rbac.RoleConstraint],
) *keel.Query[*rbac.RoleConstraint] {
	roles := rbacdb.RoleConstraints.Roles()
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
	constraintType rbac.RoleConstraintType,
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
