package receivablesql

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/factoringsql"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring/factoringdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany/factoringcompanydb"
)

const SecondsPerDay = int64(86400)

const OpenInvoice = `inv.status = 'Posted'
	  AND inv.bill_type IN ('Invoice', 'DebitMemo')
	  AND inv.total_amount_minor > inv.applied_amount_minor`

func Receivable(inv *invoicedb.InvoicesTable) keel.Pred {
	return keel.And(
		inv.Status.Eq(invoice.StatusPosted),
		inv.BillType.In(billingqueue.BillTypeInvoice, billingqueue.BillTypeDebitMemo),
	)
}

func Open(inv *invoicedb.InvoicesTable) keel.Pred {
	return keel.And(
		Receivable(inv),
		keel.DynOf(inv.TotalAmountMinor).Gt(inv.AppliedAmountMinor),
	)
}

func OpenMinor(inv *invoicedb.InvoicesTable) keel.Expr {
	return keel.DynOf(inv.TotalAmountMinor).Sub(inv.AppliedAmountMinor)
}

func SoldWithoutRecourse(inv *invoicedb.InvoicesTable) keel.Pred {
	fac := factoringcompanydb.FactoringCompanies.As("fac")
	agr := factoringdb.Agreements.As("agr")
	return keel.And(
		keel.Exists(fac.
			LeftJoin(agr, agr.ID.EqCol(inv.FactoringAgreementID)).
			Pick(keel.Raw("1")).
			Where(
				fac.ID.EqCol(inv.FactoringCompanyID),
				keel.Raw("COALESCE(? = 'NonRecourse', ?)", agr.Recourse, fac.NonRecourse),
			)),
		keel.Not(factoringsql.ChargedBackInvoicePred(inv)),
	)
}
