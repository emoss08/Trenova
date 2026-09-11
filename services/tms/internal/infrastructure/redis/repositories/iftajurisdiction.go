package repositories

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/redishelpers"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultJurisdictionTTL = 24 * time.Hour
	jurisdictionKeyPrefix  = "ifta:jurisdictions"

	defaultJurisdictionCacheSize = 64
)

type IFTAJurisdictionCacheRepositoryParams struct {
	fx.In

	Client *redis.Client
	Logger *zap.Logger
}

type iftaJurisdictionCacheRepository struct {
	client *redis.Client
	l      *zap.Logger
}

func NewIFTAJurisdictionCacheRepository(
	p IFTAJurisdictionCacheRepositoryParams,
) repositories.IFTAJurisdictionCacheRepository {
	return &iftaJurisdictionCacheRepository{
		client: p.Client,
		l:      p.Logger.Named("redis.ifta-jurisdiction-cache-repository"),
	}
}

func (r *iftaJurisdictionCacheRepository) GetAll(
	ctx context.Context,
) ([]*ifta.Jurisdiction, error) {
	log := r.l.With(zap.String("operation", "GetAll"))

	jurisdictions := make([]*ifta.Jurisdiction, 0, defaultJurisdictionCacheSize)
	if err := redishelpers.GetJSON(ctx, r.client, jurisdictionKeyPrefix, &jurisdictions); err != nil {
		if redishelpers.IsRedisNil(err) {
			return nil, errortypes.NewNotFoundError("ifta jurisdictions not cached")
		}

		log.Error("failed to get ifta jurisdictions from cache", zap.Error(err))
		return nil, err
	}

	return jurisdictions, nil
}

func (r *iftaJurisdictionCacheRepository) Set(
	ctx context.Context,
	jurisdictions []*ifta.Jurisdiction,
) error {
	log := r.l.With(zap.String("operation", "Set"))

	if err := redishelpers.SetJSON(
		ctx,
		r.client,
		jurisdictionKeyPrefix,
		jurisdictions,
		defaultJurisdictionTTL,
	); err != nil {
		log.Error("failed to set ifta jurisdictions in cache", zap.Error(err))
		return err
	}

	log.Debug(
		"stored ifta jurisdictions in cache",
		zap.Int("jurisdictionCount", len(jurisdictions)),
	)
	return nil
}

func (r *iftaJurisdictionCacheRepository) Invalidate(ctx context.Context) error {
	log := r.l.With(zap.String("operation", "Invalidate"))

	if err := r.client.Del(ctx, jurisdictionKeyPrefix).Err(); err != nil {
		log.Error("failed to invalidate ifta jurisdictions in cache", zap.Error(err))
		return err
	}

	log.Debug("invalidated ifta jurisdictions in cache")
	return nil
}
