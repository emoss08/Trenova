package seeder

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/redis/repositories"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type PermissionCacheInvalidator interface {
	InvalidateOrganization(ctx context.Context, orgID pulid.ID) error
}

func newPermissionCache(cfg *config.Config) (*redis.Client, PermissionCacheInvalidator) {
	cacheConfig := cfg.GetCacheConfig()
	client := redis.NewClient(&redis.Options{
		Addr:     cacheConfig.GetRedisAddr(),
		Username: cacheConfig.Username,
		Password: cacheConfig.Password,
		DB:       cacheConfig.DB,
	})

	return client, repositories.NewPermissionCacheRepository(
		repositories.PermissionCacheRepositoryParams{Client: client, Logger: zap.NewNop()},
	)
}

func invalidatePermissionCaches(
	ctx context.Context,
	cache PermissionCacheInvalidator,
	orgIDs []pulid.ID,
) error {
	if cache == nil {
		return nil
	}

	for _, orgID := range orgIDs {
		if err := cache.InvalidateOrganization(ctx, orgID); err != nil {
			return fmt.Errorf("refresh cached permissions for organization %s: %w", orgID, err)
		}
	}

	return nil
}
