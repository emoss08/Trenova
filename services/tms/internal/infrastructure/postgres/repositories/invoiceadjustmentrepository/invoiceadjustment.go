package invoiceadjustmentrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
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

func New(p Params) repositories.InvoiceAdjustmentRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.invoice-adjustment-repository"),
	}
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetInvoiceAdjustmentRequest,
) (*invoiceadjustment.Adjustment, error) {
	entity := new(invoiceadjustment.Adjustment)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("ia.id = ?", req.ID).
		Where("ia.organization_id = ?", req.TenantInfo.OrgID).
		Where("ia.business_unit_id = ?", req.TenantInfo.BuID).
		Relation("Lines", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order("ial.line_number ASC")
		}).
		Relation("Snapshots").
		Relation("ReconciliationExceptions").
		Relation("DocumentReferences", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Relation("Document").Order("iadr.created_at ASC")
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "InvoiceAdjustment")
	}

	return entity, nil
}

func (r *repository) GetByIdempotencyKey(
	ctx context.Context,
	req repositories.GetInvoiceAdjustmentByIdempotencyRequest,
) (*invoiceadjustment.Adjustment, error) {
	entity := new(invoiceadjustment.Adjustment)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("ia.idempotency_key = ?", req.IdempotencyKey).
		Where("ia.organization_id = ?", req.TenantInfo.OrgID).
		Where("ia.business_unit_id = ?", req.TenantInfo.BuID).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "InvoiceAdjustment")
	}

	return r.GetByID(ctx, repositories.GetInvoiceAdjustmentRequest{
		ID:         entity.ID,
		TenantInfo: req.TenantInfo,
	})
}

func (r *repository) LockInvoiceForUpdate(
	ctx context.Context,
	req repositories.LockInvoiceAdjustmentRequest,
) (*invoice.Invoice, error) {
	entity := new(invoice.Invoice)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("inv.id = ?", req.InvoiceID).
		Where("inv.organization_id = ?", req.TenantInfo.OrgID).
		Where("inv.business_unit_id = ?", req.TenantInfo.BuID).
		Relation("Lines", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order("invl.line_number ASC").For("UPDATE")
		}).
		For("UPDATE").
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Invoice")
	}

	return entity, nil
}

func (r *repository) GetInvoiceLineCreditUsage(
	ctx context.Context,
	req repositories.GetInvoiceLineCreditUsageRequest,
) (map[string]decimal.Decimal, error) {
	type row struct {
		OriginalLineID string          `bun:"original_line_id"`
		AmountCredited decimal.Decimal `bun:"amount_credited"`
	}

	rows := make([]row, 0)
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model((*invoiceadjustment.AdjustmentLine)(nil)).
		Column("original_line_id").
		ColumnExpr("COALESCE(SUM(ABS(credit_amount)), 0) AS amount_credited").
		Where("original_invoice_id = ?", req.InvoiceID).
		Where("organization_id = ?", req.TenantInfo.OrgID).
		Where("business_unit_id = ?", req.TenantInfo.BuID).
		Where("adjustment_id IN (SELECT id FROM invoice_adjustments WHERE status IN (?) AND organization_id = ? AND business_unit_id = ?)",
			bun.In([]invoiceadjustment.Status{invoiceadjustment.StatusApproved, invoiceadjustment.StatusExecuted}),
			req.TenantInfo.OrgID,
			req.TenantInfo.BuID,
		)
	if req.ExcludeAdjustmentID.IsNotNil() {
		query = query.Where("adjustment_id != ?", req.ExcludeAdjustmentID)
	}
	err := query.Group("original_line_id").Scan(ctx, &rows)
	if err != nil {
		return nil, fmt.Errorf("get line credit usage: %w", err)
	}

	result := make(map[string]decimal.Decimal, len(rows))
	for _, item := range rows {
		result[item.OriginalLineID] = item.AmountCredited
	}

	return result, nil
}

func (r *repository) GetCorrectionGroup(
	ctx context.Context,
	req repositories.GetCorrectionGroupRequest,
) (*invoiceadjustment.CorrectionGroup, error) {
	entity := new(invoiceadjustment.CorrectionGroup)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("icg.id = ?", req.ID).
		Where("icg.organization_id = ?", req.TenantInfo.OrgID).
		Where("icg.business_unit_id = ?", req.TenantInfo.BuID).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "InvoiceCorrectionGroup")
	}

	return entity, nil
}

func (r *repository) GetCorrectionGroupByRootInvoice(
	ctx context.Context,
	req repositories.GetCorrectionGroupByRootInvoiceRequest,
) (*invoiceadjustment.CorrectionGroup, error) {
	entity := new(invoiceadjustment.CorrectionGroup)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("icg.root_invoice_id = ?", req.RootInvoiceID).
		Where("icg.organization_id = ?", req.TenantInfo.OrgID).
		Where("icg.business_unit_id = ?", req.TenantInfo.BuID).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "InvoiceCorrectionGroup")
	}

	return entity, nil
}

