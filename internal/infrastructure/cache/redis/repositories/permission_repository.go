// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/cache/redis"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

const (
	defaultPermissionTTL = 15 * time.Minute
	permKeyPrefix        = "perm:"
	rolesKeyPrefix       = "roles:"
)

type PermissionRepositoryParams struct {
	fx.In

	Cache  *redis.Client
	Logger *logger.Logger
}

type permissionRepository struct {
	cache    *redis.Client
	l        *zerolog.Logger
	cacheTTL time.Duration
}

func NewPermissionRepository(p PermissionRepositoryParams) repositories.PermissionCacheRepository {
	log := p.Logger.With().
		Str("repository", "permission").
		Str("component", "redis").
		Logger()

	return &permissionRepository{
		cache:    p.Cache,
		l:        &log,
		cacheTTL: defaultPermissionTTL,
	}
}

// GetUserRoles retrieves the roles for a user
func (pc *permissionRepository) GetUserRoles(
	ctx context.Context,
	userID pulid.ID,
) ([]*string, error) {
	log := pc.l.With().
		Str("operation", "GetUserRoles").
		Str("userId", userID.String()).
		Logger()

	roles := make([]*string, 0)
	key := pc.formatRolesKey(userID)

	if err := pc.cache.GetJSON(ctx, ".", key, &roles); err != nil {
		if eris.Is(err, redis.ErrNil) {
			log.Debug().Msg("no roles found in cache")
			return nil, eris.New("no roles found in cache")
		}
		return nil, eris.Wrapf(err, "failed to get roles for user %s", userID)
	}

	// log.Debug().
	// 	Int("roleCount", len(roles)).
	// 	Msg("retrieved user roles from cache")

	return roles, nil
}

// SetUserRoles stores the roles for a user
func (pc *permissionRepository) SetUserRoles(
	ctx context.Context,
	userID pulid.ID,
	roles []*string,
) error {
	log := pc.l.With().
		Str("operation", "SetUserRoles").
		Str("userId", userID.String()).
		Logger()

	key := pc.formatRolesKey(userID)
	if err := pc.cache.SetJSON(ctx, ".", key, roles, pc.cacheTTL); err != nil {
		return eris.Wrapf(err, "failed to set roles for user %s", userID)
	}

	log.Debug().
		Int("roleCount", len(roles)).
		Msg("stored user roles in cache")

	return nil
}

// GetUserPermissions retrieves the permissions for a user
func (pc *permissionRepository) GetUserPermissions(
	ctx context.Context,
	userID pulid.ID,
) ([]*permission.Permission, error) {
	log := pc.l.With().
		Str("operation", "GetUserPermissions").
		Str("userId", userID.String()).
		Logger()

	var permissions []*permission.Permission
	key := pc.formatKey(userID)

	if err := pc.cache.GetJSON(ctx, ".", key, &permissions); err != nil {
		if eris.Is(err, redis.ErrNil) {
			log.Debug().Msg("no permissions found in cache")
			return nil, nil
		}
		return nil, eris.Wrapf(err, "failed to get permissions for user %s", userID)
	}

	return permissions, nil
}

// SetUserPermissions stores the permissions for a user
func (pc *permissionRepository) SetUserPermissions(
	ctx context.Context,
	userID pulid.ID,
	permissions []*permission.Permission,
) error {
	log := pc.l.With().
		Str("operation", "SetUserPermissions").
		Str("userId", userID.String()).
		Logger()

	key := pc.formatKey(userID)
	if err := pc.cache.SetJSON(ctx, ".", key, permissions, pc.cacheTTL); err != nil {
		return eris.Wrapf(err, "failed to set permissions for user %s", userID)
	}

	log.Debug().
		Int("permissionCount", len(permissions)).
		Msg("stored user permissions in cache")

	return nil
}

// InvalidateUserPermissions removes the permissions for a user from cache
func (pc *permissionRepository) InvalidateUserPermissions(
	ctx context.Context,
	userID pulid.ID,
) error {
	log := pc.l.With().
		Str("operation", "InvalidateUserPermissions").
		Str("userId", userID.String()).
		Logger()

	key := pc.formatKey(userID)
	if err := pc.cache.Del(ctx, key); err != nil {
		return eris.Wrapf(err, "failed to invalidate permissions for user %s", userID)
	}

	log.Debug().Msg("invalidated user permissions in cache")
	return nil
}

// InvalidateUserRoles removes the roles for a user from cache
func (pc *permissionRepository) InvalidateUserRoles(ctx context.Context, userID pulid.ID) error {
	log := pc.l.With().
		Str("operation", "InvalidateUserRoles").
		Str("userId", userID.String()).
		Logger()

	key := pc.formatRolesKey(userID)
	if err := pc.cache.Del(ctx, key); err != nil {
		return eris.Wrapf(err, "failed to invalidate roles for user %s", userID)
	}

	log.Debug().Msg("invalidated user roles in cache")
	return nil
}

// InvalidateAllUserData removes all cached data for a user
func (pc *permissionRepository) InvalidateAllUserData(ctx context.Context, userID pulid.ID) error {
	log := pc.l.With().
		Str("operation", "InvalidateAllUserData").
		Str("userId", userID.String()).
		Logger()

	// Use pipeline to delete both keys atomically
	pipe := pc.cache.Pipeline()
	pipe.Del(ctx, pc.formatKey(userID))
	pipe.Del(ctx, pc.formatRolesKey(userID))

	if _, err := pipe.Exec(ctx); err != nil {
		return eris.Wrapf(err, "failed to invalidate all data for user %s", userID)
	}

	log.Debug().Msg("invalidated all user data in cache")
	return nil
}

func (pc *permissionRepository) formatKey(userID pulid.ID) string {
	return fmt.Sprintf("%s%s", permKeyPrefix, userID)
}

func (pc *permissionRepository) formatRolesKey(userID pulid.ID) string {
	return fmt.Sprintf("%s%s", rolesKeyPrefix, userID)
}
