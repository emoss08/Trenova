package repositories

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/permission/permissiongrant"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type PermissionRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
	Cache  repositories.PermissionCacheRepository
}

type permissionRepository struct {
	db    db.Connection
	l     *zerolog.Logger
	cache repositories.PermissionCacheRepository
}

func NewPermissionRepository(p PermissionRepositoryParams) repositories.PermissionRepository {
	log := p.Logger.With().
		Str("repository", "permission").
		Str("component", "database").
		Logger()

	return &permissionRepository{
		db:    p.DB,
		l:     &log,
		cache: p.Cache,
	}
}

func (pr *permissionRepository) GetUserPermissions(
	ctx context.Context,
	userID pulid.ID,
) ([]*permission.Permission, error) {
	log := pr.l.With().
		Str("operation", "GetUserPermissions").
		Str("userId", userID.String()).
		Logger()

	// Try to get from cache first
	permissions, err := pr.cache.GetUserPermissions(ctx, userID)
	if err == nil && len(permissions) > 0 {
		log.Debug().Int("count", len(permissions)).Msg("got permissions from cache")
		return permissions, nil
	}

	// On cache miss, get from database
	dba, err := pr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	// Get permissions from database with role information
	var dbPermissions []*permission.Permission
	err = dba.NewSelect().
		Model(&dbPermissions).
		Join("JOIN role_permissions rp ON rp.permission_id = perm.id").
		Join("JOIN user_roles ur ON ur.role_id = rp.role_id").
		ColumnExpr("perm.*").
		ColumnExpr("rp.role_id AS role_id").
		Where("ur.user_id = ?", userID).
		Scan(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to get user permissions")
	}

	// Get permission grants
	grants := make([]*permissiongrant.Grant, 0)
	err = dba.NewSelect().
		Model(&grants).
		Relation("Permission").
		Where("pg.user_id = ?", userID).
		Where("pg.status = ?", permission.StatusActive).
		Where("pg.expires_at IS NULL OR pg.expires_at > ?", time.Now().Unix()).
		Scan(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to get user permission grants")
	}

	// Combine permissions from roles and grants
	allPermissions := make([]*permission.Permission, 0, len(dbPermissions)+len(grants))
	allPermissions = append(allPermissions, dbPermissions...)

	for _, grant := range grants {
		if grant.Permission != nil {
			if len(grant.FieldOverrides) > 0 {
				grant.Permission.FieldPermissions = grant.FieldOverrides
			}
			allPermissions = append(allPermissions, grant.Permission)
		}
	}

	// Cache the permissions
	if len(allPermissions) > 0 {
		if err = pr.cache.SetUserPermissions(ctx, userID, allPermissions); err != nil {
			log.Warn().Err(err).Msg("failed to cache user permissions")
		}
	}

	log.Debug().Int("count", len(allPermissions)).Msg("got permissions from database")
	return allPermissions, nil
}

func (pr *permissionRepository) GetUserRoles(
	ctx context.Context,
	userID pulid.ID,
) ([]*string, error) {
	log := pr.l.With().
		Str("operation", "GetUserRoles").
		Str("userId", userID.String()).
		Logger()

	// Try to get from cache first
	roles, err := pr.cache.GetUserRoles(ctx, userID)
	if err == nil && len(roles) > 0 {
		log.Debug().Int("count", len(roles)).Msg("got roles from cache")
		return roles, nil
	}

	// On cache miss, get from database
	dba, err := pr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	// Get roles from database
	dbRoles := make([]*permission.Role, 0)
	err = dba.NewSelect().
		Model(&dbRoles).
		Join("JOIN user_roles ur ON ur.role_id = r.id").
		Where("ur.user_id = ?", userID).
		Where("r.status = ?", permission.StatusActive).
		Where("r.expires_at IS NULL OR r.expires_at > ?", time.Now().Unix()).
		Scan(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to get user roles")
	}

	// Extract just the role names for the cache format
	roleNames := make([]*string, len(dbRoles))
	for i, role := range dbRoles {
		roleNames[i] = &role.Name
	}

	// Cache the roles
	if len(roleNames) > 0 {
		if err = pr.cache.SetUserRoles(ctx, userID, roleNames); err != nil {
			log.Warn().Err(err).Msg("failed to cache user roles")
		}
	}

	log.Debug().Int("count", len(roleNames)).Msg("got roles from database")
	return roleNames, nil
}

