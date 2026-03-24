package billingcontrolrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.BillingControlRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.billingcontrol-repository"),
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

	entity := new(tenant.BillingControl)
	if err := r.db.DB().NewSelect().
		Model(entity).
		Where("bc.organization_id = ?", orgID).
		Scan(ctx); err != nil {
		log.Error("failed to get billing control", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "BillingControl")
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *tenant.BillingControl,
) (*tenant.BillingControl, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("orgID", entity.OrganizationID.String()),
	)

	ov := entity.Version
	entity.Version++

	result, err := r.db.DB().
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update billing control", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(result, "BillingControl", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}
