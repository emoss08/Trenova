package tender

import (
	"slices"

	"github.com/emoss08/trenova/shared/pulid"
)

// MaxBounces bounds how often one tender can be walked back and sent on, so a
// lane nobody will keep cannot cycle the same carriers forever.
const MaxBounces = int16(100)

// CanBounce reports whether the tender holds an acceptance a carrier can walk
// back: it was accepted, or accepted and parked for review, on an offer that
// still stands as accepted.
func (t *Tender) CanBounce() (*TenderOffer, bool) {
	if t == nil || t.AcceptedOfferID == nil {
		return nil, false
	}
	if t.Status != StatusAccepted && t.Status != StatusNeedsReview {
		return nil, false
	}
	offer := t.FindOffer(*t.AcceptedOfferID)
	if offer == nil || offer.Status != OfferStatusAccepted {
		return nil, false
	}
	return offer, true
}

// ReofferPlan is what a tender sends next once the carrier holding revoked
// walks the acceptance back: a fresh offer for every carrier the acceptance cut
// short, ranked as before.
//
// An acceptance skips the ranks it never reached and supersedes the offers
// other carriers still held. Those are the carriers the tender would have gone
// on to, so they are asked again. An offer skipped because its carrier became
// ineligible carries the reason and is left alone, as is any offer already
// asked again in an earlier round, and no carrier that has walked an
// acceptance back on this tender is asked again.
//
// The offers are new rows rather than the old ones reopened: delivery is
// idempotent per offer, so a reopened offer would never reach the carrier.
func (t *Tender) ReofferPlan(revoked *TenderOffer) []*TenderOffer {
	if t == nil || revoked == nil {
		return nil
	}

	reoffered := make(map[pulid.ID]struct{}, len(t.Offers))
	walkedBack := map[pulid.ID]struct{}{revoked.CarrierID: {}}
	for _, offer := range t.Offers {
		if offer == nil {
			continue
		}
		if offer.ReofferedFromID != nil {
			reoffered[*offer.ReofferedFromID] = struct{}{}
		}
		if offer.Status == OfferStatusRevoked {
			walkedBack[offer.CarrierID] = struct{}{}
		}
	}

	plan := make([]*TenderOffer, 0, len(t.Offers))
	planned := make(map[pulid.ID]struct{}, len(t.Offers))
	for _, offer := range t.Offers {
		if offer == nil || offer.ID == revoked.ID || !cutShortByAcceptance(offer, revoked) {
			continue
		}
		if _, done := reoffered[offer.ID]; done {
			continue
		}
		if _, out := walkedBack[offer.CarrierID]; out {
			continue
		}
		if _, dup := planned[offer.CarrierID]; dup {
			continue
		}
		planned[offer.CarrierID] = struct{}{}
		plan = append(plan, offer.reoffer())
	}

	slices.SortStableFunc(plan, func(a, b *TenderOffer) int {
		return int(a.Rank) - int(b.Rank)
	})
	return plan
}

func cutShortByAcceptance(offer, revoked *TenderOffer) bool {
	switch offer.Status {
	case OfferStatusSuperseded:
		return true
	case OfferStatusSkipped:
		return offer.SentAt == nil && offer.DeclineReason == "" && offer.Rank > revoked.Rank
	case OfferStatusPending,
		OfferStatusSent,
		OfferStatusAccepted,
		OfferStatusDeclined,
		OfferStatusExpired,
		OfferStatusWithdrawn,
		OfferStatusDeliveryFailed,
		OfferStatusCountered,
		OfferStatusCounterRejected,
		OfferStatusRevoked:
		return false
	default:
		return false
	}
}

func (to *TenderOffer) reoffer() *TenderOffer {
	from := to.ID
	return &TenderOffer{
		BusinessUnitID:  to.BusinessUnitID,
		OrganizationID:  to.OrganizationID,
		TenderID:        to.TenderID,
		CarrierID:       to.CarrierID,
		Rank:            to.Rank,
		RateMethod:      to.RateMethod,
		Rate:            to.Rate,
		OfferTTLSeconds: to.OfferTTLSeconds,
		Channel:         to.Channel,
		Status:          OfferStatusPending,
		RecipientEmail:  to.RecipientEmail,
		EDIPartnerID:    to.EDIPartnerID,
		ReofferedFromID: &from,
	}
}
