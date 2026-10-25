package billingtransferrepository

import (
	"context"

	"github.com/bytedance/sonic"
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingtransfer"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingtransfer/billingtransferdb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingtransferports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

var nonRetryableFailureCodes = []billingtransfer.FailureCode{
	billingtransfer.FailureNotFound,
	billingtransfer.FailureAlreadyTransferred,
}

func (r *repository) SeedItems(
	ctx context.Context,
	req *billingtransferports.SeedBillingTransferRunItemsRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
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

		inserted, err := billingtransferdb.BillingTransferRunItems.CreateOrSkip(
			ctx,
			r.db.Keel(ctx),
			rows,
		)
		if err != nil {
			r.l.Error("failed to seed billing transfer run items", zap.Error(err))
			return 0, itemErrors.Map(err, req.RunID.String())
		}

		return int(inserted), nil
	})
}

func (r *repository) SeedRetryItems(
	ctx context.Context,
	req *billingtransferports.SeedBillingTransferRetryItemsRequest,
) (*billingtransferports.SeedBillingTransferRetryItemsResult, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*billingtransferports.SeedBillingTransferRetryItemsResult, error) {
			db := r.db.Keel(ctx)
			item := billingtransferdb.BillingTransferRunItems
			source, err := item.Where(item.RunID.Eq(req.SourceRunID)).
				Order(item.Sequence.Asc()).
				All(ctx, db)
			if err != nil {
				r.l.Error("failed to read source run items for retry", zap.Error(err))
				return nil, err
			}

			out := &billingtransferports.SeedBillingTransferRetryItemsResult{}
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

			if _, err = item.CreateOrSkip(ctx, db, rows); err != nil {
				r.l.Error("failed to seed retry billing transfer run items", zap.Error(err))
				return nil, itemErrors.Map(err, req.RunID.String())
			}

			return out, nil
		},
	)
}

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

func (r *repository) NextPendingShipmentIDs(
	ctx context.Context,
	req *billingtransferports.NextPendingBillingTransferItemsRequest,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		if req.Limit <= 0 {
			return []pulid.ID{}, nil
		}

		item := billingtransferdb.BillingTransferRunItems
		ids, err := keel.Into[pulid.ID](ctx, r.db.Keel(ctx), item.
			Pick(item.ShipmentID).
			Where(item.RunID.Eq(req.RunID), item.Status.Eq(billingtransfer.ItemStatusPending)).
			Order(item.Sequence.Asc()).
			Limit(req.Limit))
		if err != nil {
			r.l.Error("failed to claim next billing transfer batch", zap.Error(err))
			return nil, err
		}
		if ids == nil {
			return []pulid.ID{}, nil
		}

		return ids, nil
	})
}

