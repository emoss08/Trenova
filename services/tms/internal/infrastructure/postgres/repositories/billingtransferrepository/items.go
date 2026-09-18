package billingtransferrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/billingtransfer"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// seedChunkSize bounds one INSERT's parameter count. A run can carry 5000
// shipments and Postgres caps a statement at 65535 bind parameters.
const seedChunkSize = 500

// nonRetryableFailureCodes are the answers a second attempt cannot change. They
// are listed here, rather than derived per row, because the retryable counter is
// computed by the database in a single aggregate.
var nonRetryableFailureCodes = []billingtransfer.FailureCode{
	billingtransfer.FailureNotFound,
	billingtransfer.FailureAlreadyTransferred,
}

// SeedItems writes one Pending row per shipment, in request order, before any
// work starts. Seeding up front is what makes the report complete from the first
// moment: a run that is cancelled or dies still knows which shipments it never
// reached, instead of having to infer them from a list it no longer holds.
//
// ON CONFLICT DO NOTHING against uq_billing_transfer_run_items_run_shipment makes
// a retried seed a no-op rather than a duplicate-key failure.
func (r *repository) SeedItems(
	ctx context.Context,
	req *repositories.SeedBillingTransferRunItemsRequest,
) (int, error) {
	if len(req.ShipmentIDs) == 0 {
		return 0, nil
	}

	rows := make([]*billingtransfer.BillingTransferRunItem, 0, len(req.ShipmentIDs))
	for i, shipmentID := range req.ShipmentIDs {
		rows = append(rows, &billingtransfer.BillingTransferRunItem{
			BusinessUnitID:      req.TenantInfo.BuID,
			OrganizationID:      req.TenantInfo.OrgID,
			RunID:               req.RunID,
			ShipmentID:          shipmentID,
			Sequence:            i,
			Status:              billingtransfer.ItemStatusPending,
			MissingRequirements: []billingtransfer.MissingRequirement{},
			ValidationFailures:  []billingtransfer.ValidationFailure{},
		})
	}

	inserted := 0
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, tx bun.Tx) error {
		for start := 0; start < len(rows); start += seedChunkSize {
			end := min(start+seedChunkSize, len(rows))

			chunk := rows[start:end]
			result, execErr := tx.NewInsert().
				Model(&chunk).
				On("CONFLICT DO NOTHING").
				Exec(txCtx)
			if execErr != nil {
				return execErr
			}

			affected, raErr := result.RowsAffected()
			if raErr != nil {
				return raErr
			}
			inserted += int(affected)
		}

		return nil
	})
	if err != nil {
		r.l.Error("failed to seed billing transfer run items", zap.Error(err))
		return 0, err
	}

	return inserted, nil
}

// SeedRetryItems copies the source run's items forward.
//
// Every shipment comes across, not only the ones being retried: outcomes a
// second attempt cannot change keep the answer they already have, and the rest
// reset to Pending. That way the retry run reads as one complete report rather
// than a fragment the reader has to mentally merge with the run before it.
func (r *repository) SeedRetryItems(
	ctx context.Context,
	req *repositories.SeedBillingTransferRetryItemsRequest,
) (*repositories.SeedBillingTransferRetryItemsResult, error) {
	cols := buncolgen.BillingTransferRunItemColumns

	source := make([]*billingtransfer.BillingTransferRunItem, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&source).
		Apply(buncolgen.BillingTransferRunItemApplyTenant(req.TenantInfo)).
		Where(cols.RunID.Eq(), req.SourceRunID).
		Order(cols.Sequence.OrderAsc()).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to read source run items for retry", zap.Error(err))
		return nil, err
	}

	out := &repositories.SeedBillingTransferRetryItemsResult{}
	rows := make([]*billingtransfer.BillingTransferRunItem, 0, len(source))
	for i, item := range source {
		row := &billingtransfer.BillingTransferRunItem{
			BusinessUnitID:      req.TenantInfo.BuID,
			OrganizationID:      req.TenantInfo.OrgID,
			RunID:               req.RunID,
			ShipmentID:          item.ShipmentID,
			Sequence:            i,
			ProNumber:           item.ProNumber,
			MissingRequirements: []billingtransfer.MissingRequirement{},
			ValidationFailures:  []billingtransfer.ValidationFailure{},
		}

		if retryableItem(item) {
			row.Status = billingtransfer.ItemStatusPending
			out.PendingCount++
		} else {
			row.Status = item.Status
			row.FailureCode = item.FailureCode
			row.ErrorMessage = item.ErrorMessage
			row.MarkedReadyToInvoice = item.MarkedReadyToInvoice
			row.BillingQueueItemID = item.BillingQueueItemID
			row.BillingQueueNumber = item.BillingQueueNumber
			row.BillingQueueStatus = item.BillingQueueStatus
			row.MissingRequirements = item.MissingRequirements
			row.ValidationFailures = item.ValidationFailures
			row.ProcessedAt = item.ProcessedAt

			switch item.Status {
			case billingtransfer.ItemStatusTransferred:
				out.TransferredCount++
			case billingtransfer.ItemStatusNotTransferred:
				out.NotTransferredCount++
			case billingtransfer.ItemStatusPending, billingtransfer.ItemStatusSkipped:
			}
			if item.MarkedReadyToInvoice {
				out.MarkedReadyToInvoiceCount++
			}
		}

		rows = append(rows, row)
	}

	out.TotalCount = len(rows)
	if len(rows) == 0 {
		return out, nil
	}

	err = r.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, tx bun.Tx) error {
		for start := 0; start < len(rows); start += seedChunkSize {
			end := min(start+seedChunkSize, len(rows))

			chunk := rows[start:end]
			if _, execErr := tx.NewInsert().
				Model(&chunk).
				On("CONFLICT DO NOTHING").
				Exec(txCtx); execErr != nil {
				return execErr
			}
		}

		return nil
	})
	if err != nil {
		r.l.Error("failed to seed retry billing transfer run items", zap.Error(err))
		return nil, err
	}

	return out, nil
}

