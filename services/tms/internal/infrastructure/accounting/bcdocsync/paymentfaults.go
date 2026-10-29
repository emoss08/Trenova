package bcdocsync

import (
	"github.com/emoss08/trenova/internal/infrastructure/accounting/bcsession"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
)

func creditApplicationRefused() *accountingsync.SyncError {
	return &accountingsync.SyncError{
		Category: accountingsync.SyncErrorConfiguration,
		Code:     "credit-application",
		Message: bcsession.ProviderName + " applies a credit memo only when it is posted against " +
			"an invoice, so Trenova cannot apply or unapply an existing one",
		Resolution: "Apply the credit memo in " + bcsession.ProviderName +
			", then mark this record resolved",
	}
}

func unpostedJournal(code string) *accountingsync.SyncError {
	return &accountingsync.SyncError{
		Category: accountingsync.SyncErrorConfiguration,
		Code:     "journal-post",
		Message: bcsession.ProviderName + " would not post the " + code +
			" payment journal from Trenova, so the payment waits there unposted",
		Resolution: "Post the " + code + " payment journal in " + bcsession.ProviderName,
	}
}
