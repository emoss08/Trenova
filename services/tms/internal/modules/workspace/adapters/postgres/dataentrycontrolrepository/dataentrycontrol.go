package dataentrycontrolrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/dataentrycontrol"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/dataentrycontrol/dataentrycontroldb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/dataentrycontrolports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var dataEntryControlErrors = dberror.Mapping{Entity: "DataEntryControl"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) dataentrycontrolports.DataEntryControlRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.data-entry-control-repository"),
	}
}

func (r *repository) GetByOrgID(
	ctx context.Context,
	req dataentrycontrolports.GetDataEntryControlRequest,
) (*dataentrycontrol.DataEntryControl, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*dataentrycontrol.DataEntryControl, error) {
			entity, err := r.find(ctx)
			if err != nil {
				r.l.Error(
					"failed to get data entry control",
					zap.String("orgId", req.TenantInfo.OrgID.String()),
					zap.Error(err),
				)
				return nil, dataEntryControlErrors.Map(err, req.TenantInfo.OrgID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *dataentrycontrol.DataEntryControl,
) (*dataentrycontrol.DataEntryControl, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*dataentrycontrol.DataEntryControl, error) {
			err := dataentrycontroldb.DataEntryControls.Create(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error(
					"failed to create data entry control",
					zap.String("orgId", entity.OrganizationID.String()),
					zap.Error(err),
				)
				return nil, err
			}

			return entity, nil
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	entity *dataentrycontrol.DataEntryControl,
) (*dataentrycontrol.DataEntryControl, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*dataentrycontrol.DataEntryControl, error) {
			err := dataentrycontroldb.DataEntryControls.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error(
					"failed to update data entry control",
					zap.String("id", entity.ID.String()),
					zap.Error(err),
				)
				return nil, dataEntryControlErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
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
	return dbtx.GetOrCreate(ctx, r.db, dbtx.GetOrCreateSpec[*dataentrycontrol.DataEntryControl]{
		Find: r.find,
		Create: func(ctx context.Context) error {
			d := dataentrycontroldb.DataEntryControls
			_, insertErr := d.CreateOrSkip(
				ctx,
				r.db.Keel(ctx),
				[]*dataentrycontrol.DataEntryControl{
					dataentrycontrol.NewDefaultDataEntryControl(orgID, buID),
				},
				d.OrganizationID,
				d.BusinessUnitID,
			)
			if insertErr != nil {
				r.l.Error("failed to create default data entry control", zap.Error(insertErr))
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
			r.l.Error(
				"failed to get data entry control",
				zap.String("orgId", orgID.String()),
				zap.Error(err),
			)
			return dataEntryControlErrors.Map(err, orgID.String())
		},
	})
}

func (r *repository) find(ctx context.Context) (*dataentrycontrol.DataEntryControl, error) {
	return dataentrycontroldb.DataEntryControls.Query().One(ctx, r.db.Keel(ctx))
}
