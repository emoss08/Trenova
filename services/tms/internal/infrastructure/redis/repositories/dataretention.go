package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/redis"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultDataRetentionTTL = 24 * time.Hour
	drKeyPrefix             = "data_retention:"
	drAllKey                = "dr:all"
)

type DataRetentionRepositoryParams struct {
	fx.In

	Cache  *redis.Connection
	Logger *zap.Logger
}

type dataRetentionRepository struct {
	cache *redis.Connection
	l     *zap.Logger
}

func NewDataRetentionRepository(
	p DataRetentionRepositoryParams,
) repositories.DataRetentionCacheRepository {
	return &dataRetentionRepository{
		cache: p.Cache,
		l:     p.Logger.Named("redis.dataretention-repository"),
	}
}

func (dr *dataRetentionRepository) List(
	ctx context.Context,
) (*pagination.ListResult[*tenant.DataRetention], error) {
	log := dr.l.With(zap.String("operation", "List"))

	entities := make([]*tenant.DataRetention, 0)

	if err := dr.cache.GetJSON(ctx, drAllKey, &entities); err != nil {
		log.Error("failed to get data retentions from cache", zap.Error(err))
		return nil, err
	}

	log.Debug("retrieved data retentions from cache", zap.Int("count", len(entities)))
	return &pagination.ListResult[*tenant.DataRetention]{
		Items: entities,
		Total: len(entities),
	}, nil
}

func (dr *dataRetentionRepository) InvalidateAll(ctx context.Context) error {
	log := dr.l.With(zap.String("operation", "InvalidateAll"))

	if err := dr.cache.Delete(ctx, drAllKey); err != nil {
		log.Error("failed to invalidate data retentions in cache", zap.Error(err))
		return err
	}

	log.Debug("invalidated data retentions in cache")
	return nil
}

func (dr *dataRetentionRepository) Get(
	ctx context.Context,
	req repositories.GetDataRetentionRequest,
) (*tenant.DataRetention, error) {
	log := dr.l.With(zap.String("operation", "Get"))

	entity := new(tenant.DataRetention)
	key := dr.formatKey(req)
	if err := dr.cache.GetJSON(ctx, key, entity); err != nil {
		log.Error("failed to get data retention from cache", zap.Error(err))
		return nil, err
	}

	log.Debug("retrieved data retention from cache", zap.String("key", key))
	return entity, nil
}

func (dr *dataRetentionRepository) Set(
	ctx context.Context,
	entity *tenant.DataRetention,
) error {
	log := dr.l.With(zap.String("operation", "Set"))

	key := dr.formatKey(repositories.GetDataRetentionRequest{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})
	if err := dr.cache.SetJSON(ctx, key, entity, defaultDataRetentionTTL); err != nil {
		log.Error("failed to set data retention in cache", zap.Error(err))
		return err
	}

	log.Debug("stored data retention in cache", zap.String("key", key))
	return nil
}

func (dr *dataRetentionRepository) SetList(
	ctx context.Context,
	entities []*tenant.DataRetention,
) error {
	log := dr.l.With(zap.String("operation", "SetList"))

	if err := dr.cache.SetJSON(ctx, drAllKey, entities, defaultDataRetentionTTL); err != nil {
		log.Error("failed to set data retentions in cache", zap.Error(err))
		return err
	}

	log.Debug("stored data retentions in cache", zap.Int("count", len(entities)))
	return nil
}

func (dr *dataRetentionRepository) Invalidate(
	ctx context.Context,
	req repositories.GetDataRetentionRequest,
) error {
	log := dr.l.With(zap.String("operation", "Invalidate"))

	key := dr.formatKey(req)
	if err := dr.cache.Delete(ctx, key); err != nil {
		log.Error("failed to invalidate data retention in cache", zap.Error(err))
		return err
	}

	log.Debug("invalidated data retention in cache", zap.String("key", key))
	return nil
}

func (dr *dataRetentionRepository) formatKey(req repositories.GetDataRetentionRequest) string {
	return fmt.Sprintf(
		"%s:%s:%s",
		drKeyPrefix,
		req.OrgID.String(),
		req.BuID.String(),
	)
}
