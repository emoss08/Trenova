package rateimportrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateimport"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateimport/rateimportdb"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateimportports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var batchErrors = dberror.Mapping{Entity: "RateImportBatch"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) rateimportports.RateImportRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.rateimport-repository"),
	}
}

func batchQuery(
	req *rateimportports.GetRateImportBatchByIDRequest,
) *keel.Query[rateimport.RateImportBatch] {
	b := rateimportdb.RateImportBatches
	q := b.Where(b.ID.Eq(req.RateImportBatchID))
	if req.IncludeRows {
		rows := b.Rows()
		q = q.Load(rows.Order(rows.RowNumber.Asc()))
	}

	return q
}

func (r *repository) GetByID(
	ctx context.Context,
	req *rateimportports.GetRateImportBatchByIDRequest,
) (*rateimport.RateImportBatch, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*rateimport.RateImportBatch, error) {
		entity, err := batchQuery(req).One(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get rate import batch", zap.Error(err))
			return nil, batchErrors.Map(err, req.RateImportBatchID.String())
		}

		return entity, nil
	})
}

func batchesQuery(
	req *rateimportports.ListRateImportBatchesRequest,
) *keel.Query[rateimport.RateImportBatch] {
	b := rateimportdb.RateImportBatches
	if req.RateAgreementID != nil && !req.RateAgreementID.IsNil() {
		return b.Where(b.RateAgreementID.Eq(*req.RateAgreementID))
	}

	return b.Query()
}

func (r *repository) List(
	ctx context.Context,
	req *rateimportports.ListRateImportBatchesRequest,
) (*pagination.ListResult[*rateimport.RateImportBatch], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*rateimport.RateImportBatch], error) {
			b := rateimportdb.RateImportBatches
			page := req.Filter.Pagination
			rows := batchesQuery(req).
				Order(b.CreatedAt.Desc()).
				Limit(page.SafeLimit()).
				Offset(page.SafeOffset())

			var items []*rateimport.RateImportBatch
			var total int64
			if err := keel.Batch(
				ctx,
				r.db.Keel(ctx),
				keel.ListInto(rows, &items),
				keel.CountInto(batchesQuery(req), &total),
			); err != nil {
				r.l.Error("failed to list rate import batches", zap.Error(err))
				return nil, err
			}

			return &pagination.ListResult[*rateimport.RateImportBatch]{
				Items: items,
				Total: int(total),
			}, nil
		},
	)
}

func importRowsQuery(
	req *rateimportports.ListRateImportRowsRequest,
) *keel.Query[rateimport.RateImportRow] {
	row := rateimportdb.RateImportRows

	return row.Where(
		row.RateImportBatchID.Eq(req.RateImportBatchID),
		keel.When(req.FailedOnly, row.Error.IsNotNull()),
	)
}

func (r *repository) ListRows(
	ctx context.Context,
	req *rateimportports.ListRateImportRowsRequest,
) (*pagination.ListResult[*rateimport.RateImportRow], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*rateimport.RateImportRow], error) {
			row := rateimportdb.RateImportRows
			rows := importRowsQuery(req).Order(row.RowNumber.Asc())

			var items []*rateimport.RateImportRow
			var total int64
			read := keel.AllInto(rows, &items)
			if req.Filter != nil {
				page := req.Filter.Pagination
				read = keel.ListInto(rows.Limit(page.SafeLimit()).Offset(page.SafeOffset()), &items)
			}
			if err := keel.Batch(
				ctx,
				r.db.Keel(ctx),
				read,
				keel.CountInto(importRowsQuery(req), &total),
			); err != nil {
				r.l.Error("failed to list rate import rows", zap.Error(err))
				return nil, err
			}

			return &pagination.ListResult[*rateimport.RateImportRow]{
				Items: items,
				Total: int(total),
			}, nil
		},
	)
}

// Create stages a batch and its rows together.
//
// One transaction, so a batch never exists claiming a row count it cannot show.
func (r *repository) Create(
	ctx context.Context,
	entity *rateimport.RateImportBatch,
	rows []*rateimport.RateImportRow,
) (*rateimport.RateImportBatch, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*rateimport.RateImportBatch, error) {
		db := r.db.Keel(ctx)
		err := rateimportdb.RateImportBatches.Create(ctx, db, entity)
		if err == nil && len(rows) > 0 {
			for _, row := range rows {
				row.RateImportBatchID = entity.ID
			}
			err = rateimportdb.RateImportRows.CreateMany(ctx, db, rows)
		}
		if err != nil {
			r.l.Error("failed to stage a rate import", zap.Error(err))
			return nil, batchErrors.Map(err, entity.ID.String())
		}

		entity.Rows = rows

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *rateimport.RateImportBatch,
) (*rateimport.RateImportBatch, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*rateimport.RateImportBatch, error) {
		if err := rateimportdb.RateImportBatches.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update rate import batch", zap.Error(err))
			return nil, batchErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}
