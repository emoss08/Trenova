package accountingcontrolrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accounting"
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
	Cache  repositories.AccountingControlCacheRepository
}

type repository struct {
	db    *postgres.Connection
	cache repositories.AccountingControlCacheRepository
	l     *zap.Logger
}

func NewRepository(p Params) repositories.AccountingControlRepository {
	return &repository{
		db:    p.DB,
		cache: p.Cache,
		l:     p.Logger.Named("postgres.accountingcontrol-repository"),
	}
}

func (r *repository) GetByOrgID(
	ctx context.Context,
	orgID pulid.ID,
) (*accounting.AccountingControl, error) {
	log := r.l.With(
		zap.String("operation", "GetByOrgID"),
		zap.String("orgID", orgID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	cachedAc, err := r.cache.GetByOrgID(ctx, orgID)
	if err == nil && cachedAc.ID.IsNotNil() {
		log.Debug(
			"retrieved accounting control from cache",
			zap.String("orgID", orgID.String()),
		)

		return cachedAc, nil
	}

	ac := new(accounting.AccountingControl)
	err = db.NewSelect().Model(ac).Where("organization_id = ?", orgID).Scan(ctx)
	if err != nil {
		log.Error("failed to scan accounting control", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "accounting control")
	}

	if err = r.cache.Set(ctx, ac); err != nil {
		log.Error("failed to set accounting control in cache", zap.Error(err))
	}

	return ac, nil
}

func (r *repository) Update(
	ctx context.Context,
	ac *accounting.AccountingControl,
) (*accounting.AccountingControl, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("orgID", ac.OrganizationID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	ov := ac.Version
	ac.Version++

	results, err := db.NewUpdate().Model(ac).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update accounting control", zap.Error(err))
		return nil, err
	}

	roErr := dberror.CheckRowsAffected(results, "Accounting Control", ac.OrganizationID.String())
	if roErr != nil {
		return nil, roErr
	}

	if err = r.cache.Invalidate(ctx, ac.OrganizationID); err != nil {
		log.Error("failed to invalidate accounting control in cache", zap.Error(err))
	}

	return ac, nil
}
