package rateimportrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/rateimport"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// rowInsertBatch bounds how many staged rows go in one statement. A class
// tariff runs to thousands, and one insert that size holds locks far longer
// than the work deserves.
const rowInsertBatch = 500

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.RateImportRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.rateimport-repository"),
	}
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetRateImportBatchByIDRequest,
) (*rateimport.RateImportBatch, error) {
	entity := new(rateimport.RateImportBatch)
	cols := buncolgen.RateImportBatchColumns

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RateImportBatchScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.RateImportBatchID)
		})

	if req.IncludeRows {
		q = q.Relation(
			buncolgen.Rel(buncolgen.RateImportBatchRelations.Rows),
			func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Order(buncolgen.RateImportRowColumns.RowNumber.OrderAsc())
			},
		)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to get rate import batch", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "RateImportBatch")
	}

	return entity, nil
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListRateImportBatchesRequest,
) (*pagination.ListResult[*rateimport.RateImportBatch], error) {
	cols := buncolgen.RateImportBatchColumns
	entities := make([]*rateimport.RateImportBatch, 0, req.Filter.Pagination.SafeLimit())

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(buncolgen.RateImportBatchApplyTenant(req.Filter.TenantInfo))

	if req.RateAgreementID != nil && !req.RateAgreementID.IsNil() {
		q = q.Where(cols.RateAgreementID.Eq(), *req.RateAgreementID)
	}

	// Newest first: an import is reviewed while it is fresh.
	q = q.Order(cols.CreatedAt.OrderDesc()).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset())

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		r.l.Error("failed to list rate import batches", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*rateimport.RateImportBatch]{Items: entities, Total: total}, nil
}

func (r *repository) ListRows(
	ctx context.Context,
	req *repositories.ListRateImportRowsRequest,
) (*pagination.ListResult[*rateimport.RateImportRow], error) {
	cols := buncolgen.RateImportRowColumns
	entities := make([]*rateimport.RateImportRow, 0)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RateImportRowScopeTenant(sq, req.TenantInfo).
				Where(cols.RateImportBatchID.Eq(), req.RateImportBatchID)
		})

	// The rows that would not read are what somebody opened the import to fix,
	// and on a good sheet there are none of them among thousands.
	if req.FailedOnly {
		q = q.Where(cols.Error.IsNotNull())
	}

	q = q.Order(cols.RowNumber.OrderAsc())

	if req.Filter != nil {
		q = q.Limit(req.Filter.Pagination.SafeLimit()).
			Offset(req.Filter.Pagination.SafeOffset())
	}

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		r.l.Error("failed to list rate import rows", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*rateimport.RateImportRow]{Items: entities, Total: total}, nil
}

// Create stages a batch and its rows together.
//
// One transaction, so a batch never exists claiming a row count it cannot show.
func (r *repository) Create(
	ctx context.Context,
	entity *rateimport.RateImportBatch,
	rows []*rateimport.RateImportRow,
) (*rateimport.RateImportBatch, error) {
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if _, iErr := r.db.DBForContext(c).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(c); iErr != nil {
			return iErr
		}

		for _, row := range rows {
			row.RateImportBatchID = entity.ID
		}

		for start := 0; start < len(rows); start += rowInsertBatch {
			batch := rows[start:min(start+rowInsertBatch, len(rows))]

			if _, iErr := r.db.DBForContext(c).
				NewInsert().
				Model(&batch).
				Exec(c); iErr != nil {
				return iErr
			}
		}

		return nil
	})
	if err != nil {
		r.l.Error("failed to stage a rate import", zap.Error(err))
		return nil, err
	}

	entity.Rows = rows

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *rateimport.RateImportBatch,
) (*rateimport.RateImportBatch, error) {
	ov := entity.Version
	entity.Version++

	cols := buncolgen.RateImportBatchColumns

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(cols.Version.Eq(), ov).
		ExcludeColumn("rows").
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update rate import batch", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(
		results, "RateImportBatch", entity.ID.String(),
	); err != nil {
		return nil, err
	}

	return entity, nil
}