func (r *repository) CreateCorrectionGroup(
	ctx context.Context,
	group *invoiceadjustment.CorrectionGroup,
) (*invoiceadjustment.CorrectionGroup, error) {
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(group).Exec(ctx); err != nil {
		return nil, fmt.Errorf("create correction group: %w", err)
	}

	return r.GetCorrectionGroup(ctx, repositories.GetCorrectionGroupRequest{
		ID: group.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: group.OrganizationID,
			BuID:  group.BusinessUnitID,
		},
	})
}

func (r *repository) UpdateCorrectionGroup(
	ctx context.Context,
	group *invoiceadjustment.CorrectionGroup,
) (*invoiceadjustment.CorrectionGroup, error) {
	if _, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(group).
		Where("id = ?", group.ID).
		Where("organization_id = ?", group.OrganizationID).
		Where("business_unit_id = ?", group.BusinessUnitID).
		Column("current_invoice_id", "metadata", "updated_at").
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("update correction group: %w", err)
	}

	return r.GetCorrectionGroup(ctx, repositories.GetCorrectionGroupRequest{
		ID: group.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: group.OrganizationID,
			BuID:  group.BusinessUnitID,
		},
	})
}

func (r *repository) CreateAdjustmentArtifacts(
	ctx context.Context,
	params repositories.CreateAdjustmentArtifactsParams,
) error {
	if params.Adjustment != nil {
		params.Adjustment.SyncMinorAmounts()
		if _, err := r.db.DBForContext(ctx).NewInsert().Model(params.Adjustment).Exec(ctx); err != nil {
			return fmt.Errorf("create adjustment: %w", err)
		}
	}

	if len(params.Lines) > 0 {
		for _, line := range params.Lines {
			if line == nil {
				continue
			}

			line.SyncMinorAmounts()
		}

		if _, err := r.db.DBForContext(ctx).NewInsert().Model(&params.Lines).Exec(ctx); err != nil {
			return fmt.Errorf("create adjustment lines: %w", err)
		}
	}

	if len(params.Snapshots) > 0 {
		if _, err := r.db.DBForContext(ctx).NewInsert().Model(&params.Snapshots).Exec(ctx); err != nil {
			return fmt.Errorf("create adjustment snapshots: %w", err)
		}
	}

	if len(params.ReconciliationExceptions) > 0 {
		if _, err := r.db.DBForContext(ctx).NewInsert().Model(&params.ReconciliationExceptions).Exec(ctx); err != nil {
			return fmt.Errorf("create reconciliation exceptions: %w", err)
		}
	}

	if len(params.DocumentReferences) > 0 {
		if _, err := r.db.DBForContext(ctx).NewInsert().Model(&params.DocumentReferences).Exec(ctx); err != nil {
			return fmt.Errorf("create document references: %w", err)
		}
	}

	return nil
}

func (r *repository) UpdateAdjustment(
	ctx context.Context,
	adjustment *invoiceadjustment.Adjustment,
) (*invoiceadjustment.Adjustment, error) {
	adjustment.SyncMinorAmounts()

	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(adjustment).
		Where("id = ?", adjustment.ID).
		Where("organization_id = ?", adjustment.OrganizationID).
		Where("business_unit_id = ?", adjustment.BusinessUnitID).
		Where("version = ?", adjustment.Version).
		Set("correction_group_id = ?", adjustment.CorrectionGroupID).
		Set("credit_memo_invoice_id = ?", adjustment.CreditMemoInvoiceID).
		Set("replacement_invoice_id = ?", adjustment.ReplacementInvoiceID).
		Set("rebill_queue_item_id = ?", adjustment.RebillQueueItemID).
		Set("kind = ?", adjustment.Kind).
		Set("status = ?", adjustment.Status).
		Set("approval_status = ?", adjustment.ApprovalStatus).
		Set("replacement_review_status = ?", adjustment.ReplacementReviewStatus).
		Set("rebill_strategy = ?", adjustment.RebillStrategy).
		Set("reason = ?", adjustment.Reason).
		Set("policy_reason = ?", adjustment.PolicyReason).
		Set("accounting_date = ?", adjustment.AccountingDate).
		Set("credit_total_amount = ?", adjustment.CreditTotalAmount).
		Set("credit_total_amount_minor = ?", adjustment.CreditTotalAmountMinor).
		Set("rebill_total_amount = ?", adjustment.RebillTotalAmount).
		Set("rebill_total_amount_minor = ?", adjustment.RebillTotalAmountMinor).
		Set("net_delta_amount = ?", adjustment.NetDeltaAmount).
		Set("net_delta_amount_minor = ?", adjustment.NetDeltaAmountMinor).
		Set("rerate_variance_percent = ?", adjustment.RerateVariancePercent).
		Set("would_create_unapplied_credit = ?", adjustment.WouldCreateUnappliedCredit).
		Set("requires_reconciliation_exception = ?", adjustment.RequiresReconciliationException).
		Set("approval_required = ?", adjustment.ApprovalRequired).
		Set("submitted_by_id = ?", adjustment.SubmittedByID).
		Set("submitted_at = ?", adjustment.SubmittedAt).
		Set("approved_by_id = ?", adjustment.ApprovedByID).
		Set("approved_at = ?", adjustment.ApprovedAt).
		Set("rejected_by_id = ?", adjustment.RejectedByID).
		Set("rejected_at = ?", adjustment.RejectedAt).
		Set("rejection_reason = ?", adjustment.RejectionReason).
		Set("execution_error = ?", adjustment.ExecutionError).
		Set("metadata = ?", adjustment.Metadata).
		Set("version = version + 1").
		Exec(ctx)
	if err != nil {
		return nil, mapInvoiceAdjustmentPersistenceError(fmt.Errorf("update adjustment: %w", err))
	}
	if err = dberror.CheckRowsAffected(res, "InvoiceAdjustment", adjustment.ID.String()); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, repositories.GetInvoiceAdjustmentRequest{
		ID: adjustment.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: adjustment.OrganizationID,
			BuID:  adjustment.BusinessUnitID,
		},
	})
}

