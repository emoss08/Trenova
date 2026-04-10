package invoiceadjustmentcontrolrepository

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

func New(p Params) repositories.InvoiceAdjustmentControlRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.invoiceadjustmentcontrol-repository"),
	}
}

func (r *repository) GetByOrgID(
	ctx context.Context,
	orgID pulid.ID,
) (*tenant.InvoiceAdjustmentControl, error) {
	log := r.l.With(
		zap.String("operation", "GetByOrgID"),
		zap.String("orgID", orgID.String()),
	)

	entity := new(tenant.InvoiceAdjustmentControl)
	if err := r.db.DB().NewSelect().
		Model(entity).
		Where("iac.organization_id = ?", orgID).
		Scan(ctx); err != nil {
		log.Error("failed to get invoice adjustment control", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "InvoiceAdjustmentControl")
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *tenant.InvoiceAdjustmentControl,
) (*tenant.InvoiceAdjustmentControl, error) {
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
		log.Error("failed to update invoice adjustment control", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(result, "InvoiceAdjustmentControl", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}
