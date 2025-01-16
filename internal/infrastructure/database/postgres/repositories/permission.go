package repositories

import (
	"context"
	"time"

	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/trenova-app/transport/internal/core/domain/permission"
	"github.com/trenova-app/transport/internal/core/domain/permission/permissiongrant"
	"github.com/trenova-app/transport/internal/core/ports/db"
	"github.com/trenova-app/transport/internal/core/ports/repositories"
	"github.com/trenova-app/transport/internal/pkg/logger"
	"github.com/trenova-app/transport/pkg/types/pulid"
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

func (pr *permissionRepository) GetUserPermissions(ctx context.Context, userID pulid.ID) ([]*permission.Permission, error) {
	dba, err := pr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := pr.l.With().
		Str("operation", "GetUserPermissions").
		Str("userId", userID.String()).
		Logger()

	// Try to get from cache first
	permissions, err := pr.cache.GetUserPermissions(ctx, userID)
	if err == nil && len(permissions) > 0 {
		log.Trace().Int("count", len(permissions)).Msg("got permissions from cache")
		return permissions, nil
	}

	// Get permissions from database
	var dbPermissions []*permission.Permission
	err = dba.NewSelect().
		Model(&dbPermissions).
		Join("JOIN role_permissions rp ON rp.permission_id = perm.id").
		Join("JOIN user_roles ur ON ur.role_id = rp.role_id").
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

	permissions = make([]*permission.Permission, 0, len(dbPermissions))
	permissions = append(permissions, dbPermissions...)

	for _, grant := range grants {
		if grant.Permission != nil {
			if len(grant.FieldOverrides) > 0 {
				grant.Permission.FieldPermissions = grant.FieldOverrides
			}
			permissions = append(permissions, grant.Permission)
		}
	}

	// Cache the permissions
	if err = pr.cache.SetUserPermissions(ctx, userID, permissions); err != nil {
		log.Warn().Err(err).Msg("failed to cache user permissions")
	}

	log.Debug().Int("count", len(permissions)).Msg("got permissions from database")
	return permissions, nil
}

func (pr *permissionRepository) GetUserRoles(ctx context.Context, userID pulid.ID) ([]*string, error) {
	dba, err := pr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := pr.l.With().
		Str("operation", "GetUserRoles").
		Str("userId", userID.String()).
		Logger()

	// Try to get from cache first
	roles, err := pr.cache.GetUserRoles(ctx, userID)
	if err == nil && len(roles) > 0 {
		log.Trace().Int("count", len(roles)).Msg("got roles from cache")
		return roles, nil
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

	roleNames := make([]*string, len(dbRoles))
	for i, role := range dbRoles {
		roleNames[i] = &role.Name
	}

	// Cache the roles
	if err = pr.cache.SetUserRoles(ctx, userID, roleNames); err != nil {
		log.Warn().Err(err).Msg("failed to cache user roles")
	}

	log.Trace().Int("count", len(roleNames)).Msg("got roles from database")
	return roleNames, nil
}

func (pr *permissionRepository) InvalidateUserPermissions(ctx context.Context, userID pulid.ID) error {
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

func (pr *permissionRepository) GetRolesAndPermissions(ctx context.Context, userID pulid.ID) (*permission.RolesAndPermissions, error) {
	permissions, err := pr.GetUserPermissions(ctx, userID)
	if err != nil {
		return nil, eris.Wrap(err, "failed to get user permissions")
	}

	roles, err := pr.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, eris.Wrap(err, "failed to get user roles")
	}

	return &permission.RolesAndPermissions{
		Roles:       roles,
		Permissions: permissions,
	}, nil
}
