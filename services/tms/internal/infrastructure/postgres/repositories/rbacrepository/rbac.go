package rbacrepository

import (
	"context"
	"database/sql"
	"errors"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
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

func New(p Params) repositories.RBACRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.rbac-repository"),
	}
}

func (r *repository) ListRoleHierarchyEdges(
	ctx context.Context,
	orgID pulid.ID,
) ([]*permission.RoleHierarchyEdge, error) {
	edges := make([]*permission.RoleHierarchyEdge, 0)
	err := r.db.DB().NewSelect().
		Model(&edges).
		Relation("SeniorRole").
		Relation("JuniorRole").
		Where("rhe.organization_id = ?", orgID).
		Order("rhe.created_at ASC").
		Scan(ctx)
	return edges, err
}

func (r *repository) UpsertRoleHierarchyEdge(
	ctx context.Context,
	req *repositories.UpsertRoleHierarchyEdgeRequest,
) error {
	if req.SeniorRoleID == req.JuniorRoleID {
		return repositories.ErrCircularRoleHierarchy
	}

	closure, err := r.GetRoleClosure(ctx, []pulid.ID{req.JuniorRoleID})
	if err != nil {
		return err
	}
	if slices.Contains(closure, req.SeniorRoleID) {
		return repositories.ErrCircularRoleHierarchy
	}

	edge := &permission.RoleHierarchyEdge{
		SeniorRoleID:   req.SeniorRoleID,
		JuniorRoleID:   req.JuniorRoleID,
		OrganizationID: req.OrganizationID,
		BusinessUnitID: req.BusinessUnitID,
		CreatedBy:      req.ActorID,
	}

	return r.db.DB().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err = tx.NewInsert().
			Model(edge).
			On(`CONFLICT ("senior_role_id", "junior_role_id") DO NOTHING`).
			Exec(ctx); err != nil {
			return err
		}

		return r.syncParentRoleIDs(ctx, tx, req.SeniorRoleID)
	})
}

func (r *repository) DeleteRoleHierarchyEdge(
	ctx context.Context,
	req repositories.DeleteRoleHierarchyEdgeRequest,
) error {
	var seniorRoleID pulid.ID
	err := r.db.DB().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		edge := new(permission.RoleHierarchyEdge)
		if scanErr := tx.NewSelect().
			Model(edge).
			Where("rhe.id = ?", req.EdgeID).
			Where("rhe.organization_id = ?", req.OrganizationID).
			Scan(ctx); scanErr != nil {
			return dberror.HandleNotFoundError(scanErr, "Role hierarchy edge")
		}

		seniorRoleID = edge.SeniorRoleID
		if _, delErr := tx.NewDelete().Model(edge).WherePK().Exec(ctx); delErr != nil {
			return delErr
		}

		return r.syncParentRoleIDs(ctx, tx, seniorRoleID)
	})
	return err
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
	roles := make([]*permission.Role, 0)
	err := r.db.DB().NewSelect().
		Model(&roles).
		Join("JOIN user_role_assignments AS ura ON ura.role_id = r.id").
		Where("ura.user_id = ?", userID).
		Where("ura.organization_id = ?", orgID).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("ura.expires_at IS NULL").
				WhereOr("ura.expires_at > extract(epoch from current_timestamp)::bigint")
		}).
		Order("r.name ASC").
		Scan(ctx)
	return roles, err
}

func (r *repository) ListRoleConstraints(
	ctx context.Context,
	req repositories.ListRoleConstraintsRequest,
) ([]*permission.RoleConstraint, error) {
	constraints := make([]*permission.RoleConstraint, 0)
	query := r.db.DB().NewSelect().
		Model(&constraints).
		Where("rc.organization_id = ?", req.OrganizationID).
		Order("rc.created_at DESC")
	if req.Type != "" {
		query = query.Where("rc.type = ?", req.Type)
	}
	if err := query.Scan(ctx); err != nil {
		return nil, err
	}
	if err := r.loadConstraintRoles(ctx, constraints); err != nil {
		return nil, err
	}
	return constraints, nil
}

