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
	defaultBillingControlTTL = 24 * time.Hour
	bcKeyPrefix              = "billing_control:"
)

type BillingControlRepositoryParams struct {
	fx.In

	Cache  *redis.Connection
	Logger *zap.Logger
}

type billingControlRepository struct {
	cache *redis.Connection
	l     *zap.Logger
}

func NewBillingControlRepository(
	p BillingControlRepositoryParams,
) repositories.BillingControlCacheRepository {
	return &billingControlRepository{
		cache: p.Cache,
		l:     p.Logger.Named("redis.billingcontrol-repository"),
	}
}

func (bc *billingControlRepository) GetByOrgID(
	ctx context.Context,
	orgID pulid.ID,
) (*tenant.BillingControl, error) {
	log := bc.l.With(
		zap.String("operation", "GetByOrgID"),
		zap.String("orgID", orgID.String()),
	)

	billingControl := new(tenant.BillingControl)
	key := bc.formatKey(orgID)
	if err := bc.cache.GetJSON(ctx, key, billingControl); err != nil {
		log.Error("failed to get billing control from cache", zap.Error(err))
		return nil, err
	}

	log.Debug("retrieved billing control from cache", zap.String("key", key))
	return billingControl, nil
}

func (bc *billingControlRepository) Set(
	ctx context.Context,
	billingControl *tenant.BillingControl,
) error {
	log := bc.l.With(
		zap.String("operation", "Set"),
		zap.String("orgID", billingControl.OrganizationID.String()),
	)

	key := bc.formatKey(billingControl.OrganizationID)
	if err := bc.cache.SetJSON(ctx, key, billingControl, defaultBillingControlTTL); err != nil {
		log.Error("failed to set billing control in cache", zap.Error(err))
		return err
	}

	log.Debug("stored billing control in cache", zap.String("key", key))
	return nil
}

func (bc *billingControlRepository) Invalidate(ctx context.Context, orgID pulid.ID) error {
	log := bc.l.With(
		zap.String("operation", "Invalidate"),
		zap.String("orgID", orgID.String()),
	)

	key := bc.formatKey(orgID)
	if err := bc.cache.Delete(ctx, key); err != nil {
		log.Error("failed to invalidate billing control in cache", zap.Error(err))
		return err
	}

	log.Debug("invalidated billing control in cache", zap.String("key", key))
	return nil
}

func (bc *billingControlRepository) formatKey(orgID pulid.ID) string {
	return fmt.Sprintf("%s%s", bcKeyPrefix, orgID)
}