func (r *repository) GetLineage(
	ctx context.Context,
	correctionGroupID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*repositories.InvoiceLineageResult, error) {
	group, err := r.GetCorrectionGroup(ctx, repositories.GetCorrectionGroupRequest{
		ID:         correctionGroupID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	invoices := make([]*invoice.Invoice, 0)
	if err = r.db.DBForContext(ctx).
		NewSelect().
		Model(&invoices).
		Where("correction_group_id = ?", correctionGroupID).
		Where("organization_id = ?", tenantInfo.OrgID).
		Where("business_unit_id = ?", tenantInfo.BuID).
		Order("created_at ASC").
		Relation("Lines", func(q *bun.SelectQuery) *bun.SelectQuery { return q.Order("invl.line_number ASC") }).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("get lineage invoices: %w", err)
	}

	adjustments := make([]*invoiceadjustment.Adjustment, 0)
	if err = r.db.DBForContext(ctx).
		NewSelect().
		Model(&adjustments).
		Where("correction_group_id = ?", correctionGroupID).
		Where("organization_id = ?", tenantInfo.OrgID).
		Where("business_unit_id = ?", tenantInfo.BuID).
		Order("created_at ASC").
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("get lineage adjustments: %w", err)
	}

	return &repositories.InvoiceLineageResult{
		CorrectionGroup: group,
		Invoices:        invoices,
		Adjustments:     adjustments,
	}, nil
}

func (r *repository) ReplaceAdjustmentLines(
	ctx context.Context,
	req repositories.ReplaceAdjustmentLinesRequest,
) error {
	if _, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*invoiceadjustment.AdjustmentLine)(nil)).
		Where("adjustment_id = ?", req.AdjustmentID).
		Where("organization_id = ?", req.TenantInfo.OrgID).
		Where("business_unit_id = ?", req.TenantInfo.BuID).
		Exec(ctx); err != nil {
		return fmt.Errorf("delete adjustment lines: %w", err)
	}

	if len(req.Lines) == 0 {
		return nil
	}

	for _, line := range req.Lines {
		if line == nil {
			continue
		}

		line.SyncMinorAmounts()
	}

	if _, err := r.db.DBForContext(ctx).NewInsert().Model(&req.Lines).Exec(ctx); err != nil {
		return fmt.Errorf("insert adjustment lines: %w", err)
	}

	return nil
}

func (r *repository) ReplaceDocumentReferences(
	ctx context.Context,
	req repositories.ReplaceDocumentReferencesRequest,
) error {
	if _, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*invoiceadjustment.DocumentReference)(nil)).
		Where("adjustment_id = ?", req.AdjustmentID).
		Where("organization_id = ?", req.TenantInfo.OrgID).
		Where("business_unit_id = ?", req.TenantInfo.BuID).
		Exec(ctx); err != nil {
		return fmt.Errorf("delete document references: %w", err)
	}

	if len(req.References) == 0 {
		return nil
	}

	if _, err := r.db.DBForContext(ctx).NewInsert().Model(&req.References).Exec(ctx); err != nil {
		return fmt.Errorf("insert document references: %w", err)
	}

	return nil
}

func (r *repository) CreateBatch(
	ctx context.Context,
	batch *invoiceadjustment.Batch,
	items []*invoiceadjustment.BatchItem,
) (*invoiceadjustment.Batch, error) {
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(batch).Exec(ctx); err != nil {
		return nil, fmt.Errorf("create batch: %w", err)
	}
	if len(items) > 0 {
		if _, err := r.db.DBForContext(ctx).NewInsert().Model(&items).Exec(ctx); err != nil {
			return nil, fmt.Errorf("create batch items: %w", err)
		}
	}

	return r.GetBatchByID(ctx, repositories.GetBatchRequest{
		ID: batch.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: batch.OrganizationID,
			BuID:  batch.BusinessUnitID,
		},
	})
}

