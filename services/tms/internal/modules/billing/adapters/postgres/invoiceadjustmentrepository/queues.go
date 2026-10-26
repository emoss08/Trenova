package invoiceadjustmentrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment/invoiceadjustmentdb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type (
	approvalItem       = invoiceadjustmentports.InvoiceAdjustmentApprovalQueueItem
	reconciliationItem = invoiceadjustmentports.InvoiceAdjustmentReconciliationQueueItem
)

var approvalQueueCursorSort = []pagination.CursorSortField{
	{Field: "sortAt", Direction: "desc"},
	{Field: "id", Direction: "desc"},
}

func (r *repository) ListApprovalQueue(
	ctx context.Context,
	req *invoiceadjustmentports.ListApprovalQueueRequest,
) (*pagination.CursorListResult[*approvalItem], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*approvalItem], error) {
			if req == nil || req.Filter == nil {
				return nil, errortypes.NewValidationError(
					"filter",
					errortypes.ErrRequired,
					"Approval queue filter is required",
				)
			}

			page, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				approvalQueue(req.Filter),
				&listquery.Request[invoiceadjustment.InvoiceAdjustment]{
					Cursor: req.Cursor,
					Shape:  approvalQueueShape,
					Keyset: approvalQueueKeyset(),
				},
			)
			if err != nil {
				return nil, err
			}

			items := make([]*approvalItem, len(page.Items))
			for i, adjustment := range page.Items {
				items[i] = approvalQueueItem(adjustment)
			}

			return &pagination.CursorListResult[*approvalItem]{
				Items:           items,
				HasNextPage:     page.HasNextPage,
				TotalCount:      page.TotalCount,
				TotalCountExact: page.TotalCountExact,
				EdgeCursors:     page.EdgeCursors,
			}, nil
		},
	)
}

func approvalQueueKeyset() *listquery.Keyset {
	ia := invoiceadjustmentdb.InvoiceAdjustments
	return &listquery.Keyset{
		Orders: []keel.Order{
			ia.SubmittedAt.Coalesce(ia.CreatedAt).Desc(),
			ia.ID.Desc(),
		},
		Shape: approvalQueueCursorSort,
	}
}

func approvalQueue(
	filter *pagination.QueryOptions,
) *keel.Query[*invoiceadjustment.InvoiceAdjustment] {
	ia := invoiceadjustmentdb.InvoiceAdjustments
	preds := []keel.Pred{
		ia.Status.Eq(invoiceadjustment.StatusPendingApproval),
		adjustmentSearch(ia, filter.Query),
	}
	for _, f := range filter.FieldFilters {
		value, ok := filterText(f)
		if !ok {
			continue
		}
		switch f.Field {
		case "kind":
			preds = append(preds, ia.Kind.Eq(invoiceadjustment.Kind(value)))
		case "submittedById":
			preds = append(preds, ia.SubmittedByID.Eq(pulid.ID(value)))
		}
	}

	return ia.Where(preds...)
}

func approvalQueueShape(
	q *keel.Query[*invoiceadjustment.InvoiceAdjustment],
) *keel.Query[*invoiceadjustment.InvoiceAdjustment] {
	ia := invoiceadjustmentdb.InvoiceAdjustments
	orig := ia.OriginalInvoice()
	cm := ia.CreditMemoInvoice()
	repl := ia.ReplacementInvoice()
	rebill := ia.RebillQueueItem()

	return q.Load(
		orig.Pick(orig.Number, orig.Status, orig.BillToName),
		cm.Pick(cm.Number),
		repl.Pick(repl.Number),
		rebill.Pick(rebill.Number),
		userName(ia.SubmittedBy()),
		userName(ia.ApprovedBy()),
		userName(ia.RejectedBy()),
	)
}

