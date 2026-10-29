package tendernegotiation

import (
	"github.com/emoss08/trenova/internal/modules/operations/app/tendereffects"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/shared/pulid"
)

// ResponsePreview is what recording a carrier's answer would do: the tender
// and the answered offer as they stand, the tender with its offers as the
// answer leaves them, and, after a decline, the offer the tender sends next
// or that nobody is left to ask. Nothing in it has been saved.
type ResponsePreview struct {
	Tender      *tender.Tender
	Offer       *tender.TenderOffer
	TenderAfter *tender.Tender
	NextOffer   *tender.TenderOffer
	Exhausted   bool
}

// AcceptSettlements is what an acceptance does to the rest of the plan: an
// offer not yet sent is skipped, and one a carrier still holds is superseded.
var AcceptSettlements = []struct {
	From tender.OfferStatus
	To   tender.OfferStatus
}{
	{From: tender.OfferStatusPending, To: tender.OfferStatusSkipped},
	{From: tender.OfferStatusSent, To: tender.OfferStatusSuperseded},
}

// AcceptedTender is the tender as an acceptance leaves it: accepted on the
// answered offer, with the rest of the plan settled as FinalizeAccepted
// settles it.
func AcceptedTender(
	entity *tender.Tender,
	offerID pulid.ID,
	source tender.ResponseSource,
	now int64,
) *tender.Tender {
	accepted := tendereffects.CopyTender(entity, func(offer *tender.TenderOffer) {
		if offer.ID == offerID {
			offer.Status = tender.OfferStatusAccepted
			offer.RespondedAt = &now
			offer.ResponseSource = source
			return
		}
		offer.Status = settledOfferStatus(offer.Status)
	})
	accepted.Status = tender.StatusAccepted
	accepted.AcceptedOfferID = &offerID
	accepted.AcceptedAt = &now

	return accepted
}

// NextOfferAfterDecline is the offer a sequential tender sends once the one it
// is waiting on is declined: the first offer in rank order not yet sent. A
// broadcast tender sends nothing further.
func NextOfferAfterDecline(entity *tender.Tender) *tender.TenderOffer {
	if !entity.Mode.IsSequential() {
		return nil
	}

	var next *tender.TenderOffer
	for _, offer := range entity.Offers {
		if offer == nil || offer.Status != tender.OfferStatusPending {
			continue
		}
		if next == nil || offer.Rank < next.Rank {
			next = offer
		}
	}

	return next
}

func HasOutstandingOffer(entity *tender.Tender) bool {
	for _, offer := range entity.Offers {
		if offer != nil && offer.Status == tender.OfferStatusSent {
			return true
		}
	}

	return false
}

func settledOfferStatus(status tender.OfferStatus) tender.OfferStatus {
	for _, settlement := range AcceptSettlements {
		if settlement.From == status {
			return settlement.To
		}
	}

	return status
}
