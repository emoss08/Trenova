package invoiceadjustmentrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment/invoiceadjustmentdb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const fieldIdempotencyKey = "idempotencyKey"

var adjustmentErrors = dberror.Mapping{
	Entity: "InvoiceAdjustment",
	Unique: []dberror.Constraint{
		{
			Name:    "uq_invoice_adjustments_idempotency",
			Field:   fieldIdempotencyKey,
			Message: "An invoice adjustment with this idempotency key already exists",
		},
	},
}

var adjustmentLineErrors = dberror.Mapping{Entity: "InvoiceAdjustmentLine"}

var snapshotErrors = dberror.Mapping{Entity: "InvoiceAdjustmentSnapshot"}

var exceptionErrors = dberror.Mapping{Entity: "InvoiceReconciliationException"}

var documentReferenceErrors = dberror.Mapping{
	Entity: "InvoiceAdjustmentDocumentReference",
	Unique: []dberror.Constraint{
		{
			Name:    "uq_invoice_adjustment_document_references",
			Field:   "documentIds",
			Message: "A document can be referenced by the adjustment only once",
		},
	},
}

var correctionGroupErrors = dberror.Mapping{Entity: "InvoiceCorrectionGroup"}

var batchErrors = dberror.Mapping{
	Entity: "InvoiceAdjustmentBatch",
	Unique: []dberror.Constraint{
		{
			Name:    "uq_invoice_adjustment_batches_idempotency",
			Field:   fieldIdempotencyKey,
			Message: "An adjustment batch with this idempotency key already exists",
		},
	},
}

var batchItemErrors = dberror.Mapping{
	Entity: "InvoiceAdjustmentBatchItem",
	Unique: []dberror.Constraint{
		{
			Name:    "uq_invoice_adjustment_batch_items_idempotency",
			Field:   fieldIdempotencyKey,
			Message: "An adjustment batch item with this idempotency key already exists",
		},
	},
}

var invoiceErrors = dberror.Mapping{Entity: "Invoice"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func mapInvoiceAdjustmentPersistenceError(err error) error {
	if !dberror.IsForeignKeyConstraintViolation(err) {
		return err
	}

	switch dberror.ExtractConstraintName(err) {
	case "fk_invoice_adjustments_correction_group":
		return errortypes.NewBusinessError(
			"Invoice adjustment can no longer be processed because its correction group is no longer valid. Refresh the invoice and try again.",
		).WithInternal(err)
	default:
		return err
	}
}

func New(p Params) invoiceadjustmentports.InvoiceAdjustmentRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.invoice-adjustment-repository"),
	}
}

func adjustmentDetail() *keel.Query[invoiceadjustment.InvoiceAdjustment] {
	ia := invoiceadjustmentdb.InvoiceAdjustments
	lines := ia.Lines()
	refs := ia.DocumentReferences()

	return ia.Load(
		lines.Order(lines.LineNumber.Asc()),
		ia.Snapshots(),
		ia.ReconciliationExceptions(),
		refs.Order(refs.CreatedAt.Asc()).Load(refs.Document()),
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req invoiceadjustmentports.GetInvoiceAdjustmentRequest,
) (*invoiceadjustment.InvoiceAdjustment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*invoiceadjustment.InvoiceAdjustment, error) {
			entity, err := adjustmentDetail().
				Where(invoiceadjustmentdb.InvoiceAdjustments.ID.Eq(req.ID)).
				One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, adjustmentErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) GetByIdempotencyKey(
	ctx context.Context,
	req invoiceadjustmentports.GetInvoiceAdjustmentByIdempotencyRequest,
) (*invoiceadjustment.InvoiceAdjustment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*invoiceadjustment.InvoiceAdjustment, error) {
			ia := invoiceadjustmentdb.InvoiceAdjustments
			entity, err := adjustmentDetail().
				Where(ia.IdempotencyKey.Eq(req.IdempotencyKey)).
				One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, adjustmentErrors.Map(err, "")
			}

			return entity, nil
		},
	)
}