func (pr *permissionRepository) InvalidateUserPermissions(
	ctx context.Context,
	userID pulid.ID,
) error {
	log := pr.l.With().
		Str("operation", "InvalidateUserPermissions").
		Str("userId", userID.String()).
		Logger()

	if err := pr.cache.InvalidateAllUserData(ctx, userID); err != nil {
		log.Error().Err(err).Msg("failed to invalidate user cache")
		return eris.Wrap(err, "invalidate user cache")
	}

	log.Debug().Msg("invalidated user permissions and roles")
	return nil
}

func (pr *permissionRepository) GetRolesAndPermissions(
	ctx context.Context,
	userID pulid.ID,
) (*permission.RolesAndPermissions, error) {
	log := pr.l.With().
		Str("operation", "GetRolesAndPermissions").
		Str("userId", userID.String()).
		Logger()

	// First try to get from cache
	permissions, permErr := pr.cache.GetUserPermissions(ctx, userID)
	roles, rolesErr := pr.cache.GetUserRoles(ctx, userID)

	// If cache hit for both, return early
	if permErr == nil && rolesErr == nil && len(roles) > 0 {
		log.Debug().
			Int("roleCount", len(roles)).
			Int("permissionCount", len(permissions)).
			Msg("got roles and permissions from cache")

		return &permission.RolesAndPermissions{
			Roles:       roles,
			Permissions: permissions,
		}, nil
	}

	// If cache miss, load from database
	dba, err := pr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	// Get roles with their permissions in one query
	var dbRoles []*permission.Role
	err = dba.NewSelect().
		Model(&dbRoles).
		Relation("Permissions").
		Join("JOIN user_roles ur ON ur.role_id = r.id").
		Where("ur.user_id = ?", userID).
		Where("r.status = ?", permission.StatusActive).
		Where("r.expires_at IS NULL OR r.expires_at > ?", time.Now().Unix()).
		Scan(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to get user roles with permissions")
	}

	// Get permission grants
	grants := make([]*permissiongrant.Grant, 0)
	err = dba.NewSelect().
		Model(&grants).
		Relation("Permission").
		Where("pg.user_id = ?", userID).
		Where("pg.status = ?", permission.StatusActive).
		Where("pg.expires_at IS NULL OR pg.expires_at > ?", time.Now().Unix()).
		Scan(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to get user permission grants")
	}

	// Extract role names for cache and collect all permissions
	roleNames := make([]*string, len(dbRoles))
	allPermissions := make([]*permission.Permission, 0)

	for i, role := range dbRoles {
		roleNames[i] = &role.Name
		allPermissions = append(allPermissions, role.Permissions...)
	}

	// Add grant permissions
	for _, grant := range grants {
		if grant.Permission != nil {
			if len(grant.FieldOverrides) > 0 {
				grant.Permission.FieldPermissions = grant.FieldOverrides
			}
			allPermissions = append(allPermissions, grant.Permission)
		}
	}

	// Cache the data for future requests
	if len(roleNames) > 0 {
		if err = pr.cache.SetUserRoles(ctx, userID, roleNames); err != nil {
			log.Warn().Err(err).Msg("failed to cache user roles")
		}
	}

	if len(allPermissions) > 0 {
		if err = pr.cache.SetUserPermissions(ctx, userID, allPermissions); err != nil {
			log.Warn().Err(err).Msg("failed to cache user permissions")
		}
	}

	log.Debug().
		Int("roleCount", len(dbRoles)).
		Int("permissionCount", len(allPermissions)).
		Msg("got roles and permissions from database")

	return &permission.RolesAndPermissions{
		Roles:         roleNames,
		Permissions:   allPermissions,
		CompleteRoles: dbRoles,
	}, nil
}
