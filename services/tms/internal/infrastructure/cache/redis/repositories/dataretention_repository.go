package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/cache/redis"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

const (
	drAllKey     = "dr:all"
	drKeyPrefix  = "dr:"
	defaultDrTTL = 24 * time.Hour
)

type DataRetentionRepositoryParams struct {
	fx.In

	Cache  *redis.Client
	Logger *logger.Logger
}

type dataRetentionRepository struct {
	cache *redis.Client
	l     *zerolog.Logger
}

func NewDataRetentionRepository(
	p DataRetentionRepositoryParams,
) repositories.DataRetentionCacheRepository {
	log := p.Logger.With().
		Str("repository", "dataRetention").
		Str("component", "redis").
		Logger()

	return &dataRetentionRepository{
		cache: p.Cache,
		l:     &log,
	}
}

// List retrieves all data retention entities from the cache
//
// Parameters:
//   - ctx: The context of the request
//
// Returns:
//   - []*organization.DataRetention: A list of data retention entities
//   - error: An error if the data retention list is not retrieved from the cache
//
// Note: This returns all data retention entities regardless of the tenant information
func (dr *dataRetentionRepository) List(
	ctx context.Context,
) (*ports.ListResult[*organization.DataRetention], error) {
	log := dr.l.With().
		Str("operation", "List").
		Str("component", "redis").
		Logger()

	entities := make([]*organization.DataRetention, 0)

	if err := dr.cache.GetJSON(ctx, ".", drAllKey, &entities); err != nil {
		return nil, err
	}

	log.Debug().Str("key", drAllKey).Msg("retrieved data retention list from cache")
	return &ports.ListResult[*organization.DataRetention]{
		Items: entities,
		Total: len(entities),
	}, nil
}

// SetList stores a list of data retention entities in the cache
//
// Parameters:
//   - ctx: The context of the request
//   - drs: The list of data retention entities
//
// Returns:
//   - error: An error if the data retention list is not stored in the cache
func (dr *dataRetentionRepository) SetList(
	ctx context.Context,
	entities []*organization.DataRetention,
) error {
	log := dr.l.With().
		Str("operation", "SetList").
		Str("component", "redis").
		Logger()

	if err := dr.cache.SetJSON(ctx, ".", drAllKey, entities, defaultDrTTL); err != nil {
		return err
	}

	log.Debug().Str("key", drAllKey).Msg("stored data retention list in cache")
	return nil
}

func (dr *dataRetentionRepository) InvalidateAll(
	ctx context.Context,
) error {
	log := dr.l.With().
		Str("operation", "InvalidateAll").
		Str("component", "redis").
		Logger()

	if err := dr.cache.Del(ctx, drAllKey); err != nil {
		return err
	}

	log.Debug().Str("key", drAllKey).Msg("invalidated data retention list in cache")
	return nil
}

// GetByID retrieves a data retention entity from the cache by its ID
//
// Parameters:
//   - ctx: The context of the request
//   - drID: The ID of the data retention entity
//
// Returns:
//   - *organization.DataRetention: The data retention entity
//   - error: An error if the data retention entity is not retrieved from the cache
func (dr *dataRetentionRepository) GetByID(
	ctx context.Context,
	entityID pulid.ID,
) (*organization.DataRetention, error) {
	log := dr.l.With().
		Str("operation", "GetByID").
		Str("entityID", entityID.String()).
		Logger()

	entity := new(organization.DataRetention)

	key := fmt.Sprintf("%s:%s", drKeyPrefix, entityID)

	if err := dr.cache.GetJSON(ctx, ".", key, entity); err != nil {
		return nil, err
	}

	log.Debug().Str("key", key).Msg("retrieved data retention entity from cache")
	return entity, nil
}

// Set stores a data retention entity in the cache
//
// Parameters:
//   - ctx: The context of the request
//   - dr: The data retention entity
//
// Returns:
//   - error: An error if the data retention entity is not stored in the cache
func (dr *dataRetentionRepository) Set(
	ctx context.Context,
	entity *organization.DataRetention,
) error {
	log := dr.l.With().
		Str("operation", "Set").
		Str("drID", entity.ID.String()).
		Logger()

	key := fmt.Sprintf("%s:%s", drKeyPrefix, entity.ID)

	if err := dr.cache.SetJSON(ctx, ".", key, entity, defaultDrTTL); err != nil {
		return err
	}

	log.Debug().Str("key", key).Msg("stored data retention entity in cache")
	return nil
}

func (dr *dataRetentionRepository) Invalidate(
	ctx context.Context,
	entityID pulid.ID,
) error {
	log := dr.l.With().
		Str("operation", "Invalidate").
		Str("entityID", entityID.String()).
		Logger()

	key := fmt.Sprintf("%s:%s", drKeyPrefix, entityID)

	if err := dr.cache.Del(ctx, key); err != nil {
		return err
	}

	log.Debug().Str("key", key).Msg("invalidated data retention entity in cache")
	return nil
}
