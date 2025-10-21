package dispatchcontrolrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
	Cache  repositories.DispatchControlCacheRepository
}

type repository struct {
	db    *postgres.Connection
	cache repositories.DispatchControlCacheRepository
	l     *zap.Logger
}

func NewRepository(p Params) repositories.DispatchControlRepository {
	return &repository{
		db:    p.DB,
		cache: p.Cache,
		l:     p.Logger.Named("postgres.dispatchcontrol-repository"),
	}
}

func (r *repository) GetByOrgID(
	ctx context.Context,
	orgID pulid.ID,
) (*tenant.DispatchControl, error) {
	log := r.l.With(
		zap.String("operation", "GetByOrgID"),
		zap.String("orgID", orgID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	cachedEntity, err := r.cache.GetByOrgID(ctx, orgID)
	if err == nil && cachedEntity.ID.IsNotNil() {
		log.Debug(
			"retrieved dispatch control from cache",
			zap.String("orgID", orgID.String()),
		)

		return cachedEntity, nil
	}

	entity := new(tenant.DispatchControl)
	err = db.NewSelect().Model(entity).Where("organization_id = ?", orgID).Scan(ctx)
	if err != nil {
		log.Error("failed to scan dispatch control", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "dispatch control")
	}

	if err = r.cache.Set(ctx, entity); err != nil {
		log.Error("failed to set dispatch control in cache", zap.Error(err))
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *tenant.DispatchControl,
) (*tenant.DispatchControl, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("orgID", entity.OrganizationID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	ov := entity.Version
	entity.Version++

	results, err := db.NewUpdate().Model(entity).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.
				Where("dc.id = ?", entity.ID).
				Where("dc.organization_id = ?", entity.OrganizationID).
				Where("dc.business_unit_id = ?", entity.BusinessUnitID).
				Where("dc.version = ?", ov)
		}).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update dispatch control", zap.Error(err))
		return nil, err
	}

	roErr := dberror.CheckRowsAffected(results, "Dispatch Control", entity.OrganizationID.String())
	if roErr != nil {
		return nil, roErr
	}

	if err = r.cache.Invalidate(ctx, entity.OrganizationID); err != nil {
		log.Error("failed to invalidate dispatch control in cache", zap.Error(err))
	}

	return entity, nil
}