func (r *repository) GetBatchByID(
	ctx context.Context,
	req repositories.GetBatchRequest,
) (*invoiceadjustment.Batch, error) {
	entity := new(invoiceadjustment.Batch)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("iab.id = ?", req.ID).
		Where("iab.organization_id = ?", req.TenantInfo.OrgID).
		Where("iab.business_unit_id = ?", req.TenantInfo.BuID).
		Relation("Items").
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "InvoiceAdjustmentBatch")
	}
	return entity, nil
}

func (r *repository) GetBatchByIdempotencyKey(
	ctx context.Context,
	req repositories.GetBatchByIdempotencyRequest,
) (*invoiceadjustment.Batch, error) {
	entity := new(invoiceadjustment.Batch)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("iab.idempotency_key = ?", req.IdempotencyKey).
		Where("iab.organization_id = ?", req.TenantInfo.OrgID).
		Where("iab.business_unit_id = ?", req.TenantInfo.BuID).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "InvoiceAdjustmentBatch")
	}

	return r.GetBatchByID(ctx, repositories.GetBatchRequest{
		ID:         entity.ID,
		TenantInfo: req.TenantInfo,
	})
}

func (r *repository) UpdateBatch(
	ctx context.Context,
	batch *invoiceadjustment.Batch,
) (*invoiceadjustment.Batch, error) {
	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(batch).
		Where("id = ?", batch.ID).
		Where("organization_id = ?", batch.OrganizationID).
		Where("business_unit_id = ?", batch.BusinessUnitID).
		Where("version = ?", batch.Version).
		Set("status = ?", batch.Status).
		Set("total_count = ?", batch.TotalCount).
		Set("processed_count = ?", batch.ProcessedCount).
		Set("succeeded_count = ?", batch.SucceededCount).
		Set("failed_count = ?", batch.FailedCount).
		Set("submitted_by_id = ?", batch.SubmittedByID).
		Set("submitted_at = ?", batch.SubmittedAt).
		Set("metadata = ?", batch.Metadata).
		Set("version = version + 1").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update batch: %w", err)
	}
	if err = dberror.CheckRowsAffected(res, "InvoiceAdjustmentBatch", batch.ID.String()); err != nil {
		return nil, err
	}

	return r.GetBatchByID(ctx, repositories.GetBatchRequest{
		ID: batch.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: batch.OrganizationID,
			BuID:  batch.BusinessUnitID,
		},
	})
}

func (r *repository) UpdateBatchItem(
	ctx context.Context,
	item *invoiceadjustment.BatchItem,
) (*invoiceadjustment.BatchItem, error) {
	_, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(item).
		Where("id = ?", item.ID).
		Where("organization_id = ?", item.OrganizationID).
		Where("business_unit_id = ?", item.BusinessUnitID).
		Set("adjustment_id = ?", item.AdjustmentID).
		Set("status = ?", item.Status).
		Set("error_message = ?", item.ErrorMessage).
		Set("request_payload = ?", item.RequestPayload).
		Set("result_payload = ?", item.ResultPayload).
		Set("updated_at = extract(epoch from current_timestamp)::bigint").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update batch item: %w", err)
	}

	entity := new(invoiceadjustment.BatchItem)
	if err = r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("id = ?", item.ID).
		Where("organization_id = ?", item.OrganizationID).
		Where("business_unit_id = ?", item.BusinessUnitID).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "InvoiceAdjustmentBatchItem")
	}

	return entity, nil
}