func outcomeUpdate(
	runID pulid.ID,
	outcomes []billingtransferports.BillingTransferItemOutcome,
	now int64,
) (*keel.Query[billingtransfer.BillingTransferRunItem], []keel.Assignment, error) {
	n := len(outcomes)
	shipments := make([]pulid.ID, n)
	proNumbers := make([]string, n)
	statuses := make([]string, n)
	failureCodes := make([]string, n)
	errorMessages := make([]string, n)
	markedReady := make([]bool, n)
	queueItems := make([]pulid.ID, n)
	queueNumbers := make([]string, n)
	queueStatuses := make([]string, n)
	missing := make([]string, n)
	failures := make([]string, n)
	processed := make([]int64, n)
	for i := range outcomes {
		outcome := &outcomes[i]
		requirements := outcome.MissingRequirements
		if requirements == nil {
			requirements = []billingtransfer.MissingRequirement{}
		}
		encodedRequirements, err := sonic.MarshalString(requirements)
		if err != nil {
			return nil, nil, err
		}
		validation := outcome.ValidationFailures
		if validation == nil {
			validation = []billingtransfer.ValidationFailure{}
		}
		encodedValidation, err := sonic.MarshalString(validation)
		if err != nil {
			return nil, nil, err
		}

		processedAt := outcome.ProcessedAt
		if processedAt == 0 {
			processedAt = now
		}

		shipments[i] = outcome.ShipmentID
		proNumbers[i] = outcome.ProNumber
		statuses[i] = string(outcome.Status)
		failureCodes[i] = string(outcome.FailureCode)
		errorMessages[i] = outcome.ErrorMessage
		markedReady[i] = outcome.MarkedReadyToInvoice
		queueItems[i] = outcome.BillingQueueItemID
		queueNumbers[i] = outcome.BillingQueueNumber
		queueStatuses[i] = string(outcome.BillingQueueStatus)
		missing[i] = encodedRequirements
		failures[i] = encodedValidation
		processed[i] = processedAt
	}

	item := billingtransferdb.BillingTransferRunItems
	outcome := keel.Unnest("outcome", []string{
		"shipment_id", "pro_number", "status", "failure_code", "error_message",
		"marked_ready_to_invoice", "billing_queue_item_id", "billing_queue_number",
		"billing_queue_status", "missing_requirements", "validation_failures", "processed_at",
	}, shipments, proNumbers, statuses, failureCodes, errorMessages, markedReady,
		queueItems, queueNumbers, queueStatuses, missing, failures, processed)
	orNull := func(column string) keel.Expr {
		return keel.DynOf(outcome.Col(column)).NullIf("")
	}

	q := item.With(outcome).
		Join(outcome, outcome.Col("shipment_id").EqCol(item.ShipmentID)).
		Where(item.RunID.Eq(runID))

	return q, []keel.Assignment{
		item.Status.ToExpr(outcome.Col("status")),
		item.ProNumber.ToExpr(orNull("pro_number")),
		item.FailureCode.ToExpr(orNull("failure_code")),
		item.ErrorMessage.ToExpr(orNull("error_message")),
		item.MarkedReadyToInvoice.ToExpr(outcome.Col("marked_ready_to_invoice")),
		item.BillingQueueItemID.ToExpr(orNull("billing_queue_item_id")),
		item.BillingQueueNumber.ToExpr(orNull("billing_queue_number")),
		item.BillingQueueStatus.ToExpr(orNull("billing_queue_status")),
		item.MissingRequirements.ToExpr(keel.DynOf(outcome.Col("missing_requirements")).Cast(keel.TypeJSON)),
		item.ValidationFailures.ToExpr(keel.DynOf(outcome.Col("validation_failures")).Cast(keel.TypeJSON)),
		item.ProcessedAt.ToExpr(outcome.Col("processed_at")),
	}, nil
}

func (r *repository) RecordItemOutcomes(
	ctx context.Context,
	req *billingtransferports.RecordBillingTransferOutcomesRequest,
) (*billingtransferports.BillingTransferRunProgress, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*billingtransferports.BillingTransferRunProgress, error) {
			if len(req.Outcomes) > 0 {
				q, sets, err := outcomeUpdate(req.RunID, req.Outcomes, timeutils.NowUnix())
				if err != nil {
					r.l.Error("failed to encode billing transfer outcomes", zap.Error(err))
					return nil, err
				}

				if _, err = q.Update(ctx, r.db.Keel(ctx), sets...); err != nil {
					r.l.Error("failed to record billing transfer outcomes", zap.Error(err))
					return nil, itemErrors.Map(err, req.RunID.String())
				}
			}

			progress := new(billingtransferports.BillingTransferRunProgress)
			if err := r.refreshRunCounters(ctx, req.RunID, progress); err != nil {
				r.l.Error("failed to record billing transfer outcomes", zap.Error(err))
				return nil, err
			}

			return progress, nil
		},
	)
}