func (r *repository) GetRoleConstraint(
	ctx context.Context,
	orgID, constraintID pulid.ID,
) (*permission.RoleConstraint, error) {
	constraint := new(permission.RoleConstraint)
	if err := r.db.DB().NewSelect().
		Model(constraint).
		Where("rc.id = ?", constraintID).
		Where("rc.organization_id = ?", orgID).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Role constraint")
	}
	if err := r.loadConstraintRoles(ctx, []*permission.RoleConstraint{constraint}); err != nil {
		return nil, err
	}
	return constraint, nil
}

func (r *repository) SaveRoleConstraint(
	ctx context.Context,
	req *repositories.SaveRoleConstraintRequest,
) error {
	constraint := req.Constraint
	return r.db.DB().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if constraint.ID.IsNil() {
			if _, err := tx.NewInsert().Model(constraint).Returning("*").Exec(ctx); err != nil {
				return err
			}
		} else {
			result, err := tx.NewUpdate().
				Model(constraint).
				WherePK().
				Where("organization_id = ?", constraint.OrganizationID).
				Where("business_unit_id = ?", constraint.BusinessUnitID).
				Column("name", "description", "type", "max_roles", "enabled", "updated_at").
				Returning("*").
				Exec(ctx)
			if err != nil {
				return err
			}
			if err = dberror.CheckRowsAffected(
				result,
				"Role constraint",
				constraint.ID.String(),
			); err != nil {
				return err
			}
		}

		if _, err := tx.NewDelete().
			Model((*permission.RoleConstraintRole)(nil)).
			Where("role_constraint_id = ?", constraint.ID).
			Where("organization_id = ?", constraint.OrganizationID).
			Where("business_unit_id = ?", constraint.BusinessUnitID).
			Exec(ctx); err != nil {
			return err
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
		if len(members) == 0 {
			return nil
		}
		_, err := tx.NewInsert().Model(&members).Exec(ctx)
		return err
	})
}

func (r *repository) DeleteRoleConstraint(
	ctx context.Context,
	orgID, constraintID pulid.ID,
) error {
	_, err := r.db.DB().NewDelete().
		Model((*permission.RoleConstraint)(nil)).
		Where("id = ?", constraintID).
		Where("organization_id = ?", orgID).
		Exec(ctx)
	return err
}

func (r *repository) ValidateStaticSeparationOfDuty(
	ctx context.Context,
	userID, orgID pulid.ID,
	roleIDs []pulid.ID,
) ([]repositories.ConstraintViolation, error) {
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
) ([]repositories.ConstraintViolation, error) {
	return r.validateConstraints(ctx, orgID, permission.RoleConstraintTypeDSD, roleIDs)
}

func (r *repository) RunPreflight(
	ctx context.Context,
	orgID pulid.ID,
) (*repositories.RBACPreflightReport, error) {
	report := &repositories.RBACPreflightReport{}

	var assignments []struct {
		UserID pulid.ID `bun:"user_id"`
	}
	if err := r.db.DB().NewSelect().
		TableExpr("user_role_assignments").
		ColumnExpr("DISTINCT user_id").
		Where("organization_id = ?", orgID).
		Scan(ctx, &assignments); err != nil {
		return nil, err
	}

	for _, assignment := range assignments {
		violations, err := r.ValidateStaticSeparationOfDuty(ctx, assignment.UserID, orgID, nil)
		if err != nil {
			return nil, err
		}
		report.SSDViolations = append(report.SSDViolations, violations...)
	}

	return report, nil
}

func (r *repository) getInheritedRoleIDs(ctx context.Context, roleID pulid.ID) ([]pulid.ID, error) {
	var edgeIDs []pulid.ID
	if err := r.db.DB().NewSelect().
		Model((*permission.RoleHierarchyEdge)(nil)).
		Column("junior_role_id").
		Where("senior_role_id = ?", roleID).
		Scan(ctx, &edgeIDs); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if len(edgeIDs) > 0 {
		return edgeIDs, nil
	}

	role := new(permission.Role)
	if err := r.db.DB().NewSelect().
		Model(role).
		Column("parent_role_ids").
		Where("r.id = ?", roleID).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Role")
	}
	return role.ParentRoleIDs, nil
}

