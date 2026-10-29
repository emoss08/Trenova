package tenderresponse

import (
	"errors"

	"github.com/emoss08/trenova/internal/modules/operations/app/tendereffects"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func isNoLongerAvailable(err error) bool {
	return errors.Is(err, ErrOfferNoLongerAvailable)
}

// declinedTender is the tender with the answered offer declined as
// DeclineOffer records it.
func declinedTender(
	entity *tender.Tender,
	offerID pulid.ID,
	req *tenderports.TenderResponseRequest,
	now int64,
) *tender.Tender {
	return tendereffects.CopyTender(entity, func(offer *tender.TenderOffer) {
		if offer.ID != offerID {
			return
		}
		offer.Status = tender.OfferStatusDeclined
		offer.RespondedAt = &now
		offer.ResponseSource = req.Source
		offer.DeclineReason = req.DeclineReason
	})
}

// counteredTender is the tender with the answered offer holding the
// carrier's counter, as RecordCounter records it.
func counteredTender(
	entity *tender.Tender,
	offerID pulid.ID,
	req *tenderports.TenderResponseRequest,
	now int64,
) *tender.Tender {
	return tendereffects.CopyTender(entity, func(offer *tender.TenderOffer) {
		if offer.ID != offerID {
			return
		}
		offer.Status = tender.OfferStatusCountered
		offer.RespondedAt = &now
		offer.ResponseSource = req.Source
		offer.CounterRate = decimal.NewNullDecimal(req.CounterRate)
		offer.CounterNote = req.CounterNote
		offer.CounteredAt = &now
	})
}
