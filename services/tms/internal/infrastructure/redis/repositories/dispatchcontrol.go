package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/redis"
	"github.com/emoss08/trenova/pkg/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultDispatchControlTTL = 24 * time.Hour
	dcKeyPrefix               = "dispatch_control:"
)

type DispatchControlRepositoryParams struct {
	fx.In

	Cache  *redis.Connection
	Logger *zap.Logger
}

type dispatchControlRepository struct {
	cache *redis.Connection
	l     *zap.Logger
}

func NewDispatchControlRepository(
	p DispatchControlRepositoryParams,
) repositories.DispatchControlCacheRepository {
	return &dispatchControlRepository{
		cache: p.Cache,
		l:     p.Logger.Named("redis.dispatchcontrol-repository"),
	}
}

func (dc *dispatchControlRepository) GetByOrgID(
	ctx context.Context,
	orgID pulid.ID,
) (*tenant.DispatchControl, error) {
	log := dc.l.With(
		zap.String("operation", "GetByOrgID"),
		zap.String("orgID", orgID.String()),
	)

	dispatchControl := new(tenant.DispatchControl)
	key := dc.formatKey(orgID)
	if err := dc.cache.GetJSON(ctx, key, dispatchControl); err != nil {
		log.Error("failed to get dispatch control from cache", zap.Error(err))
		return nil, err
	}

	log.Debug("retrieved dispatch control from cache", zap.String("key", key))
	return dispatchControl, nil
}

func (dc *dispatchControlRepository) Set(
	ctx context.Context,
	dispatchControl *tenant.DispatchControl,
) error {
	log := dc.l.With(
		zap.String("operation", "Set"),
		zap.String("orgID", dispatchControl.OrganizationID.String()),
	)

	key := dc.formatKey(dispatchControl.OrganizationID)
	if err := dc.cache.SetJSON(ctx, key, dispatchControl, defaultDispatchControlTTL); err != nil {
		log.Error("failed to set dispatch control in cache", zap.Error(err))
		return err
	}

	log.Debug("stored dispatch control in cache", zap.String("key", key))
	return nil
}

func (dc *dispatchControlRepository) Invalidate(ctx context.Context, orgID pulid.ID) error {
	log := dc.l.With(
		zap.String("operation", "Invalidate"),
		zap.String("orgID", orgID.String()),
	)

	key := dc.formatKey(orgID)
	if err := dc.cache.Delete(ctx, key); err != nil {
		log.Error("failed to invalidate dispatch control in cache", zap.Error(err))
		return err
	}

	log.Debug("invalidated dispatch control in cache", zap.String("key", key))
	return nil
}

func (dc *dispatchControlRepository) formatKey(orgID pulid.ID) string {
	return fmt.Sprintf("%s%s", dcKeyPrefix, orgID)
}