func (r *repository) ListApprovalQueue(
	ctx context.Context,
	req repositories.ListApprovalQueueRequest,
) (*pagination.ListResult[*invoiceadjustment.ApprovalQueueItem], error) {
	entities := make([]*invoiceadjustment.ApprovalQueueItem, 0, req.Filter.Pagination.SafeLimit())
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		ModelTableExpr("invoice_adjustments AS ia").
		ColumnExpr("ia.id AS adjustment_id").
		ColumnExpr("ia.correction_group_id").
		ColumnExpr("ia.original_invoice_id").
		ColumnExpr("orig.number AS original_invoice_number").
		ColumnExpr("orig.status AS original_invoice_status").
		ColumnExpr("orig.bill_to_name AS customer_name").
		ColumnExpr("ia.kind").
		ColumnExpr("ia.status").
		ColumnExpr("ia.approval_status").
		ColumnExpr("ia.rebill_strategy").
		ColumnExpr("ia.reason").
		ColumnExpr("ia.policy_reason").
		ColumnExpr("COALESCE(NULLIF(ia.policy_reason, ''), 'Policy-controlled approval') AS policy_source").
		ColumnExpr("ia.credit_total_amount").
		ColumnExpr("ia.rebill_total_amount").
		ColumnExpr("ia.net_delta_amount").
		ColumnExpr("ia.rerate_variance_percent").
		ColumnExpr("ia.would_create_unapplied_credit").
		ColumnExpr("ia.requires_reconciliation_exception").
		ColumnExpr("(ia.replacement_review_status = 'Required') AS requires_replacement_invoice_review").
		ColumnExpr("ia.submitted_by_id").
		ColumnExpr("COALESCE(submitter.name, '') AS submitted_by_name").
		ColumnExpr("ia.submitted_at").
		ColumnExpr("ia.approved_by_id").
		ColumnExpr("COALESCE(approver.name, '') AS approved_by_name").
		ColumnExpr("ia.approved_at").
		ColumnExpr("ia.rejected_by_id").
		ColumnExpr("COALESCE(rejector.name, '') AS rejected_by_name").
		ColumnExpr("ia.rejected_at").
		ColumnExpr("ia.rejection_reason").
		ColumnExpr("ia.credit_memo_invoice_id").
		ColumnExpr("COALESCE(cm.number, '') AS credit_memo_invoice_number").
		ColumnExpr("ia.replacement_invoice_id").
		ColumnExpr("COALESCE(repl.number, '') AS replacement_invoice_number").
		ColumnExpr("ia.rebill_queue_item_id").
		ColumnExpr("COALESCE(rebq.number, '') AS rebill_queue_number").
		ColumnExpr("ia.batch_id").
		ColumnExpr("ia.created_at").
		ColumnExpr("ia.updated_at").
		Join("JOIN invoices AS orig ON orig.id = ia.original_invoice_id AND orig.organization_id = ia.organization_id AND orig.business_unit_id = ia.business_unit_id").
		Join("LEFT JOIN invoices AS cm ON cm.id = ia.credit_memo_invoice_id AND cm.organization_id = ia.organization_id AND cm.business_unit_id = ia.business_unit_id").
		Join("LEFT JOIN invoices AS repl ON repl.id = ia.replacement_invoice_id AND repl.organization_id = ia.organization_id AND repl.business_unit_id = ia.business_unit_id").
		Join("LEFT JOIN billing_queue_items AS rebq ON rebq.id = ia.rebill_queue_item_id AND rebq.organization_id = ia.organization_id AND rebq.business_unit_id = ia.business_unit_id").
		Join("LEFT JOIN users AS submitter ON submitter.id = ia.submitted_by_id").
		Join("LEFT JOIN users AS approver ON approver.id = ia.approved_by_id").
		Join("LEFT JOIN users AS rejector ON rejector.id = ia.rejected_by_id").
		Where("ia.organization_id = ?", req.Filter.TenantInfo.OrgID).
		Where("ia.business_unit_id = ?", req.Filter.TenantInfo.BuID).
		Where("ia.status = ?", invoiceadjustment.StatusPendingApproval)

	applyAdjustmentSearch(query, req.Filter.Query)
	applyApprovalFilters(query, req.Filter.FieldFilters)

	total, err := query.
		OrderExpr("COALESCE(ia.submitted_at, ia.created_at) DESC").
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*invoiceadjustment.ApprovalQueueItem]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) ListReconciliationQueue(
	ctx context.Context,
	req repositories.ListReconciliationQueueRequest,
) (*pagination.ListResult[*invoiceadjustment.ReconciliationQueueItem], error) {
	entities := make([]*invoiceadjustment.ReconciliationQueueItem, 0, req.Filter.Pagination.SafeLimit())
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		ModelTableExpr("invoice_reconciliation_exceptions AS ire").
		ColumnExpr("ire.id AS exception_id").
		ColumnExpr("ire.adjustment_id").
		ColumnExpr("ia.correction_group_id").
		ColumnExpr("ire.status").
		ColumnExpr("ire.reason").
		ColumnExpr("ire.amount").
		ColumnExpr("ia.original_invoice_id").
		ColumnExpr("orig.number AS original_invoice_number").
		ColumnExpr("orig.status AS original_invoice_status").
		ColumnExpr("ire.credit_memo_invoice_id").
		ColumnExpr("COALESCE(cm.number, '') AS credit_memo_invoice_number").
		ColumnExpr("ia.replacement_invoice_id").
		ColumnExpr("COALESCE(repl.number, '') AS replacement_invoice_number").
		ColumnExpr("ia.rebill_queue_item_id").
		ColumnExpr("COALESCE(rebq.number, '') AS rebill_queue_number").
		ColumnExpr("orig.bill_to_name AS customer_name").
		ColumnExpr("ia.kind AS adjustment_kind").
		ColumnExpr("ia.status AS adjustment_status").
		ColumnExpr("COALESCE(NULLIF(ia.policy_reason, ''), 'Adjustment-generated reconciliation') AS policy_source").
		ColumnExpr("ia.submitted_by_id").
		ColumnExpr("COALESCE(submitter.name, '') AS submitted_by_name").
		ColumnExpr("ia.submitted_at").
		ColumnExpr("COALESCE(ire.metadata->>'financeNotes', '') AS finance_notes").
		ColumnExpr("ire.created_at").
		ColumnExpr("ire.updated_at").
		Join("JOIN invoice_adjustments AS ia ON ia.id = ire.adjustment_id AND ia.organization_id = ire.organization_id AND ia.business_unit_id = ire.business_unit_id").
		Join("JOIN invoices AS orig ON orig.id = ia.original_invoice_id AND orig.organization_id = ia.organization_id AND orig.business_unit_id = ia.business_unit_id").
		Join("LEFT JOIN invoices AS cm ON cm.id = ire.credit_memo_invoice_id AND cm.organization_id = ire.organization_id AND cm.business_unit_id = ire.business_unit_id").
		Join("LEFT JOIN invoices AS repl ON repl.id = ia.replacement_invoice_id AND repl.organization_id = ia.organization_id AND repl.business_unit_id = ia.business_unit_id").
		Join("LEFT JOIN billing_queue_items AS rebq ON rebq.id = ia.rebill_queue_item_id AND rebq.organization_id = ia.organization_id AND rebq.business_unit_id = ia.business_unit_id").
		Join("LEFT JOIN users AS submitter ON submitter.id = ia.submitted_by_id").
		Where("ire.organization_id = ?", req.Filter.TenantInfo.OrgID).
		Where("ire.business_unit_id = ?", req.Filter.TenantInfo.BuID)

	applyAdjustmentSearch(query, req.Filter.Query)
	applyReconciliationFilters(query, req.Filter.FieldFilters)

	total, err := query.
		OrderExpr("CASE WHEN ire.status = 'Open' THEN 0 ELSE 1 END ASC").
		OrderExpr("ire.updated_at DESC").
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*invoiceadjustment.ReconciliationQueueItem]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) ListBatchQueue(
	ctx context.Context,
	req repositories.ListBatchQueueRequest,
) (*pagination.ListResult[*invoiceadjustment.BatchQueueItem], error) {
	entities := make([]*invoiceadjustment.BatchQueueItem, 0, req.Filter.Pagination.SafeLimit())
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		ModelTableExpr("invoice_adjustment_batches AS iab").
		ColumnExpr("iab.id AS batch_id").
		ColumnExpr("iab.idempotency_key").
		ColumnExpr("iab.status").
		ColumnExpr("iab.total_count").
		ColumnExpr("iab.processed_count").
		ColumnExpr("iab.succeeded_count").
		ColumnExpr("iab.failed_count").
		ColumnExpr("GREATEST(iab.total_count - iab.processed_count, 0) AS pending_count").
		ColumnExpr("iab.submitted_by_id").
		ColumnExpr("COALESCE(submitter.name, '') AS submitted_by_name").
		ColumnExpr("iab.submitted_at").
		ColumnExpr(`COALESCE((
			SELECT iabi.error_message
			FROM invoice_adjustment_batch_items AS iabi
			WHERE iabi.batch_id = iab.id
				AND iabi.organization_id = iab.organization_id
				AND iabi.business_unit_id = iab.business_unit_id
				AND iabi.error_message IS NOT NULL
				AND iabi.error_message <> ''
			ORDER BY iabi.updated_at DESC
			LIMIT 1
		), '') AS last_failure`).
		ColumnExpr(`COALESCE((
			SELECT COUNT(*)
			FROM invoice_adjustment_batch_items AS iabi
			WHERE iabi.batch_id = iab.id
				AND iabi.organization_id = iab.organization_id
				AND iabi.business_unit_id = iab.business_unit_id
				AND iabi.status = ?
		), 0) AS last_failure_count`, invoiceadjustment.BatchItemStatusFailed).
		ColumnExpr("iab.created_at").
		ColumnExpr("iab.updated_at").
		Join("LEFT JOIN users AS submitter ON submitter.id = iab.submitted_by_id").
		Where("iab.organization_id = ?", req.Filter.TenantInfo.OrgID).
		Where("iab.business_unit_id = ?", req.Filter.TenantInfo.BuID)

	applyBatchSearch(query, req.Filter.Query)
	applyBatchFilters(query, req.Filter.FieldFilters)

	total, err := query.
		OrderExpr("COALESCE(iab.submitted_at, iab.created_at) DESC").
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*invoiceadjustment.BatchQueueItem]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetOperationsSummary(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*invoiceadjustment.OperationsSummary, error) {
	summary := &invoiceadjustment.OperationsSummary{
		AdjustmentsByStatus:         make([]*invoiceadjustment.SummaryCount, 0),
		ReasonDistribution:          make([]*invoiceadjustment.SummaryCount, 0),
		RepeatedAdjustments:         make([]*invoiceadjustment.RepeatedAdjustmentSummary, 0),
		RepeatedCustomerAdjustments: make([]*invoiceadjustment.RepeatedAdjustmentSummary, 0),
	}

	type countRow struct {
		Label string `bun:"label"`
		Count int    `bun:"count"`
	}

	adjustmentCounts := make([]countRow, 0)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		TableExpr("invoice_adjustments AS ia").
		ColumnExpr("ia.status AS label").
		ColumnExpr("COUNT(*) AS count").
		Where("ia.organization_id = ?", tenantInfo.OrgID).
		Where("ia.business_unit_id = ?", tenantInfo.BuID).
		Group("ia.status").
		OrderExpr("COUNT(*) DESC").
		Scan(ctx, &adjustmentCounts); err != nil {
		return nil, fmt.Errorf("get adjustment status summary: %w", err)
	}
	for _, row := range adjustmentCounts {
		summary.AdjustmentsByStatus = append(summary.AdjustmentsByStatus, &invoiceadjustment.SummaryCount{
			Label: row.Label,
			Count: row.Count,
		})
	}

	var err error
	if summary.ApprovalsPending, err = r.countAdjustmentsByStatus(ctx, tenantInfo, invoiceadjustment.StatusPendingApproval); err != nil {
		return nil, err
	}
	if summary.ReconciliationPending, err = r.countReconciliationExceptionsByStatus(ctx, tenantInfo, invoiceadjustment.ExceptionStatusOpen); err != nil {
		return nil, err
	}
	if summary.WriteOffPending, err = r.countPendingWriteOffs(ctx, tenantInfo); err != nil {
		return nil, err
	}
	if summary.BatchesInFlight, err = r.countBatchesInFlight(ctx, tenantInfo); err != nil {
		return nil, err
	}
	if summary.FailedBatchItems, err = r.countFailedBatchItems(ctx, tenantInfo); err != nil {
		return nil, err
	}

	reasonRows := make([]countRow, 0)
	if err = r.db.DBForContext(ctx).
		NewSelect().
		TableExpr("invoice_adjustments AS ia").
		ColumnExpr("COALESCE(NULLIF(TRIM(ia.reason), ''), 'Unspecified') AS label").
		ColumnExpr("COUNT(*) AS count").
		Where("ia.organization_id = ?", tenantInfo.OrgID).
		Where("ia.business_unit_id = ?", tenantInfo.BuID).
		GroupExpr("COALESCE(NULLIF(TRIM(ia.reason), ''), 'Unspecified')").
		OrderExpr("COUNT(*) DESC").
		Limit(8).
		Scan(ctx, &reasonRows); err != nil {
		return nil, fmt.Errorf("get adjustment reason summary: %w", err)
	}
	for _, row := range reasonRows {
		summary.ReasonDistribution = append(summary.ReasonDistribution, &invoiceadjustment.SummaryCount{
			Label: row.Label,
			Count: row.Count,
		})
	}

	type repeatedRow struct {
		EntityID   pulid.ID `bun:"entity_id"`
		EntityType string   `bun:"entity_type"`
		Label      string   `bun:"label"`
		Count      int      `bun:"count"`
	}

	repeatedInvoices := make([]repeatedRow, 0)
	if err = r.db.DBForContext(ctx).
		NewSelect().
		TableExpr("invoice_adjustments AS ia").
		ColumnExpr("ia.original_invoice_id AS entity_id").
		ColumnExpr("'invoice' AS entity_type").
		ColumnExpr("orig.number AS label").
		ColumnExpr("COUNT(*) AS count").
		Join("JOIN invoices AS orig ON orig.id = ia.original_invoice_id AND orig.organization_id = ia.organization_id AND orig.business_unit_id = ia.business_unit_id").
		Where("ia.organization_id = ?", tenantInfo.OrgID).
		Where("ia.business_unit_id = ?", tenantInfo.BuID).
		Group("ia.original_invoice_id", "orig.number").
		Having("COUNT(*) > 1").
		OrderExpr("COUNT(*) DESC, MAX(ia.created_at) DESC").
		Limit(5).
		Scan(ctx, &repeatedInvoices); err != nil {
		return nil, fmt.Errorf("get repeated invoice adjustments: %w", err)
	}
	for _, row := range repeatedInvoices {
		summary.RepeatedAdjustments = append(summary.RepeatedAdjustments, &invoiceadjustment.RepeatedAdjustmentSummary{
			EntityID:   row.EntityID,
			EntityType: row.EntityType,
			Label:      row.Label,
			Count:      row.Count,
		})
	}

	repeatedCustomers := make([]repeatedRow, 0)
	if err = r.db.DBForContext(ctx).
		NewSelect().
		TableExpr("invoice_adjustments AS ia").
		ColumnExpr("orig.customer_id AS entity_id").
		ColumnExpr("'customer' AS entity_type").
		ColumnExpr("orig.bill_to_name AS label").
		ColumnExpr("COUNT(*) AS count").
		Join("JOIN invoices AS orig ON orig.id = ia.original_invoice_id AND orig.organization_id = ia.organization_id AND orig.business_unit_id = ia.business_unit_id").
		Where("ia.organization_id = ?", tenantInfo.OrgID).
		Where("ia.business_unit_id = ?", tenantInfo.BuID).
		Group("orig.customer_id", "orig.bill_to_name").
		Having("COUNT(*) > 1").
		OrderExpr("COUNT(*) DESC, MAX(ia.created_at) DESC").
		Limit(5).
		Scan(ctx, &repeatedCustomers); err != nil {
		return nil, fmt.Errorf("get repeated customer adjustments: %w", err)
	}
	for _, row := range repeatedCustomers {
		summary.RepeatedCustomerAdjustments = append(summary.RepeatedCustomerAdjustments, &invoiceadjustment.RepeatedAdjustmentSummary{
			EntityID:   row.EntityID,
			EntityType: row.EntityType,
			Label:      row.Label,
			Count:      row.Count,
		})
	}

	return summary, nil
}

