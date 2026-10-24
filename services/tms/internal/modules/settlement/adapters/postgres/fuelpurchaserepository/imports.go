package fuelpurchaserepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelpurchase/fuelpurchasedb"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/fuelpurchaseports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const importRowNumberField = "rowNumber"

func (r *repository) GetImportBatchByID(
	ctx context.Context,
	req *fuelpurchaseports.GetImportBatchByIDRequest,
) (*fuelpurchase.ImportBatch, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*fuelpurchase.ImportBatch, error) {
		b := fuelpurchasedb.ImportBatches
		q := b.Where(b.ID.Eq(req.ID))
		if req.IncludeRows {
			rows := b.Rows()
			q = q.Load(rows.Order(rows.RowNumber.Asc()))
		}

		entity, err := q.One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, importBatchErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) CreateImportBatch(
	ctx context.Context,
	entity *fuelpurchase.ImportBatch,
) (*fuelpurchase.ImportBatch, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*fuelpurchase.ImportBatch, error) {
		if err := fuelpurchasedb.ImportBatches.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create fuel purchase import batch", zap.Error(err))
			return nil, importBatchErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateImportBatch(
	ctx context.Context,
	entity *fuelpurchase.ImportBatch,
) (*fuelpurchase.ImportBatch, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*fuelpurchase.ImportBatch, error) {
		if err := r.saveBatch(ctx, entity); err != nil {
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) saveBatch(ctx context.Context, entity *fuelpurchase.ImportBatch) error {
	if err := fuelpurchasedb.ImportBatches.Save(ctx, r.db.Keel(ctx), entity); err != nil {
		r.l.Error("failed to update fuel purchase import batch", zap.Error(err))
		return importBatchErrors.Map(err, entity.ID.String())
	}

	return nil
}

func (r *repository) ReplaceImportRows(
	ctx context.Context,
	batch *fuelpurchase.ImportBatch,
	rows []*fuelpurchase.ImportRow,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.Keel(ctx)
		ir := fuelpurchasedb.ImportRows
		if _, err := ir.Where(ir.ImportBatchID.Eq(batch.ID)).Delete(ctx, db); err != nil {
			r.l.Error("failed to clear fuel purchase import rows", zap.Error(err))
			return fmt.Errorf("replace fuel purchase import rows: clear import rows: %w", err)
		}

		for _, row := range rows {
			row.ImportBatchID = batch.ID
			row.OrganizationID = batch.OrganizationID
			row.BusinessUnitID = batch.BusinessUnitID
		}

		if err := ir.CreateMany(ctx, db, rows); err != nil {
			r.l.Error("failed to insert fuel purchase import rows", zap.Error(err))
			return fmt.Errorf(
				"replace fuel purchase import rows: insert import rows: %w",
				importRowErrors.Map(err, batch.ID.String()),
			)
		}

		return nil
	})
}

func (r *repository) ListImportRows(
	ctx context.Context,
	req *fuelpurchaseports.ListImportRowsRequest,
) (*pagination.CursorListResult[*fuelpurchase.ImportRow], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*fuelpurchase.ImportRow], error) {
			ir := fuelpurchasedb.ImportRows
			filter := pagination.QueryOptions{}
			if req.Filter != nil {
				filter = *req.Filter
			}
			filter.Sort = []domaintypes.SortField{
				{Field: importRowNumberField, Direction: dbtype.SortDirectionAsc},
			}

			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				ir.Where(
					ir.ImportBatchID.Eq(req.BatchID),
					when(len(req.Statuses) > 0, ir.Status.In(req.Statuses...)),
				),
				&listquery.Request[fuelpurchase.ImportRow]{
					Filter: &filter,
					Cursor: req.Cursor,
				},
			)
			if err != nil {
				r.l.Error("failed to list fuel purchase import rows", zap.Error(err))
				return nil, fmt.Errorf("list fuel purchase import rows: %w", err)
			}

			return result, nil
		},
	)
}

