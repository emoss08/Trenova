package agentquerytoolservice

import "github.com/emoss08/trenova/internal/modules/identity/domain/permission"

const (
	toolInvoiceList          = "invoice_list"
	toolInvoiceGet           = "invoice_get"
	toolInvoiceGetAdjustment = "invoice_get_adjustment"
	toolInvoiceListDisputes  = "invoice_list_disputes"
	toolCustomerPaymentList  = "receivable_list_customer_payments"
	toolOrderList            = "order_list"

	pluralInvoices         = "invoices"
	pluralInvoiceDisputes  = "invoice disputes"
	pluralCustomerPayments = "customer payments"

	invoicesSummary = "List invoices narrowed by customer, status, whether they are paid or " +
		"disputed, the amount, or the invoice and due dates. It answers what is unpaid, " +
		"overdue or in dispute. Totals are withheld when your data access does not reach them."
	invoiceDisputesSummary = "List the dispute cases on invoices, open and closed, each " +
		"invoice's newest first: the reason and how each was resolved, with the amount in " +
		"dispute and the notes where your data access reaches them. Find disputed invoices " +
		"with invoice_list filtered to disputeStatus Disputed."
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
	invoiceAdjustmentGetSummary = "Retrieve one invoice adjustment by id: its kind, status " +
		"and approval, the invoices it made, its supporting documents and each line it " +
		"credits or rebills against the invoice's lines. Use invoice_list_adjustments first " +
		"to find one. Its reasons, amounts and quantities are withheld when your data access " +
		"does not reach them."

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
	fieldShipmentID        = "shipmentId"
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

var customerReference = idReference{
	resource: permission.ResourceCustomer,
	what:     "The customer",
	source:   toolCustomerList,
}

// billingReadSpecs are the invoice, dispute and customer payment lists, read
// through the Desk's own operations.
func billingReadSpecs() []graphQLListSpec {
	return []graphQLListSpec{
		{
			name:         toolInvoiceList,
			entityPlural: pluralInvoices,
			summary:      invoicesSummary,
			searchTerms:  []string{"last invoice", "latest invoice", "overdue invoices"},
			resource:     permission.ResourceInvoice,
			references: map[string]idReference{
				paramCustomerID: customerReference,
				fieldShipmentID: {
					resource: permission.ResourceShipment,
					what:     "The shipment billed",
					source:   shipmentListName,
				},
				fieldOrderID: {
					resource: permission.ResourceOrder,
					what:     "The order billed",
					source:   toolOrderList,
				},
			},
			notes: map[string]string{
				fieldInvoiceStatus:     noteInvoiceStatus,
				fieldSettlementStatus:  noteSettlementStatus,
				fieldInvoiceBillType:   noteInvoiceBillType,
				fieldInvoiceDueDateTo:  noteInvoiceOverdue,
				fieldInvoiceBillToName: noteInvoiceBillTo,
			},
		},
		{
			name:         toolCustomerPaymentList,
			entityPlural: pluralCustomerPayments,
			summary:      customerPaymentsSummary,
			queryNote:    paymentsQueryNote,
			resource:     permission.ResourceCustomerPayment,
			references:   map[string]idReference{paramCustomerID: customerReference},
			defaults:     map[string]any{fieldInvoiceStatus: []any{"Posted"}},
			notes:        map[string]string{fieldInvoiceStatus: notePaymentStatus},
		},
		{
			name:         toolInvoiceListDisputes,
			entityPlural: pluralInvoiceDisputes,
			summary:      invoiceDisputesSummary,
			resource:     permission.ResourceInvoiceDispute,
			references: map[string]idReference{paramReceivableInvoices: {
				resource: permission.ResourceInvoice,
				what:     "The invoices",
				source:   toolInvoiceList,
			}},
		},
	}
}

// invoiceGetSpec reads invoices by id through the Desk's invoice detail, one
// read per invoice.
func invoiceGetSpec() getSpec {
	return getSpec{
		name:        toolInvoiceGet,
		entity:      "invoice",
		searchTerms: []string{"invoices"},
		resource:    permission.ResourceInvoice,
		summary:     invoiceGetSummary,
		paramName:   paramReceivableInvoices,
		idSource:    "from invoice_list, receivable_list_open_items or the page you are on",
		batch:       invoiceGetBatch,
		shape:       boundInvoiceLines,
	}
}

// invoiceAdjustmentGetSpec reads one adjustment through the Desk's card.
func invoiceAdjustmentGetSpec() getSpec {
	return getSpec{
		name:      toolInvoiceGetAdjustment,
		entity:    "invoice adjustment",
		resource:  permission.ResourceInvoice,
		kinds:     []permission.RecordKind{permission.KindInvoiceAdjustment},
		summary:   invoiceAdjustmentGetSummary,
		paramName: paramAdjustmentID,
		idSource:  "from invoice_list_adjustments",
	}
}

// boundInvoiceLines keeps the first lines of an invoice, fewer when it is
// read beside others, and says how many it has and whether some were left
// out, as the repository tool did.
func boundInvoiceLines(record map[string]any, several bool) {
	lines, _ := record[fieldInvoiceLines].([]any)
	record[fieldLineCount] = len(lines)
	most := invoiceLinesShown
	if several {
		most = invoiceLinesShownEach
	}
	if len(lines) > most {
		record[fieldInvoiceLines] = lines[:most]
		record[fieldLinesTruncated] = true
	}
}