func (r *repository) LockInvoiceForUpdate(
	ctx context.Context,
	req invoiceadjustmentports.LockInvoiceAdjustmentRequest,
) (*invoice.Invoice, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*invoice.Invoice, error) {
		db := r.db.Keel(ctx)
		inv := invoicedb.Invoices
		entity, err := inv.Where(inv.ID.Eq(req.InvoiceID)).Lock(keel.ForUpdate).One(ctx, db)
		if err != nil {
			return nil, invoiceErrors.Map(err, req.InvoiceID.String())
		}

		il := invoicedb.InvoiceLines
		entity.Lines, err = il.Where(il.InvoiceID.Eq(req.InvoiceID)).
			Order(il.LineNumber.Asc()).
			Lock(keel.ForUpdate).
			All(ctx, db)
		if err != nil {
			return nil, err
		}

		return entity, nil
	})
}

type lineCredit struct {
	OriginalLineID pulid.ID
	AmountCredited decimal.Decimal
}

func (r *repository) GetInvoiceLineCreditUsage(
	ctx context.Context,
	req invoiceadjustmentports.GetInvoiceLineCreditUsageRequest,
) (map[string]decimal.Decimal, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[string]decimal.Decimal, error) {
		rows, err := keel.Into[lineCredit](ctx, r.db.Keel(ctx), lineCreditUsage(&req))
		if err != nil {
			return nil, err
		}

		result := make(map[string]decimal.Decimal, len(rows))
		for _, row := range rows {
			result[row.OriginalLineID.String()] = row.AmountCredited
		}

		return result, nil
	})
}

func lineCreditUsage(
	req *invoiceadjustmentports.GetInvoiceLineCreditUsageRequest,
) *keel.Query[invoiceadjustment.InvoiceAdjustmentLine] {
	l := invoiceadjustmentdb.InvoiceAdjustmentLines
	ia := invoiceadjustmentdb.InvoiceAdjustments

	return l.Pick(l.OriginalLineID, keel.DynFunc("abs", l.CreditAmount).Sum().Coalesce(0)).
		Where(
			l.OriginalInvoiceID.Eq(req.InvoiceID),
			l.AdjustmentID.InQuery(ia.Pick(ia.ID).Where(ia.Status.In(
				invoiceadjustment.StatusApproved,
				invoiceadjustment.StatusExecuted,
			))),
			keel.When(req.ExcludeAdjustmentID.IsNotNil(), l.AdjustmentID.Ne(req.ExcludeAdjustmentID)),
		).
		Group(l.OriginalLineID)
}

func (r *repository) GetCorrectionGroup(
	ctx context.Context,
	req invoiceadjustmentports.GetCorrectionGroupRequest,
) (*invoiceadjustment.InvoiceAdjustmentCorrectionGroup, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*invoiceadjustment.InvoiceAdjustmentCorrectionGroup, error) {
			entity, err := invoiceadjustmentdb.InvoiceAdjustmentCorrectionGroups.Get(
				ctx,
				r.db.Keel(ctx),
				req.ID,
			)
			if err != nil {
				return nil, correctionGroupErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) GetCorrectionGroupByRootInvoice(
	ctx context.Context,
	req invoiceadjustmentports.GetCorrectionGroupByRootInvoiceRequest,
) (*invoiceadjustment.InvoiceAdjustmentCorrectionGroup, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*invoiceadjustment.InvoiceAdjustmentCorrectionGroup, error) {
			g := invoiceadjustmentdb.InvoiceAdjustmentCorrectionGroups
			entity, err := g.Where(g.RootInvoiceID.Eq(req.RootInvoiceID)).One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, correctionGroupErrors.Map(err, req.RootInvoiceID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) CreateCorrectionGroup(
	ctx context.Context,
	group *invoiceadjustment.InvoiceAdjustmentCorrectionGroup,
) (*invoiceadjustment.InvoiceAdjustmentCorrectionGroup, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*invoiceadjustment.InvoiceAdjustmentCorrectionGroup, error) {
			g := invoiceadjustmentdb.InvoiceAdjustmentCorrectionGroups
			if err := g.Create(ctx, r.db.Keel(ctx), group); err != nil {
				return nil, correctionGroupErrors.Map(err, group.ID.String())
			}

			return r.GetCorrectionGroup(ctx, invoiceadjustmentports.GetCorrectionGroupRequest{
				ID: group.ID,
			})
		},
	)
}

