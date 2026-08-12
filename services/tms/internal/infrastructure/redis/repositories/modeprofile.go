package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/modeprofile"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/redishelpers"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	modeProfileKeyPrefix = "mode-profiles"
	modeProfileCacheTTL  = 5 * time.Minute
)

type cachedModeProfiles struct {
	Profiles []*modeprofile.Profile `json:"profiles"`
}

type ModeProfileCacheParams struct {
	fx.In

	Client *redis.Client
	Logger *zap.Logger
}

type modeProfileCacheRepository struct {
	client *redis.Client
	l      *zap.Logger
}

func NewModeProfileCacheRepository(
	p ModeProfileCacheParams,
) repositories.ModeProfileCacheRepository {
	return &modeProfileCacheRepository{
		client: p.Client,
		l:      p.Logger.Named("redis.mode-profile-cache"),
	}
}

func modeProfileCacheKey(tenantInfo pagination.TenantInfo) string {
	return fmt.Sprintf(
		"%s:%s:%s",
		modeProfileKeyPrefix,
		tenantInfo.OrgID.String(),
		tenantInfo.BuID.String(),
	)
}

func (r *modeProfileCacheRepository) GetActiveProfiles(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*modeprofile.Profile, error) {
	key := modeProfileCacheKey(tenantInfo)

	cached := new(cachedModeProfiles)
	if err := redishelpers.GetJSON(ctx, r.client, key, cached); err != nil {
		if !redishelpers.IsRedisNil(err) {
			r.l.Error("failed to read mode profile cache", zap.Error(err))
		}
		return nil, err
	}

	return cached.Profiles, nil
}

func (r *modeProfileCacheRepository) SetActiveProfiles(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	profiles []*modeprofile.Profile,
) error {
	key := modeProfileCacheKey(tenantInfo)

	if err := redishelpers.SetJSON(
		ctx,
		r.client,
		key,
		&cachedModeProfiles{Profiles: profiles},
		modeProfileCacheTTL,
	); err != nil {
		r.l.Error("failed to write mode profile cache", zap.Error(err))
		return err
	}

	return nil
}

func (r *modeProfileCacheRepository) Invalidate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) error {
	key := modeProfileCacheKey(tenantInfo)

	if err := r.client.Del(ctx, key).Err(); err != nil {
		r.l.Error("failed to invalidate mode profile cache", zap.Error(err))
		return err
	}

	return nil
}