// retryableItem reports whether a second attempt could answer differently. A
// shipment the run never reached is always worth retrying; one that failed is
// only worth retrying when the reason might have changed.
func retryableItem(item *billingtransfer.BillingTransferRunItem) bool {
	switch item.Status {
	case billingtransfer.ItemStatusPending, billingtransfer.ItemStatusSkipped:
		return true
	case billingtransfer.ItemStatusNotTransferred:
		return item.FailureCode.IsRetryable()
	case billingtransfer.ItemStatusTransferred:
		return false
	default:
		return false
	}
}

// NextPendingShipmentIDs claims the next batch in request order. It reads only
// Pending rows, so an activity that is retried after recording part of its work
// picks up where it left off instead of transferring a shipment twice.
func (r *repository) NextPendingShipmentIDs(
	ctx context.Context,
	req *repositories.NextPendingBillingTransferItemsRequest,
) ([]pulid.ID, error) {
	cols := buncolgen.BillingTransferRunItemColumns

	ids := make([]pulid.ID, 0, req.Limit)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*billingtransfer.BillingTransferRunItem)(nil)).
		Column(cols.ShipmentID.Bare()).
		Apply(buncolgen.BillingTransferRunItemApplyTenant(req.TenantInfo)).
		Where(cols.RunID.Eq(), req.RunID).
		Where(cols.Status.Eq(), billingtransfer.ItemStatusPending).
		Order(cols.Sequence.OrderAsc()).
		Limit(req.Limit).
		Scan(ctx, &ids)
	if err != nil {
		r.l.Error("failed to claim next billing transfer batch", zap.Error(err))
		return nil, err
	}

	return ids, nil
}

// outcomeColumns is the exact set a recorded outcome may write. Updating through
// the model rather than a hand-written SET clause is what makes the `nullzero`
// tags apply: an unset failure code or billing queue status has to reach Postgres
// as NULL, because "" is neither a valid enum label nor allowed by the row's
// check constraint.
var outcomeColumns = []string{
	buncolgen.BillingTransferRunItemColumns.Status.Bare(),
	buncolgen.BillingTransferRunItemColumns.ProNumber.Bare(),
	buncolgen.BillingTransferRunItemColumns.FailureCode.Bare(),
	buncolgen.BillingTransferRunItemColumns.ErrorMessage.Bare(),
	buncolgen.BillingTransferRunItemColumns.MarkedReadyToInvoice.Bare(),
	buncolgen.BillingTransferRunItemColumns.BillingQueueItemID.Bare(),
	buncolgen.BillingTransferRunItemColumns.BillingQueueNumber.Bare(),
	buncolgen.BillingTransferRunItemColumns.BillingQueueStatus.Bare(),
	buncolgen.BillingTransferRunItemColumns.MissingRequirements.Bare(),
	buncolgen.BillingTransferRunItemColumns.ValidationFailures.Bare(),
	buncolgen.BillingTransferRunItemColumns.ProcessedAt.Bare(),
	buncolgen.BillingTransferRunItemColumns.UpdatedAt.Bare(),
}