func (r *repository) UpdateCorrectionGroup(
	ctx context.Context,
	group *invoiceadjustment.InvoiceAdjustmentCorrectionGroup,
) (*invoiceadjustment.InvoiceAdjustmentCorrectionGroup, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*invoiceadjustment.InvoiceAdjustmentCorrectionGroup, error) {
			g := invoiceadjustmentdb.InvoiceAdjustmentCorrectionGroups
			err := g.Save(ctx, r.db.Keel(ctx), group, g.CurrentInvoiceID, g.Metadata)
			if err != nil {
				return nil, correctionGroupErrors.Map(err, group.ID.String())
			}

			return r.GetCorrectionGroup(ctx, invoiceadjustmentports.GetCorrectionGroupRequest{
				ID: group.ID,
			})
		},
	)
}

func (r *repository) CreateAdjustmentArtifacts(
	ctx context.Context,
	params invoiceadjustmentports.CreateAdjustmentArtifactsParams,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.Keel(ctx)
		if params.Adjustment != nil {
			params.Adjustment.SyncMinorAmounts()
			err := invoiceadjustmentdb.InvoiceAdjustments.Create(ctx, db, params.Adjustment)
			if err != nil {
				return adjustmentErrors.Map(
					mapInvoiceAdjustmentPersistenceError(err),
					params.Adjustment.ID.String(),
				)
			}
		}

		if len(params.Lines) > 0 {
			for _, line := range params.Lines {
				if line != nil {
					line.SyncMinorAmounts()
				}
			}
			if err := invoiceadjustmentdb.InvoiceAdjustmentLines.CreateMany(
				ctx,
				db,
				params.Lines,
			); err != nil {
				return adjustmentLineErrors.Map(err, "")
			}
		}

		if len(params.Snapshots) > 0 {
			if err := invoiceadjustmentdb.InvoiceAdjustmentSnapshots.CreateMany(
				ctx,
				db,
				params.Snapshots,
			); err != nil {
				return snapshotErrors.Map(err, "")
			}
		}

		if len(params.ReconciliationExceptions) > 0 {
			if err := invoiceadjustmentdb.InvoiceAdjustmentReconciliationExceptions.CreateMany(
				ctx,
				db,
				params.ReconciliationExceptions,
			); err != nil {
				return exceptionErrors.Map(err, "")
			}
		}

		if len(params.DocumentReferences) > 0 {
			if err := invoiceadjustmentdb.InvoiceAdjustmentDocumentReferences.CreateMany(
				ctx,
				db,
				params.DocumentReferences,
			); err != nil {
				return documentReferenceErrors.Map(err, "")
			}
		}

		return nil
	})
}

