package dataentrycontrolrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/dataentrycontrol"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
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
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*dataentrycontrol.DataEntryControl, error) {
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
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *dataentrycontrol.DataEntryControl,
) (*dataentrycontrol.DataEntryControl, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*dataentrycontrol.DataEntryControl, error) {
		log := r.l.With(
			zap.String("operation", "Create"),
			zap.String("orgId", entity.OrganizationID.String()),
		)

		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx); err != nil {
			log.Error("failed to create data entry control", zap.Error(err))
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *dataentrycontrol.DataEntryControl,
) (*dataentrycontrol.DataEntryControl, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*dataentrycontrol.DataEntryControl, error) {
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
			Returning("*").
			Exec(ctx)
		if err != nil {
			log.Error("failed to update data entry control", zap.Error(err))
			return nil, err
		}

		if err = dberror.CheckRowsAffected(
			results,
			"DataEntryControl",
			entity.ID.String(),
		); err != nil {
			return nil, err
		}

		return entity, nil
	})
}

// GetOrCreate reads the organization's data entry control, creating the
// default one when it has none. A preview reads inside a read-only snapshot,
// where the insert fails and aborts the whole transaction: every check after it
// then failed ("current transaction is aborted"), so no customer could be
// proposed. The read comes first, the insert runs in a savepoint, and a
// read-only caller gets the defaults without anything being written.
func (r *repository) GetOrCreate(
	ctx context.Context,
	orgID, buID pulid.ID,
) (*dataentrycontrol.DataEntryControl, error) {
	log := r.l.With(
		zap.String("operation", "GetOrCreate"),
		zap.String("orgId", orgID.String()),
	)

	return dbtx.GetOrCreate(ctx, r.db, dbtx.GetOrCreateSpec[*dataentrycontrol.DataEntryControl]{
		Find: func(ctx context.Context) (*dataentrycontrol.DataEntryControl, error) {
			return r.find(ctx, orgID, buID)
		},
		Create: func(ctx context.Context) error {
			_, insertErr := r.db.DBForContext(ctx).
				NewInsert().
				Model(dataentrycontrol.NewDefaultDataEntryControl(orgID, buID)).
				On(`CONFLICT ("organization_id", "business_unit_id") DO NOTHING`).
				Exec(ctx)
			if insertErr != nil {
				log.Error("failed to create default data entry control", zap.Error(insertErr))
				return dberror.MapRetryableTransactionError(
					insertErr,
					"Data entry control is busy. Retry the request.",
				)
			}

			return nil
		},
		Default: func() *dataentrycontrol.DataEntryControl {
			return dataentrycontrol.NewDefaultDataEntryControl(orgID, buID)
		},
		MapError: func(err error) error {
			log.Error("failed to get data entry control", zap.Error(err))
			return dberror.HandleNotFoundError(err, "DataEntryControl")
		},
	})
}

func (r *repository) find(
	ctx context.Context,
	orgID, buID pulid.ID,
) (*dataentrycontrol.DataEntryControl, error) {
	entity := new(dataentrycontrol.DataEntryControl)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("dec.organization_id = ?", orgID).
				Where("dec.business_unit_id = ?", buID)
		}).
		Scan(ctx); err != nil {
		return nil, err
	}

	return entity, nil
}
