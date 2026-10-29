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
	"additional_charges":      backdater(shipmentdb.AdditionalCharges.Handle),
	"assignments":             backdater(shipmentdb.Assignments.Handle),
	"billing_queue_events":    backdater(billingqueueeventdb.ItemEvents.Handle),
	"billing_queue_items":     backdater(billingqueuedb.BillingQueueItems.Handle),
	"carrier_assignments":     backdater(shipmentdb.CarrierAssignments.Handle),
	"carrier_cost_events":     backdater(carriersettlementdb.CostEvents.Handle),
	"carrier_edi_channels":    backdater(carrierdb.CarrierEDIChannels.Handle),
	"carrier_invoice_matches": backdater(carriersettlementdb.InvoiceMatches.Handle),
	"carrier_ledger_entries":  backdater(carriersettlementdb.LedgerEntries.Handle),
	"carrier_settlement_batches": backdater(
		carriersettlementdb.CarrierSettlementBatches.Handle,
	),
	"carrier_settlement_lines": backdater(
		carriersettlementdb.CarrierSettlementLines.Handle,
	),
	"carrier_settlements":               backdater(carriersettlementdb.CarrierSettlements.Handle),
	"check_calls":                       backdater(checkcalldb.CheckCalls.Handle),
	"credit_memo_applications":          backdater(customerpaymentdb.CreditMemoApplications.Handle),
	"customer_billing_profiles":         backdater(customerdb.CustomerBillingProfiles.Handle),
	"customer_email_profiles":           backdater(customerdb.CustomerEmailProfiles.Handle),
	"customer_ledger_entries":           backdater(customerledgerdb.CustomerLedgerEntries.Handle),
	"customer_payment_applications":     backdater(customerpaymentdb.Applications.Handle),
	"customer_payments":                 backdater(customerpaymentdb.Payments.Handle),
	"customers":                         backdater(customerdb.Customers.Handle),
	"driver_expenses":                   backdater(payexpensedb.Expenses.Handle),
	"driver_pay_events":                 backdater(driversettlementdb.PayEvents.Handle),
	"driver_settlement_batches":         backdater(driversettlementdb.SettlementBatches.Handle),
	"driver_settlement_lines":           backdater(driversettlementdb.SettlementLines.Handle),
	"driver_settlements":                backdater(driversettlementdb.Settlements.Handle),
	"edi_carrier_invoices":              backdater(edicarrierinvoicedb.CarrierInvoices.Handle),
	"edi_messages":                      backdater(edimessagedb.EDIMessages.Handle),
	"edi_partners":                      backdater(edidb.EDIPartners.Handle),
	"escrow_accounts":                   backdater(payescrowdb.EscrowAccounts.Handle),
	"escrow_transactions":               backdater(payescrowdb.EscrowTransactions.Handle),
	"factoring_agreements":              backdater(factoringdb.Agreements.Handle),
	"factoring_companies":               backdater(factoringcompanydb.FactoringCompanies.Handle),
	"factoring_fee_tiers":               backdater(factoringdb.FeeTiers.Handle),
	"factoring_remittance_lines":        backdater(factoringremittancedb.RemittanceLines.Handle),
	"factoring_remittances":             backdater(factoringremittancedb.Remittances.Handle),
	"factoring_submission_items":        backdater(factoringsubmissiondb.SubmissionItems.Handle),
	"factoring_submissions":             backdater(factoringsubmissiondb.Submissions.Handle),
	"factoring_transports":              backdater(factoringsubmissiondb.Transports.Handle),
	"fuel_cards":                        backdater(fuelpurchasedb.FuelCards.Handle),
	"fuel_purchase_import_rows":         backdater(fuelpurchasedb.ImportRows.Handle),
	"fuel_purchases":                    backdater(fuelpurchasedb.FuelPurchases.Handle),
	"gl_accounts":                       backdater(glaccountdb.GLAccounts.Handle),
	"ifta_jurisdiction_mileage_entries": backdater(iftadb.JurisdictionMileageEntries.Handle),
	"ifta_returns":                      backdater(iftadb.Returns.Handle),
	"invoice_adjustment_batch_items": backdater(
		invoiceadjustmentdb.InvoiceAdjustmentBatchItems.Handle,
	),
	"invoice_adjustment_batches": backdater(
		invoiceadjustmentdb.InvoiceAdjustmentBatches.Handle,
	),
	"invoice_adjustment_lines": backdater(
		invoiceadjustmentdb.InvoiceAdjustmentLines.Handle,
	),
	"invoice_adjustment_snapshots": backdater(
		invoiceadjustmentdb.InvoiceAdjustmentSnapshots.Handle,
	),
	"invoice_adjustments": backdater(invoiceadjustmentdb.InvoiceAdjustments.Handle),
	"invoice_correction_groups": backdater(
		invoiceadjustmentdb.InvoiceAdjustmentCorrectionGroups.Handle,
	),
	"invoice_lines": backdater(invoicedb.InvoiceLines.Handle),
	"invoice_reconciliation_exceptions": backdater(
		invoiceadjustmentdb.InvoiceAdjustmentReconciliationExceptions.Handle,
	),
	"invoices":             backdater(invoicedb.Invoices.Handle),
	"journal_batches":      backdater(journalentrydb.JournalBatches.Handle),
	"journal_entries":      backdater(journalentrydb.JournalEntries.Handle),
	"journal_entry_lines":  backdater(journalentrydb.JournalEntryLines.Handle),
	"orders":               backdater(orderdb.Orders.Handle),
	"pay_advances":         backdater(payadvancedb.PayAdvances.Handle),
	"rate_confirmations":   backdater(rateconfirmationdb.RateConfirmations.Handle),
	"recurring_deductions": backdater(recurringpaydb.RecurringDeductions.Handle),
	"recurring_earnings":   backdater(recurringpaydb.RecurringEarnings.Handle),
	"settlement_disputes":  backdater(driversettlementdb.Disputes.Handle),
	"shipment_comment_acknowledgments": backdater(
		shipmentdb.ShipmentCommentAcknowledgments.Handle,
	),
	"shipment_comment_mentions": backdater(shipmentdb.ShipmentCommentMentions.Handle),
	"shipment_comments":         backdater(shipmentdb.ShipmentComments.Handle),
	"shipment_commodities":      backdater(shipmentdb.ShipmentCommodities.Handle),
	"shipment_holds":            backdater(shipmentholddb.ShipmentHolds.Handle),
	"shipment_moves":            backdater(shipmentdb.ShipmentMoves.Handle),
	"shipments":                 backdater(shipmentdb.Shipments.Handle),
	"stops":                     backdater(shipmentdb.Stops.Handle),
	"worker_pay_assignments":    backdater(payassignmentdb.WorkerPayAssignments.Handle),
	"worker_profiles":           backdater(workerdb.WorkerProfiles.Handle),
	"worker_pto":                backdater(workerdb.WorkerPTOs.Handle),
	"workers":                   backdater(workerdb.Workers.Handle),
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
