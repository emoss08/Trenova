package fuelpurchaserepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func (r *repository) GetImportBatchByID(
	ctx context.Context,
	req *repositories.GetImportBatchByIDRequest,
) (*fuelpurchase.ImportBatch, error) {
	entity := new(fuelpurchase.ImportBatch)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ImportBatchScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.ImportBatchColumns.ID.Eq(), req.ID)
		})

	if req.IncludeRows {
		q = q.Relation(
			buncolgen.ImportBatchRelations.Rows,
			func(rq *bun.SelectQuery) *bun.SelectQuery {
				return rq.Order(buncolgen.ImportRowColumns.RowNumber.OrderAsc())
			},
		)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "FuelPurchaseImportBatch")
	}

	return entity, nil
}

func (r *repository) CreateImportBatch(
	ctx context.Context,
	entity *fuelpurchase.ImportBatch,
) (*fuelpurchase.ImportBatch, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to create fuel purchase import batch", zap.Error(err))
		return nil, fmt.Errorf("create fuel purchase import batch: %w", err)
	}

	return entity, nil
}

func (r *repository) UpdateImportBatch(
	ctx context.Context,
	entity *fuelpurchase.ImportBatch,
) (*fuelpurchase.ImportBatch, error) {
	updated, err := r.updateBatch(ctx, r.db.DBForContext(ctx), entity)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (r *repository) updateBatch(
	ctx context.Context,
	dba bun.IDB,
	entity *fuelpurchase.ImportBatch,
) (*fuelpurchase.ImportBatch, error) {
	ov := entity.Version
	entity.Version++

	results, err := dba.
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.ImportBatchColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		entity.Version = ov
		r.l.Error("failed to update fuel purchase import batch", zap.Error(err))
		return nil, fmt.Errorf("update fuel purchase import batch: %w", err)
	}
	if err = dberror.CheckRowsAffected(
		results,
		"FuelPurchaseImportBatch",
		entity.ID.String(),
	); err != nil {
		entity.Version = ov
		return nil, err
	}

	return entity, nil
}

func (r *repository) ReplaceImportRows(
	ctx context.Context,
	batch *fuelpurchase.ImportBatch,
	rows []*fuelpurchase.ImportRow,
) error {
	tenant := pagination.TenantInfo{OrgID: batch.OrganizationID, BuID: batch.BusinessUnitID}

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, tx bun.Tx) error {
		if _, dErr := tx.NewDelete().
			Model((*fuelpurchase.ImportRow)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return buncolgen.ImportRowScopeTenantDelete(dq, tenant).
					Where(buncolgen.ImportRowColumns.ImportBatchID.Eq(), batch.ID)
			}).
			Exec(txCtx); dErr != nil {
			return fmt.Errorf("clear import rows: %w", dErr)
		}

		for _, row := range rows {
			row.ImportBatchID = batch.ID
			row.OrganizationID = batch.OrganizationID
			row.BusinessUnitID = batch.BusinessUnitID
		}

		for start := 0; start < len(rows); start += rowInsertBatch {
			chunk := rows[start:min(start+rowInsertBatch, len(rows))]
			if _, iErr := tx.NewInsert().Model(&chunk).Exec(txCtx); iErr != nil {
				return fmt.Errorf("insert import rows: %w", iErr)
			}
		}

		return nil
	})
	if err != nil {
		r.l.Error("failed to replace fuel purchase import rows", zap.Error(err))
		return fmt.Errorf("replace fuel purchase import rows: %w", err)
	}

	return nil
}

func (r *repository) ListImportRows(
	ctx context.Context,
	req *repositories.ListImportRowsRequest,
) (*pagination.CursorListResult[*fuelpurchase.ImportRow], error) {
	cols := buncolgen.ImportRowColumns
	dba := r.db.DBForContext(ctx)

	scope := func(sq *bun.SelectQuery) *bun.SelectQuery {
		sq = buncolgen.ImportRowScopeTenant(sq, req.TenantInfo).
			Where(cols.ImportBatchID.Eq(), req.BatchID)
		if len(req.Statuses) > 0 {
			sq = sq.Where(cols.Status.In(), bun.In(req.Statuses))
		}
		return sq
	}

	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.NewSelect().
			Model((*fuelpurchase.ImportRow)(nil)).
			WhereGroup(" AND ", scope).
			Count(ctx)
		if err != nil {
			r.l.Error("failed to count fuel purchase import rows", zap.Error(err))
			return nil, fmt.Errorf("count fuel purchase import rows: %w", err)
		}
		totalCount = &total
	}

	cursor, err := importRowCursor(req.Cursor)
	if err != nil {
		return nil, err
	}

	limit := req.Cursor.Limit
	if limit <= 0 {
		if req.Filter != nil {
			limit = req.Filter.Pagination.SafeLimit()
		} else {
			limit = pagination.DefaultLimit
		}
	}

	rows := make([]*fuelpurchase.ImportRow, 0, limit+1)
	q := dba.NewSelect().
		Model(&rows).
		ColumnExpr(buncolgen.ImportRowTable.All()).
		WhereGroup(" AND ", scope)

	if !cursor.ID.IsNil() {
		anchor := dba.NewSelect().
			Model((*fuelpurchase.ImportRow)(nil)).
			Column(cols.RowNumber.String(), cols.ID.String()).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.ImportRowScopeTenant(sq, req.TenantInfo).
					Where(cols.ImportBatchID.Eq(), req.BatchID).
					Where(cols.ID.Eq(), cursor.ID)
			})
		q = q.Where(buncolgen.Expr("({0}, {1}) > ?", cols.RowNumber, cols.ID), anchor)
	}

	err = q.Order(cols.RowNumber.OrderAsc()).
		Order(cols.ID.OrderAsc()).
		Limit(limit + 1).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list fuel purchase import rows", zap.Error(err))
		return nil, fmt.Errorf("list fuel purchase import rows: %w", err)
	}

	return pagination.NewCursorListResultWithTotalCount(rows, limit, totalCount), nil
}

