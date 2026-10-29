package nsdocsync

import (
	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsbinding"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/shared/netsuiteerrors"
)

const (
	ReversePayment = "Reverse the payment in NetSuite, then mark this record resolved"
	unapplyPayment = "Unapply the payment in NetSuite, then retry"
)

func DeleteRefused(err error) bool {
	if netsuiteerrors.IsClosedPeriod(err) {
		return true
	}
	return nsbinding.ErrorCode(err) == netsuiteerrors.CodeUserError &&
		!netsuiteerrors.IsPermission(err)
}

func unmappedItem(purpose string) *accountingsync.SyncError {
	return &accountingsync.SyncError{
		Category:   accountingsync.SyncErrorMapping,
		Code:       "item",
		Message:    purpose + " has no " + nsbinding.ProviderName + " item mapped",
		Resolution: "Map it to a " + nsbinding.ProviderName + " item",
	}
}

func unmappedAccount(purpose string) *accountingsync.SyncError {
	return &accountingsync.SyncError{
		Category:   accountingsync.SyncErrorMapping,
		Code:       "account",
		Message:    purpose + " has no " + nsbinding.ProviderName + " account mapped",
		Resolution: "Map it to a " + nsbinding.ProviderName + " account",
	}
}

func negativeLine(purpose string) *accountingsync.SyncError {
	return &accountingsync.SyncError{
		Category: accountingsync.SyncErrorValidation,
		Code:     "negative-line",
		Message: purpose + " is negative, and " + nsbinding.ProviderName +
			" does not take a negative line on this document",
		Resolution: "Net the line into the others in Trenova, then retry",
	}
}

func paidConflict() *accountingsync.SyncError {
	return &accountingsync.SyncError{
		Category: accountingsync.SyncErrorConflict,
		Code:     "document-applied",
		Message: "A payment or credit is still applied to the document in " + nsbinding.ProviderName +
			", so Trenova cannot delete or reverse it",
		Resolution: unapplyPayment,
	}
}

func reversalRefused() *accountingsync.SyncError {
	return &accountingsync.SyncError{
		Category: accountingsync.SyncErrorConfiguration,
		Code:     "reverse-payment",
		Message: "The payment is in a closed or locked period in " + nsbinding.ProviderName +
			", which has no REST void, so Trenova cannot delete it",
		Resolution: ReversePayment,
	}
}

func creditReversalRefused(label string) *accountingsync.SyncError {
	return &accountingsync.SyncError{
		Category: accountingsync.SyncErrorConfiguration,
		Code:     "reverse-credit",
		Message: "The " + label + " is in a closed or locked period in " + nsbinding.ProviderName +
			", which refuses to delete it",
		Resolution: "Reverse the " + label + " in " + nsbinding.ProviderName +
			", then mark this record resolved",
	}
}

func creditApplicationUnknown() *accountingsync.SyncError {
	return &accountingsync.SyncError{
		Category: accountingsync.SyncErrorConfiguration,
		Code:     "credit-application",
		Message: "Trenova does not know which credit memo and invoice the application " +
			"joined in " + nsbinding.ProviderName,
		Resolution: "Unapply the credit memo in " + nsbinding.ProviderName +
			", then mark this record resolved",
	}
}
