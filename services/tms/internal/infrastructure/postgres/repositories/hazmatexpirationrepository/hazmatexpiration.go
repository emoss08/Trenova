package hazmatexpirationrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/hazmatexpiration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Cache  repositories.HazmatExpirationCacheRepository
	Logger *zap.Logger
}

type repository struct {
	db    *postgres.Connection
	cache repositories.HazmatExpirationCacheRepository
	l     *zap.Logger
}

func NewRepository(p Params) repositories.HazmatExpirationRepository {
	return &repository{
		db:    p.DB,
		cache: p.Cache,
		l:     p.Logger.Named("postgres.hazmatexpiration-repository"),
	}
}

func (r *repository) GetHazmatExpirationByStateID(
	ctx context.Context,
	stateID pulid.ID,
) (*hazmatexpiration.HazmatExpiration, error) {
	log := r.l.With(
		zap.String("operation", "GetHazmatExpirationByStateID"),
		zap.String("stateID", stateID.String()),
	)

	cachedExpiration, err := r.cache.GetHazmatExpirationByStateID(ctx, stateID)
	if err == nil && cachedExpiration.ID.IsNotNil() {
		log.Debug("retrieved hazmat expiration from cache", zap.String("stateID", stateID.String()))
		return cachedExpiration, nil
	}

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	expiration := new(hazmatexpiration.HazmatExpiration)
	err = db.NewSelect().Model(expiration).
		Where("he.state_id = ?", stateID).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Hazmat Expiration")
	}

	if err = r.cache.Set(ctx, expiration); err != nil {
		log.Error("failed to set hazmat expiration in cache", zap.Error(err))
	}

	return expiration, nil
}
