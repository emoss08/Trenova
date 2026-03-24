package dispatchcontrolrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
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

func New(p Params) repositories.DispatchControlRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.dispatch-control-repository"),
	}
}

func (r *repository) GetByOrgID(
	ctx context.Context,
	req repositories.GetDispatchControlRequest,
) (*dispatchcontrol.DispatchControl, error) {
	log := r.l.With(
		zap.String("operation", "GetByOrgID"),
		zap.String("orgId", req.TenantInfo.OrgID.String()),
	)

	entity := new(dispatchcontrol.DispatchControl)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("dc.organization_id = ?", req.TenantInfo.OrgID).
		Where("dc.business_unit_id = ?", req.TenantInfo.BuID).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get dispatch control", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "DispatchControl")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *dispatchcontrol.DispatchControl,
) (*dispatchcontrol.DispatchControl, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("orgId", entity.OrganizationID.String()),
	)

	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to create dispatch control", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *dispatchcontrol.DispatchControl,
) (*dispatchcontrol.DispatchControl, error) {
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
		log.Error("failed to update dispatch control", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "DispatchControl", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) GetOrCreate(
	ctx context.Context,
	orgID, buID pulid.ID,
) (*dispatchcontrol.DispatchControl, error) {
	log := r.l.With(
		zap.String("operation", "GetOrCreate"),
		zap.String("orgId", orgID.String()),
	)

	entity := new(dispatchcontrol.DispatchControl)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("dc.organization_id = ?", orgID).
				Where("dc.business_unit_id = ?", buID)
		}).
		Scan(ctx)

	if err == nil {
		return entity, nil
	}

	if !dberror.IsNotFoundError(err) {
		log.Error("failed to get dispatch control", zap.Error(err))
		return nil, err
	}

	newEntity := dispatchcontrol.NewDefaultDispatchControl(orgID, buID)
	err = r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, tx bun.Tx) error {
		if _, insertErr := r.db.DBForContext(c).NewInsert().
			Model(newEntity).
			Returning("*").
			Exec(c); insertErr != nil {
			log.Error("failed to create default dispatch control", zap.Error(insertErr))
			return insertErr
		}
		return nil
	})
	if err != nil {
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Dispatch control is busy. Retry the request.",
		)
	}

	return newEntity, nil
}