func approvalQueueItem(a *invoiceadjustment.InvoiceAdjustment) *approvalItem {
	item := &approvalItem{
		AdjustmentID:      a.ID,
		CorrectionGroupID: a.CorrectionGroupID,
		OriginalInvoiceID: a.OriginalInvoiceID,
		Kind:              a.Kind,
		Status:            a.Status,
		ApprovalStatus:    a.ApprovalStatus,
		RebillStrategy:    a.RebillStrategy,
		Reason:            a.Reason,
		PolicyReason:      a.PolicyReason,
		PolicySource: policySource(
			a.PolicyReason,
			"Policy-controlled approval",
		),
		CreditTotalAmount:                a.CreditTotalAmount,
		RebillTotalAmount:                a.RebillTotalAmount,
		NetDeltaAmount:                   a.NetDeltaAmount,
		RerateVariancePercent:            a.RerateVariancePercent,
		WouldCreateUnappliedCredit:       a.WouldCreateUnappliedCredit,
		RequiresReconciliationException:  a.RequiresReconciliationException,
		RequiresReplacementInvoiceReview: a.ReplacementReviewStatus == invoiceadjustment.ReplacementReviewStatusRequired,
		SubmittedByID:                    a.SubmittedByID,
		SubmittedByName:                  nameOf(a.SubmittedBy),
		SubmittedAt:                      a.SubmittedAt,
		ApprovedByID:                     a.ApprovedByID,
		ApprovedByName:                   nameOf(a.ApprovedBy),
		ApprovedAt:                       a.ApprovedAt,
		RejectedByID:                     a.RejectedByID,
		RejectedByName:                   nameOf(a.RejectedBy),
		RejectedAt:                       a.RejectedAt,
		RejectionReason:                  a.RejectionReason,
		CreditMemoInvoiceID:              a.CreditMemoInvoiceID,
		ReplacementInvoiceID:             a.ReplacementInvoiceID,
		RebillQueueItemID:                a.RebillQueueItemID,
		BatchID:                          a.BatchID,
		CreatedAt:                        a.CreatedAt,
		UpdatedAt:                        a.UpdatedAt,
	}
	if a.OriginalInvoice != nil {
		item.OriginalInvoiceNumber = a.OriginalInvoice.Number
		item.OriginalInvoiceStatus = string(a.OriginalInvoice.Status)
		item.CustomerName = a.OriginalInvoice.BillToName
	}
	if a.CreditMemoInvoice != nil {
		item.CreditMemoInvoiceNumber = a.CreditMemoInvoice.Number
	}
	if a.ReplacementInvoice != nil {
		item.ReplacementInvoiceNumber = a.ReplacementInvoice.Number
	}
	if a.RebillQueueItem != nil {
		item.RebillQueueNumber = a.RebillQueueItem.Number
	}

	return item
}

func (r *repository) ListReconciliationQueue(
	ctx context.Context,
	req *invoiceadjustmentports.ListReconciliationQueueRequest,
) (*pagination.ListResult[*reconciliationItem], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*reconciliationItem], error) {
			rows, err := listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				reconciliationQueue(&req.Filter),
				&pagination.SelectQueryRequest{Pagination: req.Filter.Pagination},
				nil,
			)
			if err != nil {
				return nil, err
			}

			items := make([]*reconciliationItem, len(rows.Items))
			for i, exception := range rows.Items {
				items[i] = reconciliationQueueItem(exception)
			}

			return &pagination.ListResult[*reconciliationItem]{
				Items:      items,
				Total:      rows.Total,
				TotalExact: rows.TotalExact,
			}, nil
		},
	)
}

func reconciliationQueue(
	filter *pagination.QueryOptions,
) *keel.Query[*invoiceadjustment.InvoiceAdjustmentReconciliationException] {
	ex := invoiceadjustmentdb.InvoiceAdjustmentReconciliationExceptions
	adj := ex.Adjustment()
	orig := adj.OriginalInvoice()
	cm := ex.CreditMemoInvoice()
	repl := adj.ReplacementInvoice()
	rebill := adj.RebillQueueItem()

	preds := []keel.Pred{
		adj.ID.IsNotNull(),
		orig.ID.IsNotNull(),
		adjustmentSearch(adj, filter.Query),
	}
	for _, f := range filter.FieldFilters {
		value, ok := filterText(f)
		if !ok {
			continue
		}
		switch f.Field {
		case "status":
			preds = append(preds, ex.Status.Eq(invoiceadjustment.ExceptionStatus(value)))
		case "adjustmentKind":
			preds = append(preds, adj.Kind.Eq(invoiceadjustment.Kind(value)))
		}
	}

	return ex.Where(preds...).
		Load(
			cm.Pick(cm.Number),
			adj.Load(
				orig.Pick(orig.Number, orig.Status, orig.BillToName),
				repl.Pick(repl.Number),
				rebill.Pick(rebill.Number),
				userName(adj.SubmittedBy()),
			),
		).
		Order(
			keel.Desc(ex.Status.Eq(invoiceadjustment.ExceptionStatusOpen)).NullsLast(),
			ex.UpdatedAt.Desc(),
		)
}

