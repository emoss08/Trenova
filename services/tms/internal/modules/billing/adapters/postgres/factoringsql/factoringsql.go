package factoringsql

import (
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring/factoringdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
)

var (
	chargedBackInvoice = invoiceWithItem(factoring.SubmissionItemStatusChargedBack)
	liveInvoice        = invoiceWithItem(factoring.LiveSubmissionItemStatuses()...)
)

func ChargedBackInvoice() string {
	return chargedBackInvoice
}

func LiveInvoice() string {
	return liveInvoice
}

func ChargedBackInvoicePred(inv *invoicedb.InvoicesTable) keel.Pred {
	return invoiceWithItemPred(inv, factoring.SubmissionItemStatusChargedBack)
}

func LiveInvoicePred(inv *invoicedb.InvoicesTable) keel.Pred {
	return invoiceWithItemPred(inv, factoring.LiveSubmissionItemStatuses()...)
}

func invoiceWithItemPred(
	inv *invoicedb.InvoicesTable,
	statuses ...factoring.SubmissionItemStatus,
) keel.Pred {
	item := factoringdb.SubmissionItems.As("fsi")
	return keel.Exists(item.Pick(keel.Raw("1")).Where(
		item.InvoiceID.EqCol(inv.ID),
		item.Status.In(statuses...),
	))
}

func invoiceWithItem(statuses ...factoring.SubmissionItemStatus) string {
	quoted := make([]string, 0, len(statuses))
	for _, status := range statuses {
		quoted = append(quoted, "'"+string(status)+"'")
	}

	return "EXISTS (SELECT 1 FROM factoring_submission_items AS fsi" +
		" WHERE fsi.organization_id = inv.organization_id" +
		" AND fsi.business_unit_id = inv.business_unit_id" +
		" AND fsi.invoice_id = inv.id" +
		" AND fsi.status IN (" + strings.Join(quoted, ", ") + "))"
}
