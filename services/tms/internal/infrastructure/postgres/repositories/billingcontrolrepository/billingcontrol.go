package billingcontrolrepository

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
	Cache  repositories.BillingControlCacheRepository
}

type repository struct {
	db    *postgres.Connection
	cache repositories.BillingControlCacheRepository
	l     *zap.Logger
}

func NewRepository(p Params) repositories.BillingControlRepository {
	return &repository{
		db:    p.DB,
		cache: p.Cache,
		l:     p.Logger.Named("postgres.billingcontrol-repository"),
	}
}

func (r *repository) GetByOrgID(
	ctx context.Context,
	orgID pulid.ID,
) (*tenant.BillingControl, error) {
	log := r.l.With(
		zap.String("operation", "GetByOrgID"),
		zap.String("orgID", orgID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	cachedBc, err := r.cache.GetByOrgID(ctx, orgID)
	if err == nil && cachedBc.ID.IsNotNil() {
		log.Debug(
			"retrieved billing control from cache",
			zap.String("orgID", orgID.String()),
		)

		return cachedBc, nil
	}

	bc := new(tenant.BillingControl)
	err = db.NewSelect().Model(bc).Where("organization_id = ?", orgID).Scan(ctx)
	if err != nil {
		log.Error("failed to scan billing control", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "billing control")
	}

	if err = r.cache.Set(ctx, bc); err != nil {
		log.Error("failed to set billing control in cache", zap.Error(err))
	}

	return bc, nil
}

func (r *repository) Update(
	ctx context.Context,
	bc *tenant.BillingControl,
) (*tenant.BillingControl, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("orgID", bc.OrganizationID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	ov := bc.Version
	bc.Version++

	results, err := db.NewUpdate().Model(bc).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update billing control", zap.Error(err))
		return nil, err
	}

	roErr := dberror.CheckRowsAffected(results, "Billing Control", bc.OrganizationID.String())
	if roErr != nil {
		return nil, roErr
	}

	if err = r.cache.Invalidate(ctx, bc.OrganizationID); err != nil {
		log.Error("failed to invalidate billing control in cache", zap.Error(err))
	}

	return bc, nil
}
