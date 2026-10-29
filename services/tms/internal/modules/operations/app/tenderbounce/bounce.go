package tenderbounce

import (
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/operations/app/tendereffects"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const maxBounceReasonLength = 500

// BounceTenderRequest records a carrier backing out of a tender it accepted.
type BounceTenderRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	TenderID   pulid.ID              `json:"tenderId"`
	// OfferID names the tender by the offer the carrier accepted, for a caller
	// that holds the offer rather than the tender. It must be that tender's
	// accepted offer.
	OfferID pulid.ID              `json:"offerId"`
	Reason  string                `json:"reason"`
	Source  tender.ResponseSource `json:"source"`
}

func (r *BounceTenderRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if r.TenderID.IsNil() && r.OfferID.IsNil() {
		multiErr.Add("tenderId", errortypes.ErrRequired, "Tender is required")
	}
	reason := strings.TrimSpace(r.Reason)
	switch {
	case reason == "":
		multiErr.Add("reason", errortypes.ErrRequired, "Say why the carrier backed out")
	case utf8.RuneCountInString(reason) > maxBounceReasonLength:
		multiErr.Add("reason", errortypes.ErrInvalid, "Reason must be at most 500 characters")
	}
	if r.Source != "" && r.Source != tender.ResponseSourceManual &&
		r.Source != tender.ResponseSourceEDI {
		multiErr.Add("source", errortypes.ErrInvalid, "Bounce source is invalid")
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

// BouncePreview is what bouncing a tender would do: the tender as it stands
// and as the bounce leaves it, the offers it sends next, and whether the
// carrier comes off the move. Nothing in it has been saved.
type BouncePreview struct {
	Before      *tender.Tender
	After       *tender.Tender
	Revoked     *tender.TenderOffer
	Reoffers    []*tender.TenderOffer
	Unassigns   bool
	WillExhaust bool
}

type bouncePlan struct {
	entity     *tender.Tender
	revoked    *tender.TenderOffer
	assignment *shipment.CarrierAssignment
	reoffers   []*tender.TenderOffer
}

// bouncedTender is the tender as Bounce leaves it: live again on the next
// round, the acceptance revoked and the re-offers waiting. The stored tender
// is left as it is.
func bouncedTender(plan *bouncePlan, req *BounceTenderRequest, now int64) *tender.Tender {
	reason := strings.TrimSpace(req.Reason)
	out := tendereffects.CopyTender(plan.entity, func(offer *tender.TenderOffer) {
		if offer.ID != plan.revoked.ID {
			return
		}
		offer.Status = tender.OfferStatusRevoked
		offer.RevokedAt = &now
		offer.RevokeReason = reason
		offer.RevokedByID = tendereffects.UserIDPtr(req.TenantInfo)
	})
	out.Status = tender.StatusActive
	out.BounceCount++
	out.LastBouncedAt = &now
	out.AcceptedOfferID = nil
	out.AcceptedAt = nil
	out.CurrentRank = 0
	for _, offer := range plan.reoffers {
		copied := *offer
		out.Offers = append(out.Offers, &copied)
	}
	return out
}
