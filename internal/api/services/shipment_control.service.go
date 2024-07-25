package services

import (
	"context"
	"fmt"
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/config"
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/redis"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type ShipmentControlService struct {
	db     *bun.DB
	logger *config.ServerLogger
	cache  *redis.Client
}

func NewShipmentControlService(s *server.Server) *ShipmentControlService {
	return &ShipmentControlService{
		db:     s.DB,
		logger: s.Logger,
		cache:  s.Cache,
	}
}

func (s ShipmentControlService) shipmentControlCacheKey(orgID uuid.UUID) string {
	return fmt.Sprintf("shipment_control:%s", orgID)
}

func (s ShipmentControlService) GetShipmentControl(ctx context.Context, buID, orgID uuid.UUID) (*models.ShipmentControl, error) {
	cacheKey := s.shipmentControlCacheKey(orgID)

	// Try to fetch the shipment control from the cache.
	cachedControl, err := s.cache.FetchFromCacheByKey(ctx, cacheKey)
	if err != nil {
		s.logger.Error().Str("orgID", orgID.String()).Err(err).Msg("Failed to fetch organization from cache")
		// Do not return an error if the organization is not in the cache.
		// We want to fetch it from the database in that case.
		// Once fetched from the database, we will cache it.
	}

	if cachedControl != "" {
		control := new(models.ShipmentControl)

		if err = sonic.Unmarshal([]byte(cachedControl), control); err != nil {
			s.logger.Error().Str("cacheKey", cacheKey).Err(err).Msg("failed to unmarshal shipment control from cache")
			return nil, err
		}

		return control, nil
	}

	// if not in cache then fetch from the database
	control := new(models.ShipmentControl)
	err = s.db.NewSelect().
		Model(control).
		Where("sc.organization_id = ?", orgID).
		Where("sc.business_unit_id = ?", buID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	controlJSON, err := sonic.Marshal(control)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to marshal shipment control")
		return nil, err
	}

	if err = s.cache.CacheByKey(ctx, cacheKey, string(controlJSON)); err != nil {
		s.logger.Error().Str("cacheKey", cacheKey).Err(err).Msg("Failed to cache shipment control")
	}

	return control, nil
}

func (s ShipmentControlService) UpdateShipmentControl(ctx context.Context, entity *models.ShipmentControl) (*models.ShipmentControl, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := entity.OptimisticUpdate(ctx, tx); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Invalidate cache
	cacheKey := s.shipmentControlCacheKey(entity.OrganizationID)
	if err = s.cache.InvalidateCacheByKey(ctx, cacheKey); err != nil {
		s.logger.Error().Err(err).Str("cacheKey", cacheKey).Msg("Failed to invalidate cache")
		return nil, err
	}

	return entity, nil
}
