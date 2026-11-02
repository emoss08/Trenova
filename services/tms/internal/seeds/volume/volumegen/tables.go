package volumegen

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount/glaccountdb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry/journalentrydb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueueevent/billingqueueeventdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger/customerledgerdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment/customerpaymentdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring/factoringdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringremittance/factoringremittancedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission/factoringsubmissiondb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment/invoiceadjustmentdb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edicarrierinvoice/edicarrierinvoicedb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage/edimessagedb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/checkcall/checkcalldb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order/orderdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmenthold/shipmentholddb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany/factoringcompanydb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateconfirmation/rateconfirmationdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement/carriersettlementdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement/driversettlementdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelpurchase/fuelpurchasedb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta/iftadb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payadvance/payadvancedb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payassignment/payassignmentdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payescrow/payescrowdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payexpense/payexpensedb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/recurringpay/recurringpaydb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
)

type tableBackdate func(ctx context.Context, db *keel.DB, stamps []Stamp, createdOnly bool) error

func backdater[T any](h keel.Handle[*T]) tableBackdate {
	return func(ctx context.Context, db *keel.DB, stamps []Stamp, createdOnly bool) error {
		return BackdateColumns(ctx, db, h, stamps, createdOnly)
	}
}

var tableBackdates = map[string]tableBackdate{
	"additional_charges":      backdater(shipmentdb.AdditionalCharges),
	"assignments":             backdater(shipmentdb.Assignments),
	"billing_queue_events":    backdater(billingqueueeventdb.ItemEvents),
	"billing_queue_items":     backdater(billingqueuedb.BillingQueueItems),
	"carrier_assignments":     backdater(shipmentdb.CarrierAssignments),
	"carrier_cost_events":     backdater(carriersettlementdb.CostEvents),
	"carrier_edi_channels":    backdater(carrierdb.CarrierEDIChannels),
	"carrier_invoice_matches": backdater(carriersettlementdb.InvoiceMatches),
	"carrier_ledger_entries":  backdater(carriersettlementdb.LedgerEntries),
	"carrier_settlement_batches": backdater(
		carriersettlementdb.CarrierSettlementBatches,
	),
	"carrier_settlement_lines": backdater(
		carriersettlementdb.CarrierSettlementLines,
	),
	"carrier_settlements":               backdater(carriersettlementdb.CarrierSettlements),
	"check_calls":                       backdater(checkcalldb.CheckCalls),
	"credit_memo_applications":          backdater(customerpaymentdb.CreditMemoApplications),
	"customer_billing_profiles":         backdater(customerdb.CustomerBillingProfiles),
	"customer_email_profiles":           backdater(customerdb.CustomerEmailProfiles),
	"customer_ledger_entries":           backdater(customerledgerdb.CustomerLedgerEntries),
	"customer_payment_applications":     backdater(customerpaymentdb.Applications),
	"customer_payments":                 backdater(customerpaymentdb.Payments),
	"customers":                         backdater(customerdb.Customers),
	"driver_expenses":                   backdater(payexpensedb.Expenses),
	"driver_pay_events":                 backdater(driversettlementdb.PayEvents),
	"driver_settlement_batches":         backdater(driversettlementdb.SettlementBatches),
	"driver_settlement_lines":           backdater(driversettlementdb.SettlementLines),
	"driver_settlements":                backdater(driversettlementdb.Settlements),
	"edi_carrier_invoices":              backdater(edicarrierinvoicedb.CarrierInvoices),
	"edi_messages":                      backdater(edimessagedb.EDIMessages),
	"edi_partners":                      backdater(edidb.EDIPartners),
	"escrow_accounts":                   backdater(payescrowdb.EscrowAccounts),
	"escrow_transactions":               backdater(payescrowdb.EscrowTransactions),
	"factoring_agreements":              backdater(factoringdb.Agreements),
	"factoring_companies":               backdater(factoringcompanydb.FactoringCompanies),
	"factoring_fee_tiers":               backdater(factoringdb.FeeTiers),
	"factoring_remittance_lines":        backdater(factoringremittancedb.RemittanceLines),
	"factoring_remittances":             backdater(factoringremittancedb.Remittances),
	"factoring_submission_items":        backdater(factoringsubmissiondb.SubmissionItems),
	"factoring_submissions":             backdater(factoringsubmissiondb.Submissions),
	"factoring_transports":              backdater(factoringsubmissiondb.Transports),
	"fuel_cards":                        backdater(fuelpurchasedb.FuelCards),
	"fuel_purchase_import_rows":         backdater(fuelpurchasedb.ImportRows),
	"fuel_purchases":                    backdater(fuelpurchasedb.FuelPurchases),
	"gl_accounts":                       backdater(glaccountdb.GLAccounts),
	"ifta_jurisdiction_mileage_entries": backdater(iftadb.JurisdictionMileageEntries),
	"ifta_returns":                      backdater(iftadb.Returns),
	"invoice_adjustment_batch_items": backdater(
		invoiceadjustmentdb.InvoiceAdjustmentBatchItems,
	),
	"invoice_adjustment_batches": backdater(
		invoiceadjustmentdb.InvoiceAdjustmentBatches,
	),
	"invoice_adjustment_lines": backdater(
		invoiceadjustmentdb.InvoiceAdjustmentLines,
	),
	"invoice_adjustment_snapshots": backdater(
		invoiceadjustmentdb.InvoiceAdjustmentSnapshots,
	),
	"invoice_adjustments": backdater(invoiceadjustmentdb.InvoiceAdjustments),
	"invoice_correction_groups": backdater(
		invoiceadjustmentdb.InvoiceAdjustmentCorrectionGroups,
	),
	"invoice_lines": backdater(invoicedb.InvoiceLines),
	"invoice_reconciliation_exceptions": backdater(
		invoiceadjustmentdb.InvoiceAdjustmentReconciliationExceptions,
	),
	"invoices":             backdater(invoicedb.Invoices),
	"journal_batches":      backdater(journalentrydb.JournalBatches),
	"journal_entries":      backdater(journalentrydb.JournalEntries),
	"journal_entry_lines":  backdater(journalentrydb.JournalEntryLines),
	"orders":               backdater(orderdb.Orders),
	"pay_advances":         backdater(payadvancedb.PayAdvances),
	"rate_confirmations":   backdater(rateconfirmationdb.RateConfirmations),
	"recurring_deductions": backdater(recurringpaydb.RecurringDeductions),
	"recurring_earnings":   backdater(recurringpaydb.RecurringEarnings),
	"settlement_disputes":  backdater(driversettlementdb.Disputes),
	"shipment_comment_acknowledgments": backdater(
		shipmentdb.ShipmentCommentAcknowledgments,
	),
	"shipment_comment_mentions": backdater(shipmentdb.ShipmentCommentMentions),
	"shipment_comments":         backdater(shipmentdb.ShipmentComments),
	"shipment_commodities":      backdater(shipmentdb.ShipmentCommodities),
	"shipment_holds":            backdater(shipmentholddb.ShipmentHolds),
	"shipment_moves":            backdater(shipmentdb.ShipmentMoves),
	"shipments":                 backdater(shipmentdb.Shipments),
	"stops":                     backdater(shipmentdb.Stops),
	"worker_pay_assignments":    backdater(payassignmentdb.WorkerPayAssignments),
	"worker_profiles":           backdater(workerdb.WorkerProfiles),
	"worker_pto":                backdater(workerdb.WorkerPTOs),
	"workers":                   backdater(workerdb.Workers),
}

func BackdateTable(ctx context.Context, db *keel.DB, table string, stamps []Stamp) error {
	return backdateNamed(ctx, db, table, stamps, false)
}

func BackdateTableCreated(ctx context.Context, db *keel.DB, table string, stamps []Stamp) error {
	return backdateNamed(ctx, db, table, stamps, true)
}

func backdateNamed(
	ctx context.Context,
	db *keel.DB,
	table string,
	stamps []Stamp,
	createdOnly bool,
) error {
	if len(stamps) == 0 {
		return nil
	}
	fn, ok := tableBackdates[table]
	if !ok {
		return fmt.Errorf("%w: %s", errUnknownTable, table)
	}

	return fn(ctx, db, stamps, createdOnly)
}