func (r *repository) Finalize(
	ctx context.Context,
	req *billingtransferports.FinalizeBillingTransferRunRequest,
) (*billingtransfer.BillingTransferRun, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*billingtransfer.BillingTransferRun, error) {
			db := r.db.Keel(ctx)
			now := timeutils.NowUnix()
			item := billingtransferdb.BillingTransferRunItems

			if _, err := item.Where(
				item.RunID.Eq(req.RunID),
				item.Status.Eq(billingtransfer.ItemStatusPending),
			).Update(ctx, db,
				item.Status.To(billingtransfer.ItemStatusSkipped),
				item.ProcessedAt.To(now),
			); err != nil {
				r.l.Error("failed to finalize billing transfer run", zap.Error(err))
				return nil, itemErrors.Map(err, req.RunID.String())
			}

			if err := r.refreshRunCounters(ctx, req.RunID, nil); err != nil {
				r.l.Error("failed to finalize billing transfer run", zap.Error(err))
				return nil, err
			}

			run := billingtransferdb.BillingTransferRuns
			rows, err := run.Where(run.ID.Eq(req.RunID)).UpdateReturning(ctx, db,
				run.Status.To(req.Status),
				run.FailureMessage.ToExpr(keel.Val(stringutils.NilIfEmpty(req.FailureMessage))),
				run.CompletedAt.ToExpr(keel.DynOf(run.CompletedAt).Coalesce(now)),
			)
			if err != nil {
				r.l.Error("failed to finalize billing transfer run", zap.Error(err))
				return nil, runErrors.Map(err, req.RunID.String())
			}
			if err = dberror.CheckAffected(
				int64(len(rows)),
				"BillingTransferRun",
				req.RunID.String(),
			); err != nil {
				return nil, err
			}

			return rows[0], nil
		},
	)
}

type itemCounts struct {
	Processed      int
	Transferred    int
	NotTransferred int
	Skipped        int
	MarkedReady    int
	Retryable      int
}

func runItemCounts(runID pulid.ID) *keel.Query[billingtransfer.BillingTransferRunItem] {
	item := billingtransferdb.BillingTransferRunItems
	return item.Pick(
		keel.Count().Filter(item.Status.Ne(billingtransfer.ItemStatusPending)),
		keel.Count().Filter(item.Status.Eq(billingtransfer.ItemStatusTransferred)),
		keel.Count().Filter(item.Status.Eq(billingtransfer.ItemStatusNotTransferred)),
		keel.Count().Filter(item.Status.Eq(billingtransfer.ItemStatusSkipped)),
		keel.Count().Filter(item.MarkedReadyToInvoice.IsTrue()),
		keel.Count().Filter(keel.And(
			item.Status.Eq(billingtransfer.ItemStatusNotTransferred),
			item.FailureCode.NotIn(nonRetryableFailureCodes...),
		)),
	).Where(item.RunID.Eq(runID))
}

func (r *repository) refreshRunCounters(
	ctx context.Context,
	runID pulid.ID,
	out *billingtransferports.BillingTransferRunProgress,
) error {
	db := r.db.Keel(ctx)
	counts, err := keel.IntoOne[itemCounts](ctx, db, runItemCounts(runID))
	if err != nil {
		return err
	}

	run := billingtransferdb.BillingTransferRuns
	rows, err := run.Where(run.ID.Eq(runID)).UpdateReturning(ctx, db,
		run.ProcessedCount.To(int64(counts.Processed)),
		run.TransferredCount.To(int64(counts.Transferred)),
		run.NotTransferredCount.To(int64(counts.NotTransferred)),
		run.SkippedCount.To(int64(counts.Skipped)),
		run.MarkedReadyToInvoiceCount.To(int64(counts.MarkedReady)),
		run.RetryableCount.To(int64(counts.Retryable)),
	)
	if err != nil {
		return runErrors.Map(err, runID.String())
	}
	if err = dberror.CheckAffected(
		int64(len(rows)),
		"BillingTransferRun",
		runID.String(),
	); err != nil {
		return err
	}

	updated := rows[0]
	if out != nil {
		out.Status = updated.Status
		out.TotalCount = updated.TotalCount
		out.ProcessedCount = updated.ProcessedCount
		out.TransferredCount = updated.TransferredCount
		out.NotTransferredCount = updated.NotTransferredCount
		out.SkippedCount = updated.SkippedCount
		out.MarkedReadyToInvoiceCount = updated.MarkedReadyToInvoiceCount
		out.RetryableCount = updated.RetryableCount
		out.CancelRequested = updated.IsCancelRequested()
	}

	return nil
}