func (r *repository) UpdateAdjustment(
	ctx context.Context,
	adjustment *invoiceadjustment.InvoiceAdjustment,
) (*invoiceadjustment.InvoiceAdjustment, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*invoiceadjustment.InvoiceAdjustment, error) {
			adjustment.SyncMinorAmounts()

			ia := invoiceadjustmentdb.InvoiceAdjustments
			err := ia.Save(ctx, r.db.Keel(ctx), adjustment,
				ia.CorrectionGroupID,
				ia.CreditMemoInvoiceID,
				ia.ReplacementInvoiceID,
				ia.RebillQueueItemID,
				ia.Kind,
				ia.Status,
				ia.ApprovalStatus,
				ia.ReplacementReviewStatus,
				ia.RebillStrategy,
				ia.Reason,
				ia.PolicyReason,
				ia.AccountingDate,
				ia.CreditTotalAmount,
				ia.CreditTotalAmountMinor,
				ia.RebillTotalAmount,
				ia.RebillTotalAmountMinor,
				ia.NetDeltaAmount,
				ia.NetDeltaAmountMinor,
				ia.RerateVariancePercent,
				ia.WouldCreateUnappliedCredit,
				ia.RequiresReconciliationException,
				ia.ApprovalRequired,
				ia.SubmittedByID,
				ia.SubmittedAt,
				ia.ApprovedByID,
				ia.ApprovedAt,
				ia.RejectedByID,
				ia.RejectedAt,
				ia.RejectionReason,
				ia.ExecutionError,
				ia.IdempotencyKey,
				ia.Metadata,
			)
			if err != nil {
				return nil, adjustmentErrors.Map(
					mapInvoiceAdjustmentPersistenceError(err),
					adjustment.ID.String(),
				)
			}

			return r.GetByID(ctx, invoiceadjustmentports.GetInvoiceAdjustmentRequest{
				ID: adjustment.ID,
			})
		},
	)
}

func (r *repository) GetLineage(
	ctx context.Context,
	correctionGroupID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*invoiceadjustmentports.InvoiceLineageResult, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*invoiceadjustmentports.InvoiceLineageResult, error) {
			group, err := r.GetCorrectionGroup(
				ctx,
				invoiceadjustmentports.GetCorrectionGroupRequest{
					ID:         correctionGroupID,
					TenantInfo: tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}

			db := r.db.Keel(ctx)
			inv := invoicedb.Invoices
			lines := inv.Lines()
			invoices, err := inv.Where(inv.CorrectionGroupID.Eq(correctionGroupID)).
				Load(lines.Order(lines.LineNumber.Asc())).
				Order(inv.CreatedAt.Asc()).
				All(ctx, db)
			if err != nil {
				return nil, err
			}

			ia := invoiceadjustmentdb.InvoiceAdjustments
			adjustments, err := ia.Where(ia.CorrectionGroupID.Eq(correctionGroupID)).
				Order(ia.CreatedAt.Asc()).
				All(ctx, db)
			if err != nil {
				return nil, err
			}

			return &invoiceadjustmentports.InvoiceLineageResult{
				CorrectionGroup: group,
				Invoices:        nonNil(invoices),
				Adjustments:     nonNil(adjustments),
			}, nil
		},
	)
}

func (r *repository) ReplaceAdjustmentLines(
	ctx context.Context,
	req invoiceadjustmentports.ReplaceAdjustmentLinesRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.Keel(ctx)
		l := invoiceadjustmentdb.InvoiceAdjustmentLines
		if _, err := l.Where(l.AdjustmentID.Eq(req.AdjustmentID)).Delete(ctx, db); err != nil {
			return adjustmentLineErrors.Map(err, req.AdjustmentID.String())
		}

		if len(req.Lines) == 0 {
			return nil
		}

		for _, line := range req.Lines {
			if line != nil {
				line.SyncMinorAmounts()
			}
		}

		return adjustmentLineErrors.Map(l.CreateMany(ctx, db, req.Lines), req.AdjustmentID.String())
	})
}

func (r *repository) ReplaceDocumentReferences(
	ctx context.Context,
	req invoiceadjustmentports.ReplaceDocumentReferencesRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.Keel(ctx)
		refs := invoiceadjustmentdb.InvoiceAdjustmentDocumentReferences
		if _, err := refs.Where(refs.AdjustmentID.Eq(req.AdjustmentID)).
			Delete(ctx, db); err != nil {
			return documentReferenceErrors.Map(err, req.AdjustmentID.String())
		}

		if len(req.References) == 0 {
			return nil
		}

		return documentReferenceErrors.Map(
			refs.CreateMany(ctx, db, req.References),
			req.AdjustmentID.String(),
		)
	})
}

