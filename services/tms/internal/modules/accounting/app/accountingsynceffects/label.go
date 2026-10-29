package accountingsynceffects

import (
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
)

func DocumentLabel(objectType accountingsync.SyncObjectType, number string) string {
	var kind string
	switch objectType {
	case accountingsync.SyncObjectCreditMemo:
		kind = "Credit memo"
	case accountingsync.SyncObjectDebitMemo:
		kind = "Debit memo"
	case accountingsync.SyncObjectCustomerPayment:
		kind = "Payment"
	case accountingsync.SyncObjectCreditApplication:
		kind = "Credit application"
	case accountingsync.SyncObjectCustomer:
		kind = "Customer"
	case accountingsync.SyncObjectInvoice:
		kind = "Invoice"
	case accountingsync.SyncObjectCarrierVendor:
		kind = "Carrier"
	case accountingsync.SyncObjectDriverVendor:
		kind = "Driver"
	case accountingsync.SyncObjectCarrierBill:
		kind = "Carrier settlement"
	case accountingsync.SyncObjectDriverBill:
		kind = "Owner-operator settlement"
	case accountingsync.SyncObjectCarrierBillPay:
		kind = "Payment of carrier settlement"
	case accountingsync.SyncObjectDriverBillPay:
		kind = "Payment of owner-operator settlement"
	case accountingsync.SyncObjectJournalEntry:
		kind = "Journal entry"
	case accountingsync.SyncObjectJournalSummary:
		kind = "Journal entries for"
	case accountingsync.DriftObjectGLAccount:
		kind = "Document"
	default:
		kind = "Document"
	}
	if number == "" {
		return kind
	}
	return kind + " " + number
}
