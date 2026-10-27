package factoringsql

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring/factoringdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
)

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
	return keel.Exists(item.Where(
		item.InvoiceID.EqCol(inv.ID),
		item.Status.In(statuses...),
	))
}
