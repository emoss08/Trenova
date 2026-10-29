package billingquerytools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/filtercatalog"
)

const (
	toolInvoiceList          = "invoice_list"
	toolInvoiceGet           = "invoice_get"
	toolInvoiceGetAdjustment = "invoice_get_adjustment"
	toolInvoiceListDisputes  = "invoice_list_disputes"
	toolCustomerPaymentList  = "receivable_list_customer_payments"

	entityInvoice = "invoice"
	statusPosted  = "Posted"

	pluralInvoices         = "invoices"
	pluralInvoiceDisputes  = "invoice disputes"
	pluralCustomerPayments = "customer payments"

	invoicesSummary = "List invoices narrowed by customer, status, whether they are paid or " +
		"disputed, the amount, or the invoice and due dates. It answers what is unpaid, " +
		"overdue or in dispute. Totals are withheld when your data access does not reach them."
	invoiceDisputesSummary = "List the dispute cases on the invoices you name, open and closed, " +
		"each invoice's newest first. Each says the reason and how it was resolved, with the " +
		"amount in dispute and the notes where your data access reaches them. Find disputed " +
		"invoices with invoice_list filtered to disputeStatus Disputed."
	customerPaymentsSummary = "List recorded customer payments, newest first, with the " +
		"amount, how much of it is applied to which invoices, the method and the reference. " +
		"Search by reference or memo text, narrow to one customer, or include reversed " +
		"payments. Amounts are in cents (amountMinor 12500 is 125.00); they and the memo " +
		"are withheld when your data access does not reach them. Use it to find the payment " +
		"a bank receipt represents when receivable_get_bank_receipt suggested nothing, and " +
		"to see whether a customer's payment is already recorded before posting another."
	invoiceGetSummary = "Retrieve invoices by id: one, or up to 50 in one call, for example " +
		"to check a batch of drafts before posting them. Each comes with its status, totals, " +
		"what is still owed, its payment, dispute and send state, and its line items; each " +
		"line's id is what an invoice adjustment names. Use invoice_list first when you have " +
		"a number or are looking for what is unpaid. An id that is not an invoice of this " +
		"organization is named in the note. Amounts, the memo and the void reason are left " +
		"out, and named in withheldByAccess, when your data access does not reach them."
	invoiceAdjustmentGetSummary = "Retrieve one invoice adjustment by id: its kind, status and " +
		"approval, the invoices it made and each line it credits or rebills. It names its " +
		"supporting documents. Use invoice_list_adjustments first to find one. Its reasons, " +
		"amounts and quantities are withheld when your data access does not reach them."

	noteInvoiceStatus    = "Where the invoice is in its own lifecycle, not whether it is paid."
	noteSettlementStatus = "This is the one to use for \"unpaid\" or \"outstanding\"."
	noteInvoiceBillType  = "CreditMemo and DebitMemo are memos; apply a posted CreditMemo with receivable_apply_credit_memo."
	noteInvoiceOverdue   = "Overdue is a due date before today with settlementStatus not Paid."
	noteInvoiceBillTo    = "Who the invoice is billed to."
	notePaymentStatus    = "Posted payments are read unless the call names a status."
	paymentsQueryNote    = "Words to look for in the reference number or the memo."

	fieldInvoiceStatus     = "status"
	fieldSettlementStatus  = "settlementStatus"
	fieldInvoiceBillType   = "billType"
	fieldInvoiceDueDateTo  = "dueDateTo"
	fieldInvoiceBillToName = "billToName"
	fieldOrderID           = "orderId"
	fieldInvoiceLines      = "lines"
	fieldLineCount         = "lineCount"
	fieldLinesTruncated    = "linesTruncated"

	// invoiceGetBatch is how many invoices one invoice_get reads.
	invoiceGetBatch = 50
	// invoiceLinesShown and invoiceLinesShownEach bound the lines an invoice
	// carries read alone and read beside others, as the repository tool did.
	invoiceLinesShown     = 100
	invoiceLinesShownEach = 25
)