func (r *repository) CommitImport(
	ctx context.Context,
	req *fuelpurchaseports.CommitImportRequest,
) (*fuelpurchaseports.CommitImportResult, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*fuelpurchaseports.CommitImportResult, error) {
			result, err := r.commitImport(ctx, req)
			if err != nil {
				if dberror.IsUniqueConstraintViolation(err) {
					return nil, duplicateReference()
				}
				r.l.Error("failed to commit fuel purchase import", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) commitImport(
	ctx context.Context,
	req *fuelpurchaseports.CommitImportRequest,
) (*fuelpurchaseports.CommitImportResult, error) {
	batch := req.Batch
	for _, purchase := range req.Purchases {
		purchase.OrganizationID = batch.OrganizationID
		purchase.BusinessUnitID = batch.BusinessUnitID
		purchase.ImportBatchID = &batch.ID
		purchase.Source = fuelpurchase.PurchaseSourceCardImport
	}

	inserted, err := r.insertImportedPurchases(ctx, req.Purchases)
	if err != nil {
		return nil, err
	}

	committedRowIDs := make([]pulid.ID, 0, len(inserted))
	committedPurchaseIDs := make([]pulid.ID, 0, len(inserted))
	skippedRowIDs := make([]pulid.ID, 0, len(req.Purchases)-len(inserted))
	for _, purchase := range req.Purchases {
		rowID, ok := req.RowIDByReference[purchase.TransactionReference]
		if !ok {
			continue
		}
		purchaseID, wasInserted := inserted[purchase.TransactionReference]
		if !wasInserted {
			skippedRowIDs = append(skippedRowIDs, rowID)
			continue
		}
		committedRowIDs = append(committedRowIDs, rowID)
		committedPurchaseIDs = append(committedPurchaseIDs, purchaseID)
	}

	if err = r.markRowsCommitted(ctx, batch.ID, committedRowIDs, committedPurchaseIDs); err != nil {
		return nil, err
	}
	if err = r.markRowsAlreadyImported(ctx, batch.ID, skippedRowIDs); err != nil {
		return nil, err
	}

	batch.CommittedCount += len(inserted)
	batch.Error = ""
	if batch.Summary != nil {
		batch.Summary.NewCount = 0
		batch.Summary.AlreadyImportedCount += len(skippedRowIDs)
	}
	if !req.KeepOpen {
		committedAt := req.CommittedAt
		batch.Status = fuelpurchase.ImportStatusCommitted
		batch.CommittedAt = &committedAt
		batch.CommittedByID = req.CommittedByID
	}
	if err = r.saveBatch(ctx, batch); err != nil {
		return nil, err
	}

	return &fuelpurchaseports.CommitImportResult{
		Committed:       len(inserted),
		AlreadyImported: len(skippedRowIDs),
	}, nil
}

func (r *repository) insertImportedPurchases(
	ctx context.Context,
	purchases []*fuelpurchase.FuelPurchase,
) (map[string]pulid.ID, error) {
	inserted := make(map[string]pulid.ID, len(purchases))
	if len(purchases) == 0 {
		return inserted, nil
	}

	db := r.db.Keel(ctx)
	fp := fuelpurchasedb.FuelPurchases
	n, err := fp.CreateOrSkip(ctx, db, purchases)
	if err != nil {
		return nil, fmt.Errorf("insert imported purchases: %w", err)
	}

	if int(n) == len(purchases) {
		for _, purchase := range purchases {
			inserted[purchase.TransactionReference] = purchase.ID
		}
		return inserted, nil
	}

	ids := make([]pulid.ID, len(purchases))
	for i, purchase := range purchases {
		ids[i] = purchase.ID
	}
	rows, err := keel.Into[referenceRow](
		ctx,
		db,
		fp.Pick(fp.ID, fp.TransactionReference).Where(fp.ID.In(ids...)),
	)
	if err != nil {
		return nil, fmt.Errorf("read imported purchases: %w", err)
	}
	for _, row := range rows {
		inserted[row.Reference] = row.ID
	}

	return inserted, nil
}

func (r *repository) markRowsCommitted(
	ctx context.Context,
	batchID pulid.ID,
	rowIDs, purchaseIDs []pulid.ID,
) error {
	if len(rowIDs) == 0 {
		return nil
	}

	ir := fuelpurchasedb.ImportRows
	committed := keel.Unnest("committed", []string{"row_id", "purchase_id"}, rowIDs, purchaseIDs)
	_, err := ir.With(committed).
		Join(committed, committed.Col("row_id").EqCol(ir.ID)).
		Where(ir.ImportBatchID.Eq(batchID)).
		Update(
			ctx,
			r.db.Keel(ctx),
			ir.Status.To(fuelpurchase.ImportRowStatusCommitted),
			ir.FuelPurchaseID.ToExpr(committed.Col("purchase_id")),
			ir.Error.ToNull(),
		)
	if err != nil {
		return fmt.Errorf("mark import rows committed: %w", err)
	}

	return nil
}

func (r *repository) markRowsAlreadyImported(
	ctx context.Context,
	batchID pulid.ID,
	rowIDs []pulid.ID,
) error {
	if len(rowIDs) == 0 {
		return nil
	}

	ir := fuelpurchasedb.ImportRows
	_, err := ir.Where(
		ir.ImportBatchID.Eq(batchID),
		ir.ID.In(rowIDs...),
	).Update(ctx, r.db.Keel(ctx), ir.Status.To(fuelpurchase.ImportRowStatusAlreadyImported))
	if err != nil {
		return fmt.Errorf("mark import rows already imported: %w", err)
	}

	return nil
}

func importBatchFilters(req *fuelpurchaseports.ListImportBatchesRequest) []keel.Pred {
	b := fuelpurchasedb.ImportBatches

	return []keel.Pred{
		when(req.Origin != "", b.Origin.Eq(req.Origin)),
		when(req.Provider != "", b.Provider.Eq(req.Provider)),
		when(len(req.Statuses) > 0, b.Status.In(req.Statuses...)),
		when(req.HeldRowsOnly, b.ErrorCount.Gt(0)),
	}
}

func (r *repository) ListImportBatches(
	ctx context.Context,
	req *fuelpurchaseports.ListImportBatchesRequest,
) (*pagination.CursorListResult[*fuelpurchase.ImportBatch], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*fuelpurchase.ImportBatch], error) {
			b := fuelpurchasedb.ImportBatches
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				b.Where(importBatchFilters(req)...),
				&listquery.Request[fuelpurchase.ImportBatch]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape: func(
						q *keel.Query[fuelpurchase.ImportBatch],
					) *keel.Query[fuelpurchase.ImportBatch] {
						if req.IncludeDefaultCard {
							return q.Load(b.DefaultFuelCard())
						}
						return q
					},
				},
			)
			if err != nil {
				r.l.Error("failed to list fuel purchase imports", zap.Error(err))
				return nil, fmt.Errorf("list fuel purchase imports: %w", err)
			}

			return result, nil
		},
	)
}
