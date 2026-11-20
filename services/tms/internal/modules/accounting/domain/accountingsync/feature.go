package accountingsync

import "github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"

func (t SyncObjectType) Feature() platformcatalog.FeatureKey {
	switch t {
	case SyncObjectCustomer, SyncObjectInvoice, SyncObjectCreditMemo, SyncObjectDebitMemo:
		return platformcatalog.FeatureAccountingIntegration
	case SyncObjectCustomerPayment,
		SyncObjectCreditApplication,
		SyncObjectCarrierVendor,
		SyncObjectDriverVendor,
		SyncObjectCarrierBill,
		SyncObjectCarrierBillPay,
		SyncObjectDriverBill,
		SyncObjectDriverBillPay,
		SyncObjectJournalEntry,
		SyncObjectJournalSummary,
		DriftObjectGLAccount:
		return platformcatalog.FeatureAccounting
	default:
		return platformcatalog.FeatureAccounting
	}
}
