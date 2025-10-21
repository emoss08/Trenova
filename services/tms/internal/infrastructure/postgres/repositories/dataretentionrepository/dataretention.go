package dataretentionrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
	Cache  repositories.DataRetentionCacheRepository
}

type repository struct {
	db    *postgres.Connection
	cache repositories.DataRetentionCacheRepository
	l     *zap.Logger
}

func NewRepository(p Params) repositories.DataRetentionRepository {
	return &repository{
		db:    p.DB,
		cache: p.Cache,
		l:     p.Logger.Named("postgres.dataretention-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
) (*pagination.ListResult[*tenant.DataRetention], error) {
	log := r.l.With(zap.String("operation", "List"))

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	cachedEntities, err := r.cache.List(ctx)
	if err == nil && cachedEntities.Total > 0 {
		log.Debug("got data retentions from cache", zap.Int("count", cachedEntities.Total))
		return cachedEntities, nil
	}

	entities := make([]*tenant.DataRetention, 0)

	total, err := db.NewSelect().Model(&entities).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan data retentions", zap.Error(err))
		return nil, err
	}

	if err = r.cache.SetList(ctx, entities); err != nil {
		log.Error("failed to set data retentions in cache", zap.Error(err))
		// ! Do not return the error because it will not affect the user experience
	}

	return &pagination.ListResult[*tenant.DataRetention]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) Get(
	ctx context.Context,
	req repositories.GetDataRetentionRequest,
) (*tenant.DataRetention, error) {
	log := r.l.With(zap.String("operation", "Get"), zap.Any("request", req))

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(tenant.DataRetention)
	err = db.NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("dr.organization_id = ?", req.OrgID).
				Where("dr.business_unit_id = ?", req.BuID)
		}).Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Data Retention")
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *tenant.DataRetention,
) (*tenant.DataRetention, error) {
	log := r.l.With(zap.String("operation", "Update"), zap.String("entityId", entity.ID.String()))

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	ov := entity.Version
	entity.Version++

	result, err := db.NewUpdate().
		Model(entity).WherePK().Where("version = ?", ov).Returning("*").Exec(ctx)
	if err != nil {
		log.Error("failed to update data retention", zap.Error(err))
		return nil, err
	}

	roErr := dberror.CheckRowsAffected(result, "Data Retention", entity.ID.String())
	if roErr != nil {
		return nil, roErr
	}

	if err = r.cache.Invalidate(ctx, repositories.GetDataRetentionRequest{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}); err != nil {
		log.Error("failed to invalidate data retention in cache", zap.Error(err))
		// ! Do not return the error because it will not affect the user experience
	}

	return entity, nil
}