func (r *repository) syncParentRoleIDs(
	ctx context.Context,
	tx bun.Tx,
	seniorRoleID pulid.ID,
) error {
	var juniorRoleIDs []pulid.ID
	if err := tx.NewSelect().
		Model((*permission.RoleHierarchyEdge)(nil)).
		Column("junior_role_id").
		Where("senior_role_id = ?", seniorRoleID).
		Scan(ctx, &juniorRoleIDs); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	_, err := tx.NewUpdate().
		Model((*permission.Role)(nil)).
		Set("parent_role_ids = ?", pgdialect.Array(juniorRoleIDs)).
		Where("id = ?", seniorRoleID).
		Exec(ctx)
	return err
}

func (r *repository) loadConstraintRoles(
	ctx context.Context,
	constraints []*permission.RoleConstraint,
) error {
	if len(constraints) == 0 {
		return nil
	}

	ids := make([]pulid.ID, 0, len(constraints))
	byID := make(map[pulid.ID]*permission.RoleConstraint, len(constraints))
	for _, constraint := range constraints {
		ids = append(ids, constraint.ID)
		byID[constraint.ID] = constraint
	}

	var rows []struct {
		ConstraintID   pulid.ID                    `bun:"role_constraint_id"`
		RoleID         pulid.ID                    `bun:"role_id"`
		BusinessUnitID pulid.ID                    `bun:"business_unit_id"`
		OrganizationID pulid.ID                    `bun:"organization_id"`
		Name           string                      `bun:"name"`
		Description    string                      `bun:"description"`
		MaxSensitivity permission.FieldSensitivity `bun:"max_sensitivity"`
		IsSystem       bool                        `bun:"is_system"`
	}
	if err := r.db.DB().NewSelect().
		TableExpr("role_constraint_roles AS rcr").
		ColumnExpr("rcr.role_constraint_id").
		ColumnExpr("r.id AS role_id").
		ColumnExpr("r.business_unit_id").
		ColumnExpr("r.organization_id").
		ColumnExpr("r.name").
		ColumnExpr("r.description").
		ColumnExpr("r.max_sensitivity").
		ColumnExpr("r.is_system").
		Join("JOIN roles AS r ON r.id = rcr.role_id").
		Where("rcr.role_constraint_id IN (?)", bun.List(ids)).
		Scan(ctx, &rows); err != nil {
		return err
	}

	for _, row := range rows {
		if constraint := byID[row.ConstraintID]; constraint != nil {
			constraint.Roles = append(constraint.Roles, &permission.Role{
				ID:             row.RoleID,
				BusinessUnitID: row.BusinessUnitID,
				OrganizationID: row.OrganizationID,
				Name:           row.Name,
				Description:    row.Description,
				MaxSensitivity: row.MaxSensitivity,
				IsSystem:       row.IsSystem,
			})
		}
	}
	return nil
}

func (r *repository) validateConstraints(
	ctx context.Context,
	orgID pulid.ID,
	constraintType permission.RoleConstraintType,
	roleIDs []pulid.ID,
) ([]repositories.ConstraintViolation, error) {
	closure, err := r.GetRoleClosure(ctx, roleIDs)
	if err != nil {
		return nil, err
	}

	activeSet := make(map[pulid.ID]struct{}, len(closure))
	for _, id := range closure {
		activeSet[id] = struct{}{}
	}

	constraints, err := r.ListRoleConstraints(ctx, repositories.ListRoleConstraintsRequest{
		OrganizationID: orgID,
		Type:           constraintType,
	})
	if err != nil {
		return nil, err
	}

	violations := make([]repositories.ConstraintViolation, 0)
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
			violations = append(violations, repositories.ConstraintViolation{
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