func (r *repository) CreateBatch(
	ctx context.Context,
	batch *invoiceadjustment.InvoiceAdjustmentBatch,
	items []*invoiceadjustment.InvoiceAdjustmentBatchItem,
) (*invoiceadjustment.InvoiceAdjustmentBatch, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*invoiceadjustment.InvoiceAdjustmentBatch, error) {
			db := r.db.Keel(ctx)
			if err := invoiceadjustmentdb.InvoiceAdjustmentBatches.Create(
				ctx,
				db,
				batch,
			); err != nil {
				return nil, batchErrors.Map(err, batch.ID.String())
			}
			if len(items) > 0 {
				err := invoiceadjustmentdb.InvoiceAdjustmentBatchItems.CreateMany(ctx, db, items)
				if err != nil {
					return nil, batchItemErrors.Map(err, batch.ID.String())
				}
			}

			return r.GetBatchByID(ctx, invoiceadjustmentports.GetBatchRequest{ID: batch.ID})
		},
	)
}

func batchDetail() *keel.Query[invoiceadjustment.InvoiceAdjustmentBatch] {
	b := invoiceadjustmentdb.InvoiceAdjustmentBatches
	return b.Load(b.Items())
}

func (r *repository) GetBatchByID(
	ctx context.Context,
	req invoiceadjustmentports.GetBatchRequest,
) (*invoiceadjustment.InvoiceAdjustmentBatch, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*invoiceadjustment.InvoiceAdjustmentBatch, error) {
			entity, err := batchDetail().
				Where(invoiceadjustmentdb.InvoiceAdjustmentBatches.ID.Eq(req.ID)).
				One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, batchErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) GetBatchByIdempotencyKey(
	ctx context.Context,
	req invoiceadjustmentports.GetBatchByIdempotencyRequest,
) (*invoiceadjustment.InvoiceAdjustmentBatch, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*invoiceadjustment.InvoiceAdjustmentBatch, error) {
			b := invoiceadjustmentdb.InvoiceAdjustmentBatches
			entity, err := batchDetail().
				Where(b.IdempotencyKey.Eq(req.IdempotencyKey)).
				One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, batchErrors.Map(err, "")
			}

			return entity, nil
		},
	)
}

func (r *repository) UpdateBatch(
	ctx context.Context,
	batch *invoiceadjustment.InvoiceAdjustmentBatch,
) (*invoiceadjustment.InvoiceAdjustmentBatch, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*invoiceadjustment.InvoiceAdjustmentBatch, error) {
			b := invoiceadjustmentdb.InvoiceAdjustmentBatches
			err := b.Save(ctx, r.db.Keel(ctx), batch,
				b.Status,
				b.TotalCount,
				b.ProcessedCount,
				b.SucceededCount,
				b.FailedCount,
				b.SubmittedByID,
				b.SubmittedAt,
				b.Metadata,
			)
			if err != nil {
				return nil, batchErrors.Map(err, batch.ID.String())
			}

			return r.GetBatchByID(ctx, invoiceadjustmentports.GetBatchRequest{ID: batch.ID})
		},
	)
}

func (r *repository) UpdateBatchItem(
	ctx context.Context,
	item *invoiceadjustment.InvoiceAdjustmentBatchItem,
) (*invoiceadjustment.InvoiceAdjustmentBatchItem, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*invoiceadjustment.InvoiceAdjustmentBatchItem, error) {
			db := r.db.Keel(ctx)
			bi := invoiceadjustmentdb.InvoiceAdjustmentBatchItems
			err := bi.Save(ctx, db, item,
				bi.AdjustmentID,
				bi.Status,
				bi.ErrorMessage,
				bi.RequestPayload,
				bi.ResultPayload,
			)
			if err != nil {
				return nil, batchItemErrors.Map(err, item.ID.String())
			}

			entity, err := bi.Get(ctx, db, item.ID)
			if err != nil {
				return nil, batchItemErrors.Map(err, item.ID.String())
			}

			return entity, nil
		},
	)
}

func nonNil[T any](rows []*T) []*T {
	if rows == nil {
		return []*T{}
	}

	return rows
}
