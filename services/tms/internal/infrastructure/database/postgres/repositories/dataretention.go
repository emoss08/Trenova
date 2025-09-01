package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories/common"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type DataRetentionRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
	Cache  repositories.DataRetentionCacheRepository
}

type dataRetentionRepository struct {
	db    db.Connection
	l     *zerolog.Logger
	cache repositories.DataRetentionCacheRepository
}

func NewDataRetentionRepository(
	p DataRetentionRepositoryParams,
) repositories.DataRetentionRepository {
	log := p.Logger.With().
		Str("repository", "dataRetention").
		Str("component", "postgres").
		Logger()

	return &dataRetentionRepository{
		db:    p.DB,
		l:     &log,
		cache: p.Cache,
	}
}

func (dr *dataRetentionRepository) List(
	ctx context.Context,
) (*ports.ListResult[*organization.DataRetention], error) {
	dba, err := dr.db.ReadDB(ctx)
	if err != nil {
		return nil, err
	}

	log := dr.l.With().
		Str("operation", "List").
		Logger()

	cachedEntities, err := dr.cache.List(ctx)
	if err == nil && cachedEntities.Total > 0 {
		log.Debug().Int("count", cachedEntities.Total).Msg("got data retention list from cache")
		return cachedEntities, nil
	}

	entities := make([]*organization.DataRetention, 0)

	count, err := dba.NewSelect().Model(&entities).ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to list data retention entities")
		return nil, err
	}

	if err = dr.cache.SetList(ctx, entities); err != nil {
		log.Error().Err(err).Msg("failed to set data retention list in cache")
		// ! Do not return the error because it will not affect the user experience
	}

	return &ports.ListResult[*organization.DataRetention]{
		Items: entities,
		Total: count,
	}, nil
}

func (dr *dataRetentionRepository) Get(
	ctx context.Context,
	req repositories.GetDataRetentionRequest,
) (*organization.DataRetention, error) {
	dba, err := dr.db.ReadDB(ctx)
	if err != nil {
		return nil, err
	}

	log := dr.l.With().
		Str("operation", "Get").
		Interface("req", req).
		Logger()

	entity := new(organization.DataRetention)

	query := dba.NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("dr.organization_id = ?", req.OrgID).
				Where("dr.business_unit_id = ?", req.BuID)
		})

	if err = query.Scan(ctx); err != nil {
		log.Error().Err(err).Msg("failed to get data retention")
		return nil, common.HandleNotFoundError(err, "Data Retention")
	}

	return entity, nil
}

func (dr *dataRetentionRepository) Update(
	ctx context.Context,
	entity *organization.DataRetention,
) (*organization.DataRetention, error) {
	dba, err := dr.db.WriteDB(ctx)
	if err != nil {
		return nil, err
	}

	log := dr.l.With().
		Str("operation", "Update").
		Interface("entity", entity).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		return common.OptimisticUpdate(ctx, tx, entity, "Data Retention")
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update data retention")
		return nil, err
	}

	if err = dr.cache.Invalidate(ctx, repositories.GetDataRetentionRequest{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}); err != nil {
		log.Error().Err(err).Msg("failed to invalidate data retention in cache")
		// ! Do not return the error because it will not affect the user experience
	}

	return entity, nil
}