func outcomeRow(
	tenant pagination.TenantInfo,
	runID pulid.ID,
	outcome repositories.BillingTransferItemOutcome,
	now int64,
) *billingtransfer.BillingTransferRunItem {
	processedAt := outcome.ProcessedAt
	if processedAt == 0 {
		processedAt = now
	}

	row := &billingtransfer.BillingTransferRunItem{
		BusinessUnitID:       tenant.BuID,
		OrganizationID:       tenant.OrgID,
		RunID:                runID,
		ShipmentID:           outcome.ShipmentID,
		ProNumber:            outcome.ProNumber,
		Status:               outcome.Status,
		FailureCode:          outcome.FailureCode,
		ErrorMessage:         outcome.ErrorMessage,
		MarkedReadyToInvoice: outcome.MarkedReadyToInvoice,
		BillingQueueItemID:   outcome.BillingQueueItemID,
		BillingQueueNumber:   outcome.BillingQueueNumber,
		BillingQueueStatus:   outcome.BillingQueueStatus,
		MissingRequirements:  outcome.MissingRequirements,
		ValidationFailures:   outcome.ValidationFailures,
		ProcessedAt:          &processedAt,
	}
	// Both columns are NOT NULL jsonb, so an absent list is an empty one.
	if row.MissingRequirements == nil {
		row.MissingRequirements = []billingtransfer.MissingRequirement{}
	}
	if row.ValidationFailures == nil {
		row.ValidationFailures = []billingtransfer.ValidationFailure{}
	}

	return row
}

// RecordItemOutcomes writes a batch's answers and recomputes the run's counters.
//
// The counters are derived from the item rows rather than incremented. That is
// the whole point: Temporal will retry an activity whose work already landed,
// and an increment would count it twice. A recomputation cannot, however many
// times it runs.
func (r *repository) RecordItemOutcomes(
	ctx context.Context,
	req *repositories.RecordBillingTransferOutcomesRequest,
) (*repositories.BillingTransferRunProgress, error) {
	cols := buncolgen.BillingTransferRunItemColumns
	now := timeutils.NowUnix()

	progress := new(repositories.BillingTransferRunProgress)
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, tx bun.Tx) error {
		for _, outcome := range req.Outcomes {
			row := outcomeRow(req.TenantInfo, req.RunID, outcome, now)

			if _, execErr := tx.NewUpdate().
				Model(row).
				Column(outcomeColumns...).
				Apply(func(uq *bun.UpdateQuery) *bun.UpdateQuery {
					return buncolgen.BillingTransferRunItemScopeTenantUpdate(uq, req.TenantInfo)
				}).
				Where(cols.RunID.Eq(), req.RunID).
				Where(cols.ShipmentID.Eq(), outcome.ShipmentID).
				Exec(txCtx); execErr != nil {
				return execErr
			}
		}

		return r.refreshRunCounters(txCtx, tx, req.TenantInfo, req.RunID, now, progress)
	})
	if err != nil {
		r.l.Error("failed to record billing transfer outcomes", zap.Error(err))
		return nil, err
	}

	return progress, nil
}

