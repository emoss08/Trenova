package tendercancel

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
)

// offerWithdrawals is what withdrawing a tender does to its offers: an offer
// a carrier holds is withdrawn, and one not yet sent is skipped.
var offerWithdrawals = []struct {
	from tender.OfferStatus
	to   tender.OfferStatus
}{
	{from: tender.OfferStatusSent, to: tender.OfferStatusWithdrawn},
	{from: tender.OfferStatusPending, to: tender.OfferStatusSkipped},
}

// WithdrawnOfferStatus is the status withdrawing a tender leaves an offer
// in, and whether it moves at all.
func WithdrawnOfferStatus(status tender.OfferStatus) (tender.OfferStatus, bool) {
	for _, withdrawal := range offerWithdrawals {
		if withdrawal.from == status {
			return withdrawal.to, true
		}
	}

	return status, false
}
