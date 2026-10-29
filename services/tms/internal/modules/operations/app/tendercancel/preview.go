package tendercancel

import (
	"github.com/emoss08/trenova/internal/modules/operations/app/tendereffects"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/pkg/pagination"
)

// CancelPreview is a tender before and as withdrawing it would leave it, its
// offers included. Nothing in it has been saved.
type CancelPreview struct {
	Before *tender.Tender
	After  *tender.Tender
}

// withdrawnTender is the tender as withdrawing it leaves it: canceled, with
// every offer moved the way offerWithdrawals moves it. The stored tender and
// its offers are left as they are.
func withdrawnTender(
	entity *tender.Tender,
	tenantInfo pagination.TenantInfo,
	reason string,
	now int64,
) *tender.Tender {
	canceled := tendereffects.CopyTender(entity, func(offer *tender.TenderOffer) {
		offer.Status, _ = WithdrawnOfferStatus(offer.Status)
	})
	canceled.Status = tender.StatusCanceled
	canceled.CancellationReason = reason
	canceled.CanceledAt = &now
	canceled.CanceledByID = tendereffects.UserIDPtr(tenantInfo)

	return canceled
}
