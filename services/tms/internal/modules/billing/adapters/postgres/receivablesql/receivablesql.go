package receivablesql

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/factoringsql"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt/bankreceiptdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingvocab"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger/customerledgerdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment/customerpaymentdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring/factoringdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany/factoringcompanydb"
)

const SecondsPerDay = int64(86400)

func LedgerInvoiceID(cle *customerledgerdb.CustomerLedgerEntriesTable) keel.Expr {
	return cle.RelatedInvoiceID.Coalesce(
		keel.CaseString().When(cle.SourceObjectType.Eq("Invoice"), cle.SourceObjectID).End(),
	)
}

func ReceivableBillType(inv *invoicedb.InvoicesTable) keel.Pred {
	return inv.BillType.In(billingvocab.BillTypeInvoice, billingvocab.BillTypeDebitMemo)
}

func Receivable(inv *invoicedb.InvoicesTable) keel.Pred {
	return keel.And(inv.Status.Eq(invoice.StatusPosted), ReceivableBillType(inv))
}

func Open(inv *invoicedb.InvoicesTable) keel.Pred {
	return keel.And(
		Receivable(inv),
		keel.Int64Of(inv.TotalAmountMinor).GtExpr(inv.AppliedAmountMinor),
	)
}

func OpenMinor(inv *invoicedb.InvoicesTable) keel.Int64Expr {
	return inv.TotalAmountMinor.Sub(inv.AppliedAmountMinor)
}

func UnmatchedPayment(cp *customerpaymentdb.PaymentsTable) keel.Pred {
	br := bankreceiptdb.BankReceipts.As("br")

	return keel.NotExists(br.Where(
		br.MatchedCustomerPaymentID.EqCol(cp.ID),
		br.Status.Eq(bankreceipt.StatusMatched),
	))
}

func ShownCandidate(
	c *bankreceiptdb.CandidatesTable,
	cp *customerpaymentdb.PaymentsTable,
	inv *invoicedb.InvoicesTable,
) keel.Pred {
	return keel.Or(
		keel.And(
			c.Kind.Eq(bankreceipt.CandidateCustomerPayment),
			cp.Status.Eq(customerpayment.StatusPosted),
			UnmatchedPayment(cp),
		),
		keel.And(c.Kind.Eq(bankreceipt.CandidateInvoice), inv.ID.IsNotNull()),
	)
}

func SoldWithoutRecourse(inv *invoicedb.InvoicesTable) keel.Pred {
	fac := factoringcompanydb.FactoringCompanies.As("fac")
	agr := factoringdb.Agreements.As("agr")
	return keel.And(
		keel.Exists(fac.
			LeftJoin(agr, agr.ID.EqCol(inv.FactoringAgreementID)).
			Where(
				fac.ID.EqCol(inv.FactoringCompanyID),
				keel.BoolOf(agr.Recourse.Eq(factoring.RecourseNone)).Coalesce(fac.NonRecourse),
			)),
		keel.Not(factoringsql.ChargedBackInvoicePred(inv)),
	)
}
