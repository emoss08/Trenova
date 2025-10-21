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
	defaultShipmentControlTTL = 24 * time.Hour
	scKeyPrefix               = "shipment_control:"
)

type ShipmentControlRepositoryParams struct {
	fx.In

	Cache  *redis.Connection
	Logger *zap.Logger
}

type shipmentControlRepository struct {
	cache *redis.Connection
	l     *zap.Logger
}

func NewShipmentControlRepository(
	p ShipmentControlRepositoryParams,
) repositories.ShipmentControlCacheRepository {
	return &shipmentControlRepository{
		cache: p.Cache,
		l:     p.Logger.Named("redis.shipmentcontrol-repository"),
	}
}

func (sc *shipmentControlRepository) GetByOrgID(
	ctx context.Context,
	orgID pulid.ID,
) (*tenant.ShipmentControl, error) {
	log := sc.l.With(
		zap.String("operation", "GetByOrgID"),
		zap.String("orgID", orgID.String()),
	)

	shipmentControl := new(tenant.ShipmentControl)
	key := sc.formatKey(orgID)
	if err := sc.cache.GetJSON(ctx, key, shipmentControl); err != nil {
		log.Error("failed to get shipment control from cache", zap.Error(err))
		return nil, err
	}

	log.Debug("retrieved shipment control from cache", zap.String("key", key))
	return shipmentControl, nil
}

func (sc *shipmentControlRepository) Set(
	ctx context.Context,
	shipmentControl *tenant.ShipmentControl,
) error {
	log := sc.l.With(
		zap.String("operation", "Set"),
		zap.String("orgID", shipmentControl.OrganizationID.String()),
	)

	key := sc.formatKey(shipmentControl.OrganizationID)
	if err := sc.cache.SetJSON(ctx, key, shipmentControl, defaultShipmentControlTTL); err != nil {
		log.Error("failed to set shipment control in cache", zap.Error(err))
		return err
	}

	log.Debug("stored shipment control in cache", zap.String("key", key))
	return nil
}

func (sc *shipmentControlRepository) Invalidate(ctx context.Context, orgID pulid.ID) error {
	log := sc.l.With(
		zap.String("operation", "Invalidate"),
		zap.String("orgID", orgID.String()),
	)

	key := sc.formatKey(orgID)
	if err := sc.cache.Delete(ctx, key); err != nil {
		log.Error("failed to invalidate shipment control in cache", zap.Error(err))
		return err
	}

	log.Debug("invalidated shipment control in cache", zap.String("key", key))
	return nil
}

func (sc *shipmentControlRepository) formatKey(orgID pulid.ID) string {
	return fmt.Sprintf("%s%s", scKeyPrefix, orgID)
}
