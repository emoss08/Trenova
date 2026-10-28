package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	permissionCachePrefix  = "perms:v5"
	permissionEpochPrefix  = "perms:epoch"
	userEpochPrefix        = "perms:uepoch"
	grantFingerprintPrefix = 16
	userEpochTTL           = 24 * time.Hour
)

type PermissionCacheRepositoryParams struct {
	fx.In

	Client *redis.Client
	Logger *zap.Logger
}

type permissionCacheRepository struct {
	client *redis.Client
	l      *zap.Logger
	prefix string
}

func NewPermissionCacheRepository(
	p PermissionCacheRepositoryParams,
) permissionports.PermissionCacheRepository {
	grants := permission.NewRegistry().GrantFingerprint()[:grantFingerprintPrefix]

	return &permissionCacheRepository{
		client: p.Client,
		l:      p.Logger.Named("redis.permission-cache-repository"),
		prefix: permissionCachePrefix + ":" + grants,
	}
}

func (r *permissionCacheRepository) Get(
	ctx context.Context,
	key permissionports.PermissionCacheKey,
) (*permissionports.CachedPermissions, error) {
	log := r.l.With(
		zap.String("operation", "Get"),
		zap.String("userID", key.UserID.String()),
		zap.String("orgID", key.OrgID.String()),
		zap.String("variant", key.Variant),
	)

	pipe := r.client.Pipeline()
	entryCmd := pipe.HGet(ctx, r.getPermissionKey(key.UserID, key.OrgID), key.Variant)
	epochCmd := pipe.Get(ctx, r.getEpochKey(key.OrgID))
	userEpochCmd := pipe.Get(ctx, r.getUserEpochKey(key.UserID, key.OrgID))
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		log.Error("failed to get permissions from cache", zap.Error(err))
		return nil, err
	}

	raw, err := entryCmd.Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil //nolint:nilnil // nil is valid for redis.Nil
	}
	if err != nil {
		log.Error("failed to read cached permissions", zap.Error(err))
		return nil, err
	}

	epoch, err := epochCmd.Int64()
	if errors.Is(err, redis.Nil) {
		return nil, nil //nolint:nilnil // an organization without an epoch has no trusted entries
	}
	if err != nil {
		log.Error("failed to read the organization's permission epoch", zap.Error(err))
		return nil, err
	}

	userEpoch, err := userEpochOf(userEpochCmd)
	if err != nil {
		log.Error("failed to read the user's permission epoch", zap.Error(err))
		return nil, err
	}

	perms := new(permissionports.CachedPermissions)
	if err = sonic.Unmarshal(raw, perms); err != nil {
		log.Error("failed to decode cached permissions", zap.Error(err))
		return nil, err
	}

	current := permissionports.CacheEpoch{Organization: epoch, User: userEpoch}
	if !current.Current(perms) {
		return nil, nil //nolint:nilnil // an entry from an earlier epoch is stale
	}

	return perms, nil
}

func (r *permissionCacheRepository) Set(
	ctx context.Context,
	key permissionports.PermissionCacheKey,
	perms *permissionports.CachedPermissions,
	ttl time.Duration,
) error {
	log := r.l.With(
		zap.String("operation", "Set"),
		zap.String("userID", key.UserID.String()),
		zap.String("orgID", key.OrgID.String()),
		zap.String("variant", key.Variant),
	)

	payload, err := sonic.Marshal(perms)
	if err != nil {
		log.Error("failed to encode permissions for cache", zap.Error(err))
		return err
	}

	redisKey := r.getPermissionKey(key.UserID, key.OrgID)
	pipe := r.client.TxPipeline()
	pipe.HSet(ctx, redisKey, key.Variant, payload)
	pipe.Expire(ctx, redisKey, ttl)
	if _, err = pipe.Exec(ctx); err != nil {
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

	userEpochKey := r.getUserEpochKey(userID, orgID)
	pipe := r.client.TxPipeline()
	pipe.Incr(ctx, userEpochKey)
	pipe.Expire(ctx, userEpochKey, userEpochTTL)
	pipe.Del(ctx, r.getPermissionKey(userID, orgID))
	if _, err := pipe.Exec(ctx); err != nil {
		log.Error("failed to delete permissions from cache", zap.Error(err))
		return err
	}

	return nil
}

func (r *permissionCacheRepository) Epoch(
	ctx context.Context,
	userID, orgID pulid.ID,
) (permissionports.CacheEpoch, error) {
	log := r.l.With(
		zap.String("operation", "Epoch"),
		zap.String("userID", userID.String()),
		zap.String("orgID", orgID.String()),
	)

	userEpoch, err := userEpochOf(r.client.Get(ctx, r.getUserEpochKey(userID, orgID)))
	if err != nil {
		log.Error("failed to read the user's permission epoch", zap.Error(err))
		return permissionports.CacheEpoch{}, err
	}

	key := r.getEpochKey(orgID)
	epoch, err := r.client.Get(ctx, key).Int64()
	if err == nil {
		return permissionports.CacheEpoch{Organization: epoch, User: userEpoch}, nil
	}
	if !errors.Is(err, redis.Nil) {
		log.Error("failed to read permission epoch", zap.Error(err))
		return permissionports.CacheEpoch{}, err
	}

	seed := time.Now().UnixNano()
	if err = r.client.SetNX(ctx, key, seed, 0).Err(); err != nil {
		log.Error("failed to seed permission epoch", zap.Error(err))
		return permissionports.CacheEpoch{}, err
	}

	epoch, err = r.client.Get(ctx, key).Int64()
	if err != nil {
		return permissionports.CacheEpoch{}, err
	}

	return permissionports.CacheEpoch{Organization: epoch, User: userEpoch}, nil
}

func userEpochOf(cmd *redis.StringCmd) (int64, error) {
	epoch, err := cmd.Int64()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}

	return epoch, err
}

func (r *permissionCacheRepository) InvalidateOrganization(
	ctx context.Context,
	orgID pulid.ID,
) error {
	if err := r.client.Incr(ctx, r.getEpochKey(orgID)).Err(); err != nil {
		r.l.Error(
			"failed to advance the organization's permission epoch",
			zap.String("orgID", orgID.String()),
			zap.Error(err),
		)
		return err
	}

	return nil
}

func (r *permissionCacheRepository) getPermissionKey(userID, orgID pulid.ID) string {
	return fmt.Sprintf("%s:%s:%s", r.prefix, userID.String(), orgID.String())
}

func (r *permissionCacheRepository) getEpochKey(orgID pulid.ID) string {
	return fmt.Sprintf("%s:%s", permissionEpochPrefix, orgID.String())
}

func (r *permissionCacheRepository) getUserEpochKey(userID, orgID pulid.ID) string {
	return fmt.Sprintf("%s:%s:%s", userEpochPrefix, userID.String(), orgID.String())
}
