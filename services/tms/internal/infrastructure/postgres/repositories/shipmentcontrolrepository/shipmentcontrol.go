package shipmentcontrolrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
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
	Logger *zap.Logger
	Cache  repositories.ShipmentControlCacheRepository
}

type repository struct {
	db    *postgres.Connection
	cache repositories.ShipmentControlCacheRepository
	l     *zap.Logger
}

func NewRepository(p Params) repositories.ShipmentControlRepository {
	return &repository{
		db:    p.DB,
		cache: p.Cache,
		l:     p.Logger.Named("postgres.shipmentcontrol-repository"),
	}
}

func (r *repository) GetByOrgID(
	ctx context.Context,
	orgID pulid.ID,
) (*tenant.ShipmentControl, error) {
	log := r.l.With(
		zap.String("operation", "GetByOrgID"),
		zap.String("orgID", orgID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	cachedSc, err := r.cache.GetByOrgID(ctx, orgID)
	if err == nil && cachedSc.ID.IsNotNil() {
		log.Debug(
			"retrieved shipment control from cache",
			zap.String("orgID", orgID.String()),
		)

		return cachedSc, nil
	}

	sc := new(tenant.ShipmentControl)
	err = db.NewSelect().Model(sc).Where("organization_id = ?", orgID).Scan(ctx)
	if err != nil {
		log.Error("failed to scan shipment control", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "shipment control")
	}

	if err = r.cache.Set(ctx, sc); err != nil {
		log.Error("failed to set shipment control in cache", zap.Error(err))
	}

	return sc, nil
}

func (r *repository) Update(
	ctx context.Context,
	sc *tenant.ShipmentControl,
) (*tenant.ShipmentControl, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("orgID", sc.OrganizationID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	ov := sc.Version
	sc.Version++

	results, err := db.NewUpdate().Model(sc).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update shipment control", zap.Error(err))
		return nil, err
	}

	roErr := dberror.CheckRowsAffected(results, "Shipment Control", sc.OrganizationID.String())
	if roErr != nil {
		return nil, roErr
	}

	if err = r.cache.Invalidate(ctx, sc.OrganizationID); err != nil {
		log.Error("failed to invalidate shipment control in cache", zap.Error(err))
	}

	return sc, nil
}
