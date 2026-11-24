package billingboard

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
)

const (
	MeasureAmountDue = "amountDue"
	MeasureTotal     = "total"
	MeasureAge       = "age"
	MeasureReceived  = "received"

	MeasureAgainstCandidate = "againstCandidate"
)

var Kinds = []boardcatalog.Kind{
	{Kind: recordkind.Kind(permission.ResourceInvoice)},
	{Kind: recordkind.Kind(permission.ResourceInvoiceDispute)},
	{Kind: recordkind.Kind(permission.ResourceBankReceipt), Words: "bank receipts"},
}

var Measures = []boardcatalog.Measure{
	{
		Name:     MeasureAmountDue,
		Kind:     recordkind.Kind(permission.ResourceInvoice),
		Shape:    boardcatalog.ShapeAmount,
		Value:    boardcatalog.ValueMoney,
		Summable: true,
		Adverse:  true,
		Default:  true,
		Aliases: []string{
			"openAmount", "amountOpen", "balanceDue", "balanceDueMinor", "balance", "due",
		},
		Reads: boardcatalog.Read(permission.ResourceInvoice, "balanceDue"),
		Says: []string{
			"owed", "owe", "owes", "owing", "outstanding", "unpaid", "due", "balance", "open",
		},
		Label:   "Amount due",
		Total:   "Total due",
		Largest: "Largest amount due",
		Heading: "Due",
	},
	{
		Name:     MeasureTotal,
		Kind:     recordkind.Kind(permission.ResourceInvoice),
		Shape:    boardcatalog.ShapeAmount,
		Value:    boardcatalog.ValueMoney,
		Summable: true,
		Aliases:  []string{"totalAmount", "totalAmountMinor", "invoiceTotal", "amount"},
		Reads:    boardcatalog.Read(permission.ResourceInvoice, "totalAmount"),
		Says:     []string{"billed", "invoiced", "charged"},
		Label:    "Invoice total",
		Total:    "Total billed",
		Largest:  "Largest invoice total",
		Heading:  "Total",
	},
	{
		Name:    MeasureAge,
		Kind:    recordkind.Kind(permission.ResourceInvoice),
		Shape:   boardcatalog.ShapeTrack,
		Value:   boardcatalog.ValueDuration,
		Adverse: true,
		Aliases: []string{"daysOld", "ageDays", "daysOutstanding"},
		Reads:   boardcatalog.Read(permission.ResourceInvoice, "invoiceDate"),
		Label:   "Age",
		Total:   "Total age",
		Largest: "Oldest",
		Heading: "Age",
	},
	{
		Name:     MeasureReceived,
		Kind:     recordkind.Kind(permission.ResourceBankReceipt),
		Shape:    boardcatalog.ShapeAmount,
		Value:    boardcatalog.ValueMoney,
		Summable: true,
		Aliases: []string{
			"amount", "amountMinor", "receivedAmount", "receiptAmount", "deposit", "total",
		},
		Reads: boardcatalog.Read(
			permission.ResourceBankReceipt, "amountMinor", "exceptionReason", "receiptDate",
		),
		Words:   "what the bank received, and why it is not applied",
		Label:   "Received",
		Total:   "Total received",
		Largest: "Largest receipt",
		Heading: "Received",
	},
	{
		Name:    MeasureAgainstCandidate,
		Kind:    recordkind.Kind(permission.ResourceBankReceipt),
		Shape:   boardcatalog.ShapeBounded,
		Value:   boardcatalog.ValueMoney,
		Default: true,
		Requires: []permission.Resource{
			permission.ResourceInvoice, permission.ResourceCustomerPayment,
		},
		Aliases: []string{
			"candidate", "bestCandidate", "candidates", "match", "bestMatch", "matchesInvoice",
			"againstInvoice", "shortOrOver", "shortOver", "difference",
		},
		Reads: slices.Concat(
			boardcatalog.Read(
				permission.ResourceBankReceipt, "amountMinor", "referenceNumber", "exceptionReason",
			),
			boardcatalog.Read(permission.ResourceInvoice, "number", "balanceDue"),
			boardcatalog.Read(permission.ResourceCustomerPayment, "referenceNumber", "amountMinor"),
		),
		Words:   "against the open item it likely pays: exact, short or over",
		Label:   "Against its best candidate",
		Total:   "Total received",
		Largest: "Largest receipt",
		Heading: "Against",
	},
}