// Finalize closes the run out. Whatever is still Pending becomes Skipped —
// the durable record of "this run never got to these" — and the counters are
// recomputed one last time so the report and the summary agree.
func (r *repository) Finalize(
	ctx context.Context,
	req *repositories.FinalizeBillingTransferRunRequest,
) (*billingtransfer.BillingTransferRun, error) {
	itemCols := buncolgen.BillingTransferRunItemColumns
	runCols := buncolgen.BillingTransferRunColumns
	now := timeutils.NowUnix()

	entity := new(billingtransfer.BillingTransferRun)
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, tx bun.Tx) error {
		if _, execErr := tx.NewUpdate().
			Model((*billingtransfer.BillingTransferRunItem)(nil)).
			Set(itemCols.Status.Set(), billingtransfer.ItemStatusSkipped).
			Set(itemCols.ProcessedAt.Set(), now).
			Set(itemCols.UpdatedAt.Set(), now).
			Apply(func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.BillingTransferRunItemScopeTenantUpdate(uq, req.TenantInfo)
			}).
			Where(itemCols.RunID.Eq(), req.RunID).
			Where(itemCols.Status.Eq(), billingtransfer.ItemStatusPending).
			Exec(txCtx); execErr != nil {
			return execErr
		}

		progress := new(repositories.BillingTransferRunProgress)
		if err := r.refreshRunCounters(
			txCtx, tx, req.TenantInfo, req.RunID, now, progress,
		); err != nil {
			return err
		}

		result, execErr := tx.NewUpdate().
			Model(entity).
			Set(runCols.Status.Set(), req.Status).
			Set(runCols.FailureMessage.Set(), req.FailureMessage).
			Set(runCols.CompletedAt.SetExpr("COALESCE({}, ?)"), now).
			Set(runCols.UpdatedAt.Set(), now).
			Set(runCols.Version.Inc(1)).
			Apply(func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.BillingTransferRunScopeTenantUpdate(uq, req.TenantInfo)
			}).
			Where(runCols.ID.Eq(), req.RunID).
			Returning("*").
			Exec(txCtx)
		if execErr != nil {
			return execErr
		}

		return dberror.CheckRowsAffected(result, "BillingTransferRun", req.RunID.String())
	})
	if err != nil {
		r.l.Error("failed to finalize billing transfer run", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

// refreshRunCounters recomputes every counter from the item rows and writes them
// back, reporting what it wrote plus whether a cancel has been asked for — the
// workflow needs both and this saves it a second round trip.
func (r *repository) refreshRunCounters(
	ctx context.Context,
	tx bun.Tx,
	tenant pagination.TenantInfo,
	runID pulid.ID,
	now int64,
	out *repositories.BillingTransferRunProgress,
) error {
	itemCols := buncolgen.BillingTransferRunItemColumns
	runCols := buncolgen.BillingTransferRunColumns

	var counts itemCounts
	err := tx.NewSelect().
		Model((*billingtransfer.BillingTransferRunItem)(nil)).
		ColumnExpr(
			buncolgen.CountFilter("processed", itemCols.Status.Ne()),
			billingtransfer.ItemStatusPending,
		).
		ColumnExpr(
			buncolgen.CountFilter("transferred", itemCols.Status.Eq()),
			billingtransfer.ItemStatusTransferred,
		).
		ColumnExpr(
			buncolgen.CountFilter("not_transferred", itemCols.Status.Eq()),
			billingtransfer.ItemStatusNotTransferred,
		).
		ColumnExpr(
			buncolgen.CountFilter("skipped", itemCols.Status.Eq()),
			billingtransfer.ItemStatusSkipped,
		).
		ColumnExpr(buncolgen.CountFilter("marked_ready", itemCols.MarkedReadyToInvoice.IsTrue())).
		ColumnExpr(
			buncolgen.CountFilter(
				"retryable",
				itemCols.Status.Eq(),
				itemCols.FailureCode.NotIn(),
			),
			billingtransfer.ItemStatusNotTransferred,
			bun.List(nonRetryableFailureCodes),
		).
		Apply(buncolgen.BillingTransferRunItemApplyTenant(tenant)).
		Where(itemCols.RunID.Eq(), runID).
		Scan(ctx, &counts)
	if err != nil {
		return err
	}

	run := new(billingtransfer.BillingTransferRun)
	result, err := tx.NewUpdate().
		Model(run).
		Set(runCols.ProcessedCount.Set(), counts.Processed).
		Set(runCols.TransferredCount.Set(), counts.Transferred).
		Set(runCols.NotTransferredCount.Set(), counts.NotTransferred).
		Set(runCols.SkippedCount.Set(), counts.Skipped).
		Set(runCols.MarkedReadyToInvoiceCount.Set(), counts.MarkedReady).
		Set(runCols.RetryableCount.Set(), counts.Retryable).
		Set(runCols.UpdatedAt.Set(), now).
		Set(runCols.Version.Inc(1)).
		Apply(func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.BillingTransferRunScopeTenantUpdate(uq, tenant)
		}).
		Where(runCols.ID.Eq(), runID).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return err
	}
	if err = dberror.CheckRowsAffected(result, "BillingTransferRun", runID.String()); err != nil {
		return err
	}

	if out != nil {
		out.Status = run.Status
		out.TotalCount = run.TotalCount
		out.ProcessedCount = run.ProcessedCount
		out.TransferredCount = run.TransferredCount
		out.NotTransferredCount = run.NotTransferredCount
		out.SkippedCount = run.SkippedCount
		out.MarkedReadyToInvoiceCount = run.MarkedReadyToInvoiceCount
		out.RetryableCount = run.RetryableCount
		out.CancelRequested = run.IsCancelRequested()
	}

	return nil
}

type itemCounts struct {
	Processed      int `bun:"processed"`
	Transferred    int `bun:"transferred"`
	NotTransferred int `bun:"not_transferred"`
	Skipped        int `bun:"skipped"`
	MarkedReady    int `bun:"marked_ready"`
	Retryable      int `bun:"retryable"`
}
