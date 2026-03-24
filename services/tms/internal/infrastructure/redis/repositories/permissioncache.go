package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/redishelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	permissionCachePrefix = "perms"
	roleUsersPrefix       = "role_users"
	orgUsersPrefix        = "org_users"
)

type PermissionCacheRepositoryParams struct {
	fx.In

	Client *redis.Client
	Logger *zap.Logger
}

type permissionCacheRepository struct {
	client *redis.Client
	l      *zap.Logger
}

func NewPermissionCacheRepository(
	p PermissionCacheRepositoryParams,
) repositories.PermissionCacheRepository {
	return &permissionCacheRepository{
		client: p.Client,
		l:      p.Logger.Named("redis.permission-cache-repository"),
	}
}

func (r *permissionCacheRepository) Get(
	ctx context.Context,
	userID, orgID pulid.ID,
) (*repositories.CachedPermissions, error) {
	log := r.l.With(
		zap.String("operation", "Get"),
		zap.String("userID", userID.String()),
		zap.String("orgID", orgID.String()),
	)

	perms := new(repositories.CachedPermissions)
	if err := redishelpers.GetJSON(ctx, r.client, r.getPermissionKey(userID, orgID), perms); err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil //nolint:nilnil // nil is valid for redis.Nil
		}

		log.Error("failed to get permissions from cache", zap.Error(err))
		return nil, err
	}

	return perms, nil
}

func (r *permissionCacheRepository) Set(
	ctx context.Context,
	userID, orgID pulid.ID,
	perms *repositories.CachedPermissions,
	ttl time.Duration,
) error {
	log := r.l.With(
		zap.String("operation", "Set"),
		zap.String("userID", userID.String()),
		zap.String("orgID", orgID.String()),
	)

	if err := redishelpers.SetJSON(ctx, r.client, r.getPermissionKey(userID, orgID), perms, ttl); err != nil {
		log.Error("failed to set permissions in cache", zap.Error(err))
		return err
	}

	return nil
}

func (r *permissionCacheRepository) Delete(ctx context.Context, userID, orgID pulid.ID) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("userID", userID.String()),
		zap.String("orgID", orgID.String()),
	)

	if err := r.client.Del(ctx, r.getPermissionKey(userID, orgID)).Err(); err != nil {
		log.Error("failed to delete permissions from cache", zap.Error(err))
		return err
	}

	return nil
}

func (r *permissionCacheRepository) InvalidateByRole(
	ctx context.Context,
	roleID pulid.ID,
	roleRepo repositories.RoleRepository,
) error {
	log := r.l.With(
		zap.String("operation", "InvalidateByRole"),
		zap.String("roleID", roleID.String()),
	)

	impactedUsers, err := roleRepo.GetUsersWithRole(ctx, roleID)
	if err != nil {
		log.Error("failed to get users with role", zap.Error(err))
		return err
	}

	if len(impactedUsers) == 0 {
		return nil
	}

	keys := make([]string, len(impactedUsers))
	for i, user := range impactedUsers {
		keys[i] = r.getPermissionKey(user.UserID, user.OrganizationID)
	}

	if err = r.client.Del(ctx, keys...).Err(); err != nil {
		log.Error(
			"failed to invalidate permission cache for users",
			zap.Error(err),
			zap.Int("userCount", len(keys)),
		)
		return err
	}

	log.Debug("invalidated permission cache for users", zap.Int("userCount", len(keys)))
	return nil
}

func (r *permissionCacheRepository) InvalidateOrganization(
	ctx context.Context,
	orgID pulid.ID,
) error {
	log := r.l.With(
		zap.String("operation", "InvalidateOrganization"),
		zap.String("orgID", orgID.String()),
	)

	pattern := fmt.Sprintf("%s:*:%s", permissionCachePrefix, orgID.String())

	var cursor uint64
	var keysToDelete []string

	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			log.Error("failed to scan for organization permission keys", zap.Error(err))
			return err
		}

		keysToDelete = append(keysToDelete, keys...)
		cursor = nextCursor

		if cursor == 0 {
			break
		}
	}

	if len(keysToDelete) == 0 {
		return nil
	}

	if err := r.client.Del(ctx, keysToDelete...).Err(); err != nil {
		log.Error("failed to delete organization permission keys", zap.Error(err))
		return err
	}

	log.Debug(
		"invalidated permission cache for organization",
		zap.Int("keyCount", len(keysToDelete)),
	)
	return nil
}

func (r *permissionCacheRepository) getPermissionKey(userID, orgID pulid.ID) string {
	return fmt.Sprintf("%s:%s:%s", permissionCachePrefix, userID.String(), orgID.String())
}