// ReadSpecs are the invoice, dispute and customer payment lists, read
// through the Desk's own operations.
func ReadSpecs() []readspec.List {
	return []readspec.List{
		{
			Name:         toolInvoiceList,
			EntityPlural: pluralInvoices,
			Summary:      invoicesSummary,
			SearchTerms:  []string{"last invoice", "latest invoice", "overdue invoices"},
			Resource:     permission.ResourceInvoice,
			References: map[string]readspec.Reference{
				querytool.ParamCustomerID: readspec.CustomerReference,
				querytool.FieldShipmentID: {
					Resource: permission.ResourceShipment,
					What:     "The shipment billed",
					Source:   querytool.ShipmentListName,
				},
				fieldOrderID: {
					Resource: permission.ResourceOrder,
					What:     "The order billed",
					Source:   querytool.ToolOrderList,
				},
			},
			Notes: map[string]string{
				fieldInvoiceStatus:     noteInvoiceStatus,
				fieldSettlementStatus:  noteSettlementStatus,
				fieldInvoiceBillType:   noteInvoiceBillType,
				fieldInvoiceDueDateTo:  noteInvoiceOverdue,
				fieldInvoiceBillToName: noteInvoiceBillTo,
			},
		},
		{
			Name:         toolCustomerPaymentList,
			EntityPlural: pluralCustomerPayments,
			Summary:      customerPaymentsSummary,
			QueryNote:    paymentsQueryNote,
			Resource:     permission.ResourceCustomerPayment,
			References: map[string]readspec.Reference{
				querytool.ParamCustomerID: readspec.CustomerReference,
			},
			Defaults: map[string]any{fieldInvoiceStatus: []any{statusPosted}},
			Notes:    map[string]string{fieldInvoiceStatus: notePaymentStatus},
		},
		{
			Name:         toolInvoiceListDisputes,
			EntityPlural: pluralInvoiceDisputes,
			Summary:      invoiceDisputesSummary,
			Resource:     permission.ResourceInvoiceDispute,
			References: map[string]readspec.Reference{querytool.ParamReceivableInvoices: {
				Resource: permission.ResourceInvoice,
				What:     "The invoices",
				Source:   toolInvoiceList,
			}},
		},
	}
}

// InvoiceGetSpec reads invoices by id through the Desk's invoice detail, one
// read per invoice.
func InvoiceGetSpec() readspec.Get {
	return readspec.Get{
		Name:        toolInvoiceGet,
		Entity:      entityInvoice,
		SearchTerms: []string{pluralInvoices},
		Resource:    permission.ResourceInvoice,
		Summary:     invoiceGetSummary,
		ParamName:   querytool.ParamReceivableInvoices,
		IDSource:    "from invoice_list, receivable_list_open_items or the page you are on",
		Batch:       invoiceGetBatch,
		Shape:       countInvoiceLines,
		ShapeEach:   keepInvoiceLinesEach,
	}
}

// InvoiceAdjustmentGetSpec reads one adjustment through the Desk's card.
func InvoiceAdjustmentGetSpec() readspec.Get {
	return readspec.Get{
		Name:      toolInvoiceGetAdjustment,
		Entity:    "invoice adjustment",
		Resource:  permission.ResourceInvoice,
		Kinds:     []permission.RecordKind{permission.KindInvoiceAdjustment},
		Summary:   invoiceAdjustmentGetSummary,
		ParamName: querytool.ParamAdjustmentID,
		IDSource:  "from invoice_list_adjustments",
	}
}

// countInvoiceLines says how many lines an invoice has and keeps the first
// of them, as the repository tool did. Lines access withholds are not
// counted.
func countInvoiceLines(record map[string]any, withheld map[string]struct{}, _ filtercatalog.Clock) {
	if _, hidden := withheld[fieldInvoiceLines]; hidden {
		return
	}
	lines, _ := record[fieldInvoiceLines].([]any)
	record[fieldLineCount] = len(lines)
	keepInvoiceLines(record, invoiceLinesShown)
}

// keepInvoiceLinesEach keeps fewer lines of an invoice read beside others.
func keepInvoiceLinesEach(record map[string]any) {
	keepInvoiceLines(record, invoiceLinesShownEach)
}

// keepInvoiceLines keeps an invoice's first lines and says whether some were
// left out.
func keepInvoiceLines(record map[string]any, most int) {
	lines, _ := record[fieldInvoiceLines].([]any)
	if len(lines) > most {
		record[fieldInvoiceLines] = lines[:most]
		record[fieldLinesTruncated] = true
	}
}
