package rolerepository

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/pulid"
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

func NewRepository(p Params) ports.RoleRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.role-repository"),
	}
}

func (r *repository) GetByID(
	ctx context.Context,
	id pulid.ID,
) (*permission.Role, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("roleID", id.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(permission.Role)
	if err = db.NewSelect().
		Model(entity).
		Where("r.id = ?", id).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Role")
	}

	return entity, nil
}

func (r *repository) GetByBusinessUnit(
	ctx context.Context,
	businessUnitID pulid.ID,
) ([]*permission.Role, error) {
	log := r.l.With(
		zap.String("operation", "GetByBusinessUnit"),
		zap.String("buID", businessUnitID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*permission.Role, 0)
	if err = db.NewSelect().
		Model(&entities).
		Where("r.business_unit_id = ?", businessUnitID).
		Order("r.level", "r.name").
		Scan(ctx); err != nil {
		log.Error("failed to scan roles", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetUserRoles(
	ctx context.Context,
	userID, organizationID pulid.ID,
) ([]*permission.Role, error) {
	log := r.l.With(
		zap.String("operation", "GetUserRoles"),
		zap.String("userID", userID.String()),
		zap.String("orgID", organizationID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*permission.Role, 0)
	if err = db.NewSelect().
		Model(&entities).
		Join("INNER JOIN user_organization_roles uor ON uor.role_id = r.id").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("uor.user_id = ?", userID).
				Where("uor.organization_id = ?", organizationID).
				Where("(uor.expires_at IS NULL OR uor.expires_at > ?)", time.Now().Unix())
		}).
		Order("r.level", "r.name").
		Scan(ctx); err != nil {
		log.Error("failed to scan user roles", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) HasAdminRole(
	ctx context.Context,
	userID, organizationID pulid.ID,
) (bool, error) {
	log := r.l.With(
		zap.String("operation", "HasAdminRole"),
		zap.String("userID", userID.String()),
		zap.String("orgID", organizationID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return false, err
	}

	exists, err := db.NewSelect().
		Model((*permission.Role)(nil)).
		Join("INNER JOIN user_organization_roles uor ON uor.role_id = r.id").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("uor.user_id = ?", userID).
				Where("uor.organization_id = ?", organizationID).
				Where("r.is_admin = ?", true).
				Where("(uor.expires_at IS NULL OR uor.expires_at > ?)", time.Now().Unix())
		}).
		Exists(ctx)
	if err != nil {
		log.Error("failed to check admin role", zap.Error(err))
		return false, err
	}

	log.Debug("admin role check completed", zap.Bool("hasAdmin", exists))
	return exists, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *permission.Role,
) error {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("roleID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	if _, err = db.NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		log.Error("failed to insert role", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *permission.Role,
) error {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("roleID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	result, err := db.NewUpdate().
		Model(entity).
		WherePK().
		Exec(ctx)
	if err != nil {
		log.Error("failed to update role", zap.Error(err))
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Error("failed to get rows affected", zap.Error(err))
		return err
	}

	if rowsAffected == 0 {
		return dberror.HandleNotFoundError(err, "Role")
	}

	return nil
}

func (r *repository) Delete(
	ctx context.Context,
	id pulid.ID,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("roleID", id.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	result, err := db.NewDelete().
		Model((*permission.Role)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete role", zap.Error(err))
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Error("failed to get rows affected", zap.Error(err))
		return err
	}

	if rowsAffected == 0 {
		return dberror.HandleNotFoundError(err, "Role")
	}

	return nil
}

func (r *repository) AssignToUser(
	ctx context.Context,
	userID, organizationID, roleID pulid.ID,
	assignedBy pulid.ID,
	expiresAt *time.Time,
) error {
	log := r.l.With(
		zap.String("operation", "AssignToUser"),
		zap.String("userID", userID.String()),
		zap.String("orgID", organizationID.String()),
		zap.String("roleID", roleID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	var expiresAtUnix *int64
	if expiresAt != nil {
		unix := expiresAt.Unix()
		expiresAtUnix = &unix
	}

	assignment := &tenant.OrganizationMembership{
		UserID:         userID,
		OrganizationID: organizationID,
		RoleIDs:        []pulid.ID{roleID},
		JoinedAt:       time.Now().Unix(),
		GrantedByID:    assignedBy,
		ExpiresAt:      expiresAtUnix,
	}

	if _, err = db.NewInsert().
		Model(assignment).
		On("CONFLICT (user_id, organization_id) DO UPDATE").
		Set("joined_at = EXCLUDED.joined_at").
		Set("granted_by_id = EXCLUDED.granted_by_id").
		Set("expires_at = EXCLUDED.expires_at").
		Exec(ctx); err != nil {
		log.Error("failed to assign role to user", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) RemoveFromUser(
	ctx context.Context,
	userID, organizationID, roleID pulid.ID,
) error {
	log := r.l.With(
		zap.String("operation", "RemoveFromUser"),
		zap.String("userID", userID.String()),
		zap.String("orgID", organizationID.String()),
		zap.String("roleID", roleID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	result, err := db.NewDelete().
		Model((*tenant.OrganizationMembership)(nil)).
		Where("user_id = ?", userID).
		Where("organization_id = ?", organizationID).
		Where("role_ids @> ARRAY[?]", roleID).
		Exec(ctx)
	if err != nil {
		log.Error("failed to remove role from user", zap.Error(err))
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Error("failed to get rows affected", zap.Error(err))
		return err
	}

	if rowsAffected == 0 {
		return dberror.HandleNotFoundError(err, "RoleAssignment")
	}

	return nil
}

func (r *repository) GetRoleHierarchy(
	ctx context.Context,
	roleID pulid.ID,
) ([]*permission.Role, error) {
	log := r.l.With(
		zap.String("operation", "GetRoleHierarchy"),
		zap.String("roleID", roleID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	var roles []*permission.Role
	if err = db.NewSelect().
		Model(&roles).
		With("RECURSIVE role_hierarchy", db.NewSelect().
			Model((*permission.Role)(nil)).
			Where("id = ?", roleID).
			UnionAll(
				db.NewSelect().
					Model((*permission.Role)(nil)).
					ColumnExpr("r.*").
					Join("INNER JOIN role_hierarchy rh ON rh.parent_roles && ARRAY[r.id]"),
			),
		).
		TableExpr("role_hierarchy").
		Scan(ctx); err != nil {
		log.Error("failed to get role hierarchy", zap.Error(err))
		return nil, err
	}

	return roles, nil
}