func applyAdjustmentSearch(query *bun.SelectQuery, search string) {
	if search == "" {
		return
	}

	pattern := "%" + search + "%"
	query.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.
			Where("ia.id ILIKE ?", pattern).
			WhereOr("orig.number ILIKE ?", pattern).
			WhereOr("orig.bill_to_name ILIKE ?", pattern).
			WhereOr("ia.reason ILIKE ?", pattern).
			WhereOr("ia.policy_reason ILIKE ?", pattern).
			WhereOr("submitter.name ILIKE ?", pattern)
	})
}

func applyApprovalFilters(query *bun.SelectQuery, filters []domaintypes.FieldFilter) {
	for _, filter := range filters {
		switch filter.Field {
		case "kind":
			query.Where("ia.kind = ?", filter.Value)
		case "submittedById":
			query.Where("ia.submitted_by_id = ?", filter.Value)
		}
	}
}

func applyReconciliationFilters(query *bun.SelectQuery, filters []domaintypes.FieldFilter) {
	for _, filter := range filters {
		switch filter.Field {
		case "status":
			query.Where("ire.status = ?", filter.Value)
		case "adjustmentKind":
			query.Where("ia.kind = ?", filter.Value)
		}
	}
}

func applyBatchSearch(query *bun.SelectQuery, search string) {
	if search == "" {
		return
	}

	pattern := "%" + search + "%"
	query.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.
			Where("iab.id ILIKE ?", pattern).
			WhereOr("iab.idempotency_key ILIKE ?", pattern).
			WhereOr("submitter.name ILIKE ?", pattern)
	})
}