func reconciliationQueueItem(
	e *invoiceadjustment.InvoiceAdjustmentReconciliationException,
) *reconciliationItem {
	item := &reconciliationItem{
		ExceptionID:         e.ID,
		AdjustmentID:        e.AdjustmentID,
		Status:              e.Status,
		Reason:              e.Reason,
		Amount:              e.Amount,
		CreditMemoInvoiceID: e.CreditMemoInvoiceID,
		PolicySource:        policySource("", "Adjustment-generated reconciliation"),
		CreatedAt:           e.CreatedAt,
		UpdatedAt:           e.UpdatedAt,
	}
	if notes, ok := e.Metadata["financeNotes"].(string); ok {
		item.FinanceNotes = notes
	}
	if e.CreditMemoInvoice != nil {
		item.CreditMemoInvoiceNumber = e.CreditMemoInvoice.Number
	}

	a := e.Adjustment
	if a == nil {
		return item
	}
	item.CorrectionGroupID = a.CorrectionGroupID
	item.OriginalInvoiceID = a.OriginalInvoiceID
	item.ReplacementInvoiceID = a.ReplacementInvoiceID
	item.RebillQueueItemID = a.RebillQueueItemID
	item.AdjustmentKind = a.Kind
	item.AdjustmentStatus = a.Status
	item.PolicySource = policySource(a.PolicyReason, "Adjustment-generated reconciliation")
	item.SubmittedByID = a.SubmittedByID
	item.SubmittedByName = nameOf(a.SubmittedBy)
	item.SubmittedAt = a.SubmittedAt
	if a.OriginalInvoice != nil {
		item.OriginalInvoiceNumber = a.OriginalInvoice.Number
		item.OriginalInvoiceStatus = string(a.OriginalInvoice.Status)
		item.CustomerName = a.OriginalInvoice.BillToName
	}
	if a.ReplacementInvoice != nil {
		item.ReplacementInvoiceNumber = a.ReplacementInvoice.Number
	}
	if a.RebillQueueItem != nil {
		item.RebillQueueNumber = a.RebillQueueItem.Number
	}

	return item
}

func (r *repository) ListBatchQueue(
	ctx context.Context,
	req *invoiceadjustmentports.ListBatchQueueRequest,
) (*pagination.ListResult[*invoiceadjustment.InvoiceAdjustmentBatch], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*invoiceadjustment.InvoiceAdjustmentBatch], error) {
			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				batchQueue(&req.Filter),
				&pagination.SelectQueryRequest{Pagination: req.Filter.Pagination},
				nil,
			)
		},
	)
}

func batchQueue(
	filter *pagination.QueryOptions,
) *keel.Query[*invoiceadjustment.InvoiceAdjustmentBatch] {
	b := invoiceadjustmentdb.InvoiceAdjustmentBatches
	bi := invoiceadjustmentdb.InvoiceAdjustmentBatchItems.As("iabi")
	failures := invoiceadjustmentdb.InvoiceAdjustmentBatchItems.As("iabf")
	submitter := b.SubmittedBy()

	preds := []keel.Pred{}
	if filter.Query != "" {
		preds = append(preds, keel.Or(
			b.ID.ContainsFold(filter.Query),
			b.IdempotencyKey.ContainsFold(filter.Query),
			submitter.Name.ContainsFold(filter.Query),
		))
	}
	for _, f := range filter.FieldFilters {
		if value, ok := filterText(f); ok && f.Field == "status" {
			preds = append(preds, b.Status.Eq(invoiceadjustment.BatchStatus(value)))
		}
	}

	lastFailure := keel.Scalar(bi.Pick(bi.ErrorMessage).
		Where(bi.BatchID.EqCol(b.ID), bi.ErrorMessage.IsNotNull(), bi.ErrorMessage.Ne("")).
		Order(bi.UpdatedAt.Desc()).
		Limit(1))
	failureCount := keel.Scalar(failures.Pick(keel.Count()).Where(
		failures.BatchID.EqCol(b.ID),
		failures.Status.Eq(invoiceadjustment.BatchItemStatusFailed),
	))

	return b.Where(preds...).
		Compute(
			b.PendingCount,
			keel.Int64Of(keel.DynOf(b.TotalCount).Sub(b.ProcessedCount)).Greatest(keel.Int64(0)),
		).
		Compute(b.SubmittedByName, submitter.Name.Coalesce("")).
		Compute(b.LastFailure, keel.StringOf(lastFailure).Coalesce("")).
		Compute(b.LastFailureCount, keel.Int64Of(failureCount).Coalesce(0)).
		Order(b.SubmittedAt.Coalesce(b.CreatedAt).Desc())
}

func adjustmentSearch(ia *invoiceadjustmentdb.InvoiceAdjustmentsTable, search string) keel.Pred {
	if search == "" {
		return nil
	}

	orig := ia.OriginalInvoice()
	return keel.Or(
		ia.ID.ContainsFold(search),
		orig.Number.ContainsFold(search),
		orig.BillToName.ContainsFold(search),
		ia.Reason.ContainsFold(search),
		ia.PolicyReason.ContainsFold(search),
		ia.SubmittedBy().Name.ContainsFold(search),
	)
}

func userName(u *tenantdb.UsersTable) *keel.Query[*tenant.User] {
	return u.Pick(u.Name)
}

func nameOf(u *tenant.User) string {
	if u == nil {
		return ""
	}

	return u.Name
}

func policySource(reason, fallback string) string {
	if reason == "" {
		return fallback
	}

	return reason
}

func filterText(f domaintypes.FieldFilter) (string, bool) {
	value, ok := f.Value.(string)
	return value, ok && value != ""
}
