package bcdocsync

import (
	"github.com/emoss08/trenova/internal/infrastructure/accounting/bcsession"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
)

const (
	reverseTransaction = "Reverse the payment's entries in Business Central " +
		"(Reverse transaction), then mark this record resolved"
)

func unmappedItem(purpose string) *accountingsync.SyncError {
	return &accountingsync.SyncError{
		Category:   accountingsync.SyncErrorMapping,
		Code:       "item",
		Message:    purpose + " has no " + bcsession.ProviderName + " item mapped",
		Resolution: "Map it to a " + bcsession.ProviderName + " item",
	}
}

func unmappedAccount(purpose string) *accountingsync.SyncError {
	return &accountingsync.SyncError{
		Category:   accountingsync.SyncErrorMapping,
		Code:       "account",
		Message:    purpose + " has no " + bcsession.ProviderName + " account mapped",
		Resolution: "Map it to a " + bcsession.ProviderName + " account",
	}
}

func negativeLine(purpose string) *accountingsync.SyncError {
	return &accountingsync.SyncError{
		Category: accountingsync.SyncErrorValidation,
		Code:     "negative-line",
		Message: purpose + " is negative, and " + bcsession.ProviderName +
			" does not take a negative line on this document",
		Resolution: "Net the line into the others in Trenova, then retry",
	}
}

func paidConflict() *accountingsync.SyncError {
	return &accountingsync.SyncError{
		Category: accountingsync.SyncErrorConflict,
		Code:     "document-paid",
		Message: "The document has a payment or credit applied in " + bcsession.ProviderName +
			", which refuses to cancel it",
		Resolution: "Unapply the payment in " + bcsession.ProviderName + ", then retry",
	}
}

func reversalRefused() *accountingsync.SyncError {
	return &accountingsync.SyncError{
		Category: accountingsync.SyncErrorConfiguration,
		Code:     "reverse-transaction",
		Message: bcsession.ProviderName + "'s API can neither unapply nor reverse a posted payment, " +
			"so Trenova cannot change or void it",
		Resolution: reverseTransaction,
	}
}

func billLinesUnknown() *accountingsync.SyncError {
	return &accountingsync.SyncError{
		Category: accountingsync.SyncErrorConfiguration,
		Code:     "bill-lines",
		Message: "Trenova does not know which accounts the bill was posted to, " +
			"so it cannot credit it in " + bcsession.ProviderName,
		Resolution: "Credit the bill with a purchase credit memo in " + bcsession.ProviderName +
			", then mark this record resolved",
	}
}
