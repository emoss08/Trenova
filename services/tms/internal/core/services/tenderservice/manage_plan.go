package tenderservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	portservices "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

// CancelPreview is a tender before and as withdrawing it would leave it, its
// offers included. Nothing in it has been saved.
type CancelPreview struct {
	Before *tender.Tender
	After  *tender.Tender
}

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

// acceptSettlements is what an acceptance does to the rest of the plan: an
// offer not yet sent is skipped, and one a carrier still holds is superseded.
var acceptSettlements = []struct {
	from tender.OfferStatus
	to   tender.OfferStatus
}{
	{from: tender.OfferStatusPending, to: tender.OfferStatusSkipped},
	{from: tender.OfferStatusSent, to: tender.OfferStatusSuperseded},
}

// PreviewCancel is what Cancel would do, from the same checks, without
// signalling the workflow or writing.
func (s *Service) PreviewCancel(
	ctx context.Context,
	req *CancelTenderRequest,
) (*CancelPreview, error) {
	entity, err := s.planCancel(ctx, req)
	if err != nil {
		return nil, err
	}

	return &CancelPreview{
		Before: entity,
		After:  withdrawnTender(entity, req.TenantInfo, req.Reason, timeutils.NowUnix()),
	}, nil
}

// PreviewResponse is what RecordResponse would lead to, from the same checks,
// without signalling the workflow or filing a late answer: the transitions
// the workflow makes when it lands the answer.
func (s *Service) PreviewResponse(
	ctx context.Context,
	req *portservices.TenderResponseRequest,
) (*ResponsePreview, error) {
	offer, err := s.planResponse(ctx, req)
	if err != nil {
		return nil, err
	}

	entity, err := s.repo.GetByID(ctx, repositories.GetTenderByIDRequest{
		TenantInfo:    req.TenantInfo,
		TenderID:      offer.TenderID,
		IncludeOffers: true,
	})
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	out := &ResponsePreview{Tender: entity, Offer: offer}
	if req.Action == tender.ResponseActionAccept {
		out.TenderAfter = acceptedTender(entity, offer.ID, req.Source, now)

		return out, nil
	}

	out.TenderAfter = declinedTender(entity, offer.ID, req, now)
	out.NextOffer = nextOfferAfterDecline(out.TenderAfter)
	if out.NextOffer == nil && !hasOutstandingOffer(out.TenderAfter) {
		out.Exhausted = true
		out.TenderAfter.Status = tender.StatusExhausted
		out.TenderAfter.ExhaustedAt = &now
	}

	return out, nil
}

// planCancel is everything Cancel decides before it acts: the tender, with
// its offers, is still live.
func (s *Service) planCancel(ctx context.Context, req *CancelTenderRequest) (*tender.Tender, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	entity, err := s.repo.GetByID(ctx, repositories.GetTenderByIDRequest{
		TenantInfo:    req.TenantInfo,
		TenderID:      req.TenderID,
		IncludeOffers: true,
	})
	if err != nil {
		return nil, err
	}
	if !entity.IsLive() {
		return nil, errortypes.NewBusinessError("Tender has already finished").
			WithParam("tenderId", entity.ID.String()).
			WithParam("status", entity.Status.String())
	}

	return entity, nil
}

// planResponse is everything RecordResponse decides before it signals: the
// answer is well formed and the offer can still take it. An offer that can
// no longer take it is returned with ErrOfferNoLongerAvailable, so the write
// can file the answer as late.
func (s *Service) planResponse(
	ctx context.Context,
	req *portservices.TenderResponseRequest,
) (*tender.TenderOffer, error) {
	if req.OfferID.IsNil() {
		return nil, errortypes.NewValidationError(
			"offerId", errortypes.ErrRequired, "Offer is required",
		)
	}
	if !req.Action.IsValid() {
		return nil, errortypes.NewValidationError(
			"action", errortypes.ErrInvalid, "Response action is invalid",
		)
	}
	if !req.Source.IsValid() {
		return nil, errortypes.NewValidationError(
			"source", errortypes.ErrInvalid, "Response source is invalid",
		)
	}
	if len(req.DeclineReason) > maxDeclineReasonLength {
		return nil, errortypes.NewValidationError(
			"declineReason",
			errortypes.ErrInvalid,
			"Decline reason must be at most 500 characters",
		)
	}

	offer, err := s.repo.GetOfferByID(ctx, repositories.GetTenderOfferByIDRequest{
		TenantInfo:    req.TenantInfo,
		OfferID:       req.OfferID,
		IncludeTender: true,
	})
	if err != nil {
		return nil, err
	}

	if offer.Tender == nil || offer.Tender.Status != tender.StatusActive ||
		offer.Status != tender.OfferStatusSent {
		return offer, ErrOfferNoLongerAvailable
	}

	return offer, nil
}

func isNoLongerAvailable(err error) bool {
	return errors.Is(err, ErrOfferNoLongerAvailable)
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
	canceled := copyTender(entity, func(offer *tender.TenderOffer) {
		offer.Status, _ = WithdrawnOfferStatus(offer.Status)
	})
	canceled.Status = tender.StatusCanceled
	canceled.CancellationReason = reason
	canceled.CanceledAt = &now
	canceled.CanceledByID = userIDPtr(tenantInfo)

	return canceled
}

// acceptedTender is the tender as an acceptance leaves it: accepted on the
// answered offer, with the rest of the plan settled as FinalizeAccepted
// settles it.
func acceptedTender(
	entity *tender.Tender,
	offerID pulid.ID,
	source tender.ResponseSource,
	now int64,
) *tender.Tender {
	accepted := copyTender(entity, func(offer *tender.TenderOffer) {
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

// declinedTender is the tender with the answered offer declined as
// DeclineOffer records it.
func declinedTender(
	entity *tender.Tender,
	offerID pulid.ID,
	req *portservices.TenderResponseRequest,
	now int64,
) *tender.Tender {
	return copyTender(entity, func(offer *tender.TenderOffer) {
		if offer.ID != offerID {
			return
		}
		offer.Status = tender.OfferStatusDeclined
		offer.RespondedAt = &now
		offer.ResponseSource = req.Source
		offer.DeclineReason = req.DeclineReason
	})
}

// nextOfferAfterDecline is the offer a sequential tender sends once the one it
// is waiting on is declined: the first offer in rank order not yet sent. A
// broadcast tender sends nothing further.
func nextOfferAfterDecline(entity *tender.Tender) *tender.TenderOffer {
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

func hasOutstandingOffer(entity *tender.Tender) bool {
	for _, offer := range entity.Offers {
		if offer != nil && offer.Status == tender.OfferStatusSent {
			return true
		}
	}

	return false
}

func settledOfferStatus(status tender.OfferStatus) tender.OfferStatus {
	for _, settlement := range acceptSettlements {
		if settlement.from == status {
			return settlement.to
		}
	}

	return status
}

// copyTender copies the tender and each of its offers, applying change to
// every offer copy, so a projection never moves the rows it was read from.
func copyTender(entity *tender.Tender, change func(*tender.TenderOffer)) *tender.Tender {
	out := *entity
	out.Offers = make([]*tender.TenderOffer, 0, len(entity.Offers))
	for _, offer := range entity.Offers {
		if offer == nil {
			continue
		}
		copied := *offer
		change(&copied)
		out.Offers = append(out.Offers, &copied)
	}

	return &out
}
