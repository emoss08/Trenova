package manualjournalrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/manualjournal"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/manualjournal/manualjournaldb"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/manualjournalports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var manualJournalErrors = dberror.Mapping{Entity: "ManualJournalRequest"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) manualjournalports.ManualJournalRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.manual-journal-repository")}
}

func (r *repository) List(
	ctx context.Context,
	req *manualjournalports.ListManualJournalRequest,
) (*pagination.ListResult[*manualjournal.Request], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*manualjournal.Request], error) {
			mjr := manualjournaldb.Requests
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				mjr.Query(),
				&listquery.Request[manualjournal.Request]{
					Filter: req.Filter,
					Then:   []keel.Order{mjr.CreatedAt.Desc()},
				},
			)
			if err != nil {
				return nil, fmt.Errorf("list manual journals: %w", err)
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *manualjournalports.ListManualJournalConnectionRequest,
) (*pagination.CursorListResult[*manualjournal.Request], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*manualjournal.Request], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				manualjournaldb.Requests.Query(),
				&listquery.Request[manualjournal.Request]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.ManualJournalColumns,
				},
			)
			if err != nil {
				r.l.Error("failed to list manual journals", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req manualjournalports.GetManualJournalByIDRequest,
) (*manualjournal.Request, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*manualjournal.Request, error) {
		return r.get(ctx, req.ID)
	})
}

func (r *repository) get(ctx context.Context, id pulid.ID) (*manualjournal.Request, error) {
	mjr := manualjournaldb.Requests
	lines := mjr.Lines()
	account := lines.GLAccount()
	entity, err := mjr.Where(mjr.ID.Eq(id)).
		Load(lines.Order(lines.LineNumber.Asc()).
			Load(account.Pick(account.ID, account.AccountCode, account.Name))).
		One(ctx, r.db.Keel(ctx))
	if err != nil {
		return nil, manualJournalErrors.Map(err, id.String())
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *manualjournal.Request,
) (*manualjournal.Request, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*manualjournal.Request, error) {
		if entity.ID.IsNil() {
			entity.ID = pulid.MustNew("mjr_")
		}
		entity.SyncTotals()
		assignLineTenantFields(entity)

		db := r.db.Keel(ctx)
		if err := manualjournaldb.Requests.Create(ctx, db, entity); err != nil {
			return nil, fmt.Errorf(
				"create manual journal request: %w",
				manualJournalErrors.Map(err, entity.ID.String()),
			)
		}

		if err := manualjournaldb.Lines.CreateMany(ctx, db, entity.Lines); err != nil {
			return nil, fmt.Errorf("create manual journal request lines: %w", err)
		}

		return r.get(ctx, entity.ID)
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *manualjournal.Request,
) (*manualjournal.Request, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*manualjournal.Request, error) {
		if entity.Lines != nil {
			entity.SyncTotals()
			assignLineTenantFields(entity)
		}

		db := r.db.Keel(ctx)
		mjr := manualjournaldb.Requests
		err := mjr.Save(
			ctx,
			db,
			entity,
			mjr.Status,
			mjr.Description,
			mjr.Reason,
			mjr.AccountingDate,
			mjr.RequestedFiscalYearID,
			mjr.RequestedFiscalPeriodID,
			mjr.CurrencyCode,
			mjr.TotalDebit,
			mjr.TotalCredit,
			mjr.ApprovedAt,
			mjr.ApprovedByID,
			mjr.RejectedAt,
			mjr.RejectedByID,
			mjr.RejectionReason,
			mjr.CancelledAt,
			mjr.CancelledByID,
			mjr.CancelReason,
			mjr.PostedBatchID,
			mjr.UpdatedByID,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"update manual journal request: %w",
				manualJournalErrors.Map(err, entity.ID.String()),
			)
		}

		if entity.Lines != nil {
			mjrl := manualjournaldb.Lines
			if _, err = mjrl.Where(mjrl.ManualJournalRequestID.Eq(entity.ID)).
				Delete(ctx, db); err != nil {
				return nil, fmt.Errorf("replace manual journal request lines: %w", err)
			}

			if err = mjrl.CreateMany(ctx, db, entity.Lines); err != nil {
				return nil, fmt.Errorf("insert manual journal request lines: %w", err)
			}
		}

		return r.get(ctx, entity.ID)
	})
}

func assignLineTenantFields(entity *manualjournal.Request) {
	for idx, line := range entity.Lines {
		if line == nil {
			continue
		}

		line.OrganizationID = entity.OrganizationID
		line.BusinessUnitID = entity.BusinessUnitID
		line.ManualJournalRequestID = entity.ID
		line.LineNumber = idx + 1
	}
}
