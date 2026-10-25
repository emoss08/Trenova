package invoiceadjustmentrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment/invoiceadjustmentdb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type summaryCount struct {
	Label string
	Count int
}

type repeatedCount struct {
	EntityID pulid.ID
	Label    string
	Count    int
}

func (r *repository) GetOperationsSummary(
	ctx context.Context,
	_ pagination.TenantInfo,
) (*invoiceadjustmentports.InvoiceAdjustmentOperationsSummary, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*invoiceadjustmentports.InvoiceAdjustmentOperationsSummary, error) {
			db := r.db.Keel(ctx)
			ia := invoiceadjustmentdb.InvoiceAdjustments
			ex := invoiceadjustmentdb.InvoiceAdjustmentReconciliationExceptions
			b := invoiceadjustmentdb.InvoiceAdjustmentBatches
			bi := invoiceadjustmentdb.InvoiceAdjustmentBatchItems
			orig := ia.OriginalInvoice()
			reason := keel.DynFunc("trim", ia.Reason).NullIf("").Coalesce("Unspecified")

			var (
				byStatus, reasons            []summaryCount
				repeatedInvoices, repeatedCu []repeatedCount
				approvals, reconciliation    int64
				writeOffs, batches, failures int64
			)
			err := keel.Batch(
				ctx,
				db,
				keel.IntoSlice(ia.Group(ia.Status).
					Pick(ia.Status, keel.Count()).
					Order(keel.Count().Desc()), &byStatus),
				keel.IntoSlice(ia.Group(reason).
					Pick(reason, keel.Count()).
					Order(keel.Count().Desc()).
					Limit(8), &reasons),
				keel.IntoSlice(ia.Where(orig.ID.IsNotNull()).
					Group(ia.OriginalInvoiceID, orig.Number).
					Pick(ia.OriginalInvoiceID, orig.Number, keel.Count()).
					Having(keel.Count().Gt(1)).
					Order(keel.Count().Desc(), ia.CreatedAt.Max().Desc()).
					Limit(5), &repeatedInvoices),
				keel.IntoSlice(ia.Where(orig.ID.IsNotNull()).
					Group(orig.CustomerID, orig.BillToName).
					Pick(orig.CustomerID, orig.BillToName, keel.Count()).
					Having(keel.Count().Gt(1)).
					Order(keel.Count().Desc(), ia.CreatedAt.Max().Desc()).
					Limit(5), &repeatedCu),
				keel.CountInto(
					ia.Where(ia.Status.Eq(invoiceadjustment.StatusPendingApproval)),
					&approvals,
				),
				keel.CountInto(
					ex.Where(ex.Status.Eq(invoiceadjustment.ExceptionStatusOpen)),
					&reconciliation,
				),
				keel.CountInto(ia.Where(
					ia.Kind.Eq(invoiceadjustment.KindWriteOff),
					ia.Status.In(
						invoiceadjustment.StatusPendingApproval,
						invoiceadjustment.StatusApproved,
						invoiceadjustment.StatusExecuting,
						invoiceadjustment.StatusExecuted,
						invoiceadjustment.StatusExecutionFailed,
					),
				), &writeOffs),
				keel.CountInto(b.Where(b.Status.In(
					invoiceadjustment.BatchStatusPending,
					invoiceadjustment.BatchStatusQueued,
					invoiceadjustment.BatchStatusSubmitted,
					invoiceadjustment.BatchStatusRunning,
				)), &batches),
				keel.CountInto(
					bi.Where(bi.Status.Eq(invoiceadjustment.BatchItemStatusFailed)),
					&failures,
				),
			)
			if err != nil {
				return nil, err
			}

			return &invoiceadjustmentports.InvoiceAdjustmentOperationsSummary{
				AdjustmentsByStatus:         summaryCounts(byStatus),
				ApprovalsPending:            int(approvals),
				ReconciliationPending:       int(reconciliation),
				WriteOffPending:             int(writeOffs),
				BatchesInFlight:             int(batches),
				FailedBatchItems:            int(failures),
				ReasonDistribution:          summaryCounts(reasons),
				RepeatedAdjustments:         repeatedSummaries(repeatedInvoices, "invoice"),
				RepeatedCustomerAdjustments: repeatedSummaries(repeatedCu, "customer"),
			}, nil
		},
	)
}

func summaryCounts(rows []summaryCount) []*invoiceadjustmentports.InvoiceAdjustmentSummaryCount {
	out := make([]*invoiceadjustmentports.InvoiceAdjustmentSummaryCount, 0, len(rows))
	for _, row := range rows {
		out = append(out, &invoiceadjustmentports.InvoiceAdjustmentSummaryCount{
			Label: row.Label,
			Count: row.Count,
		})
	}

	return out
}

func repeatedSummaries(
	rows []repeatedCount,
	entityType string,
) []*invoiceadjustmentports.InvoiceAdjustmentRepeatedSummary {
	out := make([]*invoiceadjustmentports.InvoiceAdjustmentRepeatedSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, &invoiceadjustmentports.InvoiceAdjustmentRepeatedSummary{
			EntityID:   row.EntityID,
			EntityType: entityType,
			Label:      row.Label,
			Count:      row.Count,
		})
	}

	return out
}