func importRowCursor(info pagination.CursorInfo) (pagination.Cursor, error) {
	if !info.Cursor.ID.IsNil() {
		return info.Cursor, nil
	}
	if info.After == "" {
		return pagination.Cursor{}, nil
	}

	cursor, err := pagination.DecodeCursor(info.After)
	if err != nil {
		return pagination.Cursor{}, fmt.Errorf("decode import row cursor: %w", err)
	}

	return cursor, nil
}

type committedPurchase struct {
	ID        pulid.ID `bun:"id"`
	Reference string   `bun:"transaction_reference"`
}

func (r *repository) CommitImport(
	ctx context.Context,
	req *repositories.CommitImportRequest,
) (*repositories.CommitImportResult, error) {
	cols := buncolgen.FuelPurchaseColumns
	rowCols := buncolgen.ImportRowColumns
	batch := req.Batch
	tenant := pagination.TenantInfo{OrgID: batch.OrganizationID, BuID: batch.BusinessUnitID}
	conflictClause := "CONFLICT (" + cols.OrganizationID.Name + ", " + cols.BusinessUnitID.Name +
		", " + cols.TransactionReference.Name + ") WHERE " + cols.TransactionReference.Name +
		" IS NOT NULL DO NOTHING"

	result := &repositories.CommitImportResult{}

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, tx bun.Tx) error {
		for _, purchase := range req.Purchases {
			purchase.OrganizationID = batch.OrganizationID
			purchase.BusinessUnitID = batch.BusinessUnitID
			purchase.ImportBatchID = &batch.ID
			purchase.Source = fuelpurchase.PurchaseSourceCardImport
		}

		inserted := make(map[string]pulid.ID, len(req.Purchases))
		insertedCount := 0
		for start := 0; start < len(req.Purchases); start += purchaseInsertBatch {
			chunk := req.Purchases[start:min(start+purchaseInsertBatch, len(req.Purchases))]
			returned := make([]committedPurchase, 0, len(chunk))
			if _, iErr := tx.NewInsert().
				Model(&chunk).
				On(conflictClause).
				Returning(cols.ID.Name+", "+cols.TransactionReference.Name).
				Exec(txCtx, &returned); iErr != nil {
				return fmt.Errorf("insert imported purchases: %w", iErr)
			}
			insertedCount += len(returned)
			for _, row := range returned {
				inserted[row.Reference] = row.ID
			}
		}

		committedRows := make([]*fuelpurchase.ImportRow, 0, len(inserted))
		skippedRowIDs := make([]pulid.ID, 0, len(req.Purchases)-insertedCount)
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
			committedRows = append(committedRows, &fuelpurchase.ImportRow{
				ID:             rowID,
				OrganizationID: batch.OrganizationID,
				BusinessUnitID: batch.BusinessUnitID,
				FuelPurchaseID: &purchaseID,
			})
		}

		for start := 0; start < len(committedRows); start += rowInsertBatch {
			chunk := committedRows[start:min(start+rowInsertBatch, len(committedRows))]
			tuples := make([][]any, 0, len(chunk))
			for _, row := range chunk {
				tuples = append(tuples, []any{row.ID, *row.FuelPurchaseID})
			}
			if _, uErr := tx.NewUpdate().
				Model((*fuelpurchase.ImportRow)(nil)).
				TableExpr("(VALUES ?) AS committed (row_id, purchase_id)", bun.In(tuples)).
				Set(rowCols.Status.Set(), fuelpurchase.ImportRowStatusCommitted).
				Set(rowCols.FuelPurchaseID.SetExpr("committed.purchase_id")).
				Set(rowCols.Error.SetNull()).
				WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
					return buncolgen.ImportRowScopeTenantUpdate(uq, tenant).
						Where(rowCols.ImportBatchID.Eq(), batch.ID).
						Where(rowCols.ID.Expr("{} = committed.row_id"))
				}).
				Exec(txCtx); uErr != nil {
				return fmt.Errorf("mark import rows committed: %w", uErr)
			}
		}

		if len(skippedRowIDs) > 0 {
			if _, uErr := tx.NewUpdate().
				Model((*fuelpurchase.ImportRow)(nil)).
				Set(rowCols.Status.Set(), fuelpurchase.ImportRowStatusAlreadyImported).
				WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
					return buncolgen.ImportRowScopeTenantUpdate(uq, tenant).
						Where(rowCols.ImportBatchID.Eq(), batch.ID).
						Where(rowCols.ID.In(), bun.List(skippedRowIDs))
				}).
				Exec(txCtx); uErr != nil {
				return fmt.Errorf("mark import rows already imported: %w", uErr)
			}
		}

		committedAt := req.CommittedAt
		batch.Status = fuelpurchase.ImportStatusCommitted
		batch.CommittedCount = insertedCount
		batch.CommittedAt = &committedAt
		batch.CommittedByID = req.CommittedByID
		batch.Error = ""
		if batch.Summary != nil {
			batch.Summary.NewCount = 0
			batch.Summary.AlreadyImportedCount += len(skippedRowIDs)
		}
		if _, uErr := r.updateBatch(txCtx, tx, batch); uErr != nil {
			return uErr
		}

		result.Committed = insertedCount
		result.AlreadyImported = len(skippedRowIDs)

		return nil
	})
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, duplicateReference()
		}
		r.l.Error("failed to commit fuel purchase import", zap.Error(err))
		return nil, err
	}

	return result, nil
}
