package dataentrycontrolrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/dataentrycontrol"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
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

func New(p Params) repositories.DataEntryControlRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.data-entry-control-repository"),
	}
}

func (r *repository) GetByOrgID(
	ctx context.Context,
	req repositories.GetDataEntryControlRequest,
) (*dataentrycontrol.DataEntryControl, error) {
	log := r.l.With(
		zap.String("operation", "GetByOrgID"),
		zap.String("orgId", req.TenantInfo.OrgID.String()),
	)

	entity := new(dataentrycontrol.DataEntryControl)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("dec.organization_id = ?", req.TenantInfo.OrgID).
				Where("dec.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get data entry control", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "DataEntryControl")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *dataentrycontrol.DataEntryControl,
) (*dataentrycontrol.DataEntryControl, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("orgId", entity.OrganizationID.String()),
	)

	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to create data entry control", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *dataentrycontrol.DataEntryControl,
) (*dataentrycontrol.DataEntryControl, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		OmitZero().
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update data entry control", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "DataEntryControl", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) GetOrCreate(
	ctx context.Context,
	orgID, buID pulid.ID,
) (*dataentrycontrol.DataEntryControl, error) {
	log := r.l.With(
		zap.String("operation", "GetOrCreate"),
		zap.String("orgId", orgID.String()),
	)

	entity := new(dataentrycontrol.DataEntryControl)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("dec.organization_id = ?", orgID).
				Where("dec.business_unit_id = ?", buID)
		}).
		Scan(ctx)

	if err == nil {
		return entity, nil
	}

	if !dberror.IsNotFoundError(err) {
		log.Error("failed to get data entry control", zap.Error(err))
		return nil, err
	}

	newEntity := dataentrycontrol.NewDefaultDataEntryControl(orgID, buID)
	err = r.db.WithTx(ctx, ports.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		if _, insertErr := r.db.DBForContext(ctx).NewInsert().
			Model(newEntity).
			Returning("*").
			Exec(ctx); insertErr != nil {
			log.Error("failed to create default data entry control", zap.Error(insertErr))
			return insertErr
		}
		return nil
	})
	if err != nil {
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Data entry control is busy. Retry the request.",
		)
	}

	return newEntity, nil
}
