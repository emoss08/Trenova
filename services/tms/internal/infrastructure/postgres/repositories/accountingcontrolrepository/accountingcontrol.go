package accountingcontrolrepository

import (
	"context"

	accountingcontrol "github.com/emoss08/trenova/internal/core/domain/tenant"
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

func New(p Params) repositories.AccountingControlRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.accountingcontrol-repository"),
	}
}

func (r *repository) GetByOrgID(
	ctx context.Context,
	orgID pulid.ID,
) (*accountingcontrol.AccountingControl, error) {
	log := r.l.With(
		zap.String("operation", "GetByOrgID"),
		zap.String("orgID", orgID.String()),
	)

	entity := new(accountingcontrol.AccountingControl)
	if err := r.db.DB().NewSelect().
		Model(entity).
		Where("ac.organization_id = ?", orgID).
		Scan(ctx); err != nil {
		log.Error("failed to get accounting control", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "AccountingControl")
	}

	return entity, nil
}

func (r *repository) ListWithScheduledPeriodClose(
	ctx context.Context,
) ([]*accountingcontrol.AccountingControl, error) {
	log := r.l.With(zap.String("operation", "ListWithScheduledPeriodClose"))

	entities := make([]*accountingcontrol.AccountingControl, 0)
	if err := r.db.DB().
		NewSelect().
		Model(&entities).
		Where("ac.period_close_mode = ?", accountingcontrol.PeriodCloseModeSystemScheduled).
		Scan(ctx); err != nil {
		log.Error("failed to list accounting controls with scheduled period close", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *accountingcontrol.AccountingControl,
) (*accountingcontrol.AccountingControl, error) {
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
		log.Error("failed to update accounting control", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(result, "AccountingControl", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}