func applyBatchFilters(query *bun.SelectQuery, filters []domaintypes.FieldFilter) {
	for _, filter := range filters {
		if filter.Field == "status" {
			query.Where("iab.status = ?", filter.Value)
		}
	}
}

func (r *repository) countAdjustmentsByStatus(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	status invoiceadjustment.Status,
) (int, error) {
	count, err := r.db.DBForContext(ctx).
		NewSelect().
		TableExpr("invoice_adjustments AS ia").
		Where("ia.organization_id = ?", tenantInfo.OrgID).
		Where("ia.business_unit_id = ?", tenantInfo.BuID).
		Where("ia.status = ?", status).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("count invoice adjustments by status: %w", err)
	}
	return count, nil
}

func (r *repository) countReconciliationExceptionsByStatus(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	status invoiceadjustment.ExceptionStatus,
) (int, error) {
	count, err := r.db.DBForContext(ctx).
		NewSelect().
		TableExpr("invoice_reconciliation_exceptions AS ire").
		Where("ire.organization_id = ?", tenantInfo.OrgID).
		Where("ire.business_unit_id = ?", tenantInfo.BuID).
		Where("ire.status = ?", status).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("count reconciliation exceptions by status: %w", err)
	}
	return count, nil
}

func (r *repository) countPendingWriteOffs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (int, error) {
	count, err := r.db.DBForContext(ctx).
		NewSelect().
		TableExpr("invoice_adjustments AS ia").
		Where("ia.organization_id = ?", tenantInfo.OrgID).
		Where("ia.business_unit_id = ?", tenantInfo.BuID).
		Where("ia.kind = ?", invoiceadjustment.KindWriteOff).
		Where("ia.status IN (?)", bun.In([]invoiceadjustment.Status{
			invoiceadjustment.StatusPendingApproval,
			invoiceadjustment.StatusApproved,
			invoiceadjustment.StatusExecuting,
			invoiceadjustment.StatusExecuted,
			invoiceadjustment.StatusExecutionFailed,
		})).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("count pending write-offs: %w", err)
	}
	return count, nil
}

func (r *repository) countBatchesInFlight(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (int, error) {
	count, err := r.db.DBForContext(ctx).
		NewSelect().
		TableExpr("invoice_adjustment_batches AS iab").
		Where("iab.organization_id = ?", tenantInfo.OrgID).
		Where("iab.business_unit_id = ?", tenantInfo.BuID).
		Where("iab.status IN (?)", bun.In([]invoiceadjustment.BatchStatus{
			invoiceadjustment.BatchStatusPending,
			invoiceadjustment.BatchStatusQueued,
			invoiceadjustment.BatchStatusSubmitted,
			invoiceadjustment.BatchStatusRunning,
		})).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("count in-flight adjustment batches: %w", err)
	}
	return count, nil
}

func (r *repository) countFailedBatchItems(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (int, error) {
	count, err := r.db.DBForContext(ctx).
		NewSelect().
		TableExpr("invoice_adjustment_batch_items AS iabi").
		Where("iabi.organization_id = ?", tenantInfo.OrgID).
		Where("iabi.business_unit_id = ?", tenantInfo.BuID).
		Where("iabi.status = ?", invoiceadjustment.BatchItemStatusFailed).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("count failed adjustment batch items: %w", err)
	}
	return count, nil
}
