package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/hazmatexpiration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/redis"
	"github.com/emoss08/trenova/pkg/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultHazmatExpirationTTL = 24 * time.Hour
	hazmatExpirationKeyPrefix  = "hazmat_expiration:"
)

type HazmatExpirationRepositoryParams struct {
	fx.In

	Cache  *redis.Connection
	Logger *zap.Logger
}

type hazmatExpirationRepository struct {
	cache *redis.Connection
	l     *zap.Logger
}

func NewHazmatExpirationRepository(
	p HazmatExpirationRepositoryParams,
) repositories.HazmatExpirationCacheRepository {
	return &hazmatExpirationRepository{
		cache: p.Cache,
		l:     p.Logger.Named("redis.hazmatexpiration-repository"),
	}
}

func (h *hazmatExpirationRepository) GetHazmatExpirationByStateID(
	ctx context.Context,
	stateID pulid.ID,
) (*hazmatexpiration.HazmatExpiration, error) {
	log := h.l.With(
		zap.String("operation", "GetHazmatExpirationByStateID"),
		zap.String("stateID", stateID.String()),
	)

	hazmatExpiration := new(hazmatexpiration.HazmatExpiration)
	key := h.formatKey(stateID)
	if err := h.cache.GetJSON(ctx, key, hazmatExpiration); err != nil {
		log.Error("failed to get hazmat expiration from cache", zap.Error(err))
		return nil, err
	}

	log.Debug("retrieved hazmat expiration from cache", zap.String("key", key))
	return hazmatExpiration, nil
}

func (h *hazmatExpirationRepository) Set(
	ctx context.Context,
	expiration *hazmatexpiration.HazmatExpiration,
) error {
	log := h.l.With(
		zap.String("operation", "Set"),
		zap.String("stateID", expiration.StateID.String()),
	)

	key := h.formatKey(expiration.StateID)
	if err := h.cache.SetJSON(ctx, key, expiration, defaultHazmatExpirationTTL); err != nil {
		log.Error("failed to set hazmat expiration in cache", zap.Error(err))
		return err
	}

	log.Debug("stored hazmat expiration in cache", zap.String("key", key))
	return nil
}

func (h *hazmatExpirationRepository) Invalidate(ctx context.Context, stateID pulid.ID) error {
	log := h.l.With(
		zap.String("operation", "Invalidate"),
		zap.String("stateID", stateID.String()),
	)
	key := h.formatKey(stateID)
	if err := h.cache.Delete(ctx, key); err != nil {
		log.Error("failed to invalidate hazmat expiration in cache", zap.Error(err))
		return err
	}

	log.Debug("invalidated hazmat expiration in cache", zap.String("key", key))
	return nil
}

func (h *hazmatExpirationRepository) formatKey(stateID pulid.ID) string {
	return fmt.Sprintf("%s%s", hazmatExpirationKeyPrefix, stateID.String())
}
