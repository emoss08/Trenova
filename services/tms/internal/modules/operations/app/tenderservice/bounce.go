package tenderservice

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenteventservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
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

// planBounce is every check Bounce makes before it acts, so a preview refuses
// exactly what the write would.
func (s *Service) planBounce(ctx context.Context, req *BounceTenderRequest) (*bouncePlan, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}
	if req.TenderID.IsNil() {
		offer, err := s.repo.GetOfferByID(ctx, tenderports.GetTenderOfferByIDRequest{
			TenantInfo: req.TenantInfo,
			OfferID:    req.OfferID,
		})
		if err != nil {
			return nil, err
		}
		req.TenderID = offer.TenderID
	}

	entity, err := s.repo.GetByID(ctx, tenderports.GetTenderByIDRequest{
		TenantInfo:    req.TenantInfo,
		TenderID:      req.TenderID,
		IncludeOffers: true,
	})
	if err != nil {
		return nil, err
	}
	revoked, ok := entity.CanBounce()
	if !ok || (!req.OfferID.IsNil() && revoked.ID != req.OfferID) {
		return nil, errortypes.NewBusinessError(
			"Only an offer a carrier accepted can be backed out of",
		).WithParam("status", entity.Status.String())
	}
	if entity.BounceCount >= tender.MaxBounces {
		return nil, errortypes.NewBusinessError(
			"This tender has been bounced too many times; start a new tender for the move",
		)
	}

	live, err := s.repo.GetLiveByMoveID(ctx, tenderports.GetLiveTenderByMoveRequest{
		TenantInfo: req.TenantInfo,
		MoveID:     entity.ShipmentMoveID,
	})
	if err != nil && !errortypes.IsNotFoundError(err) {
		return nil, err
	}
	if live != nil && live.ID != entity.ID {
		return nil, errortypes.NewBusinessError(
			"The move already has another tender in progress",
		)
	}

	assignment, err := s.carrierAssignmentRepo.GetActiveByMoveID(
		ctx, req.TenantInfo, entity.ShipmentMoveID,
	)
	if err != nil {
		return nil, err
	}
	if assignment != nil && assignment.CarrierID != revoked.CarrierID {
		return nil, errortypes.NewBusinessError(
			"The move is covered by a different carrier than the one that accepted the tender",
		)
	}
	if assignment != nil && s.assigner == nil {
		return nil, errortypes.NewBusinessError(
			"Carrier assignment is unavailable right now. Retry the request",
		)
	}

	return &bouncePlan{
		entity:     entity,
		revoked:    revoked,
		assignment: assignment,
		reoffers:   entity.ReofferPlan(revoked),
	}, nil
}

// PreviewBounce is what Bounce would do, from the same checks, without
// writing.
func (s *Service) PreviewBounce(
	ctx context.Context,
	req *BounceTenderRequest,
) (*BouncePreview, error) {
	plan, err := s.planBounce(ctx, req)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	after := bouncedTender(plan, req, now)
	return &BouncePreview{
		Before:      plan.entity,
		After:       after,
		Revoked:     after.FindOffer(plan.revoked.ID),
		Reoffers:    plan.reoffers,
		Unassigns:   plan.assignment != nil,
		WillExhaust: len(plan.reoffers) == 0,
	}, nil
}

// Bounce walks back a carrier's acceptance and sends the tender on to the
// carriers the acceptance cut short, in rank order. The carrier comes off the
// move first, voiding its rate confirmation; a move already under way cannot
// be bounced, because the assignment can no longer be canceled. With nobody
// left to ask, the new round exhausts at once and the organization's
// escalation policy takes over.
//
// It is safe to retry: a retry that finds the carrier already off the move
// carries on from there.
func (s *Service) Bounce(ctx context.Context, req *BounceTenderRequest) (*tender.Tender, error) {
	plan, err := s.planBounce(ctx, req)
	if err != nil {
		return nil, err
	}
	if req.Source == "" {
		req.Source = tender.ResponseSourceManual
	}
	reason := strings.TrimSpace(req.Reason)

	if plan.assignment != nil {
		if err = s.assigner.Cancel(ctx, &shipmentports.CancelCarrierAssignmentRequest{
			TenantInfo:     req.TenantInfo,
			ShipmentMoveID: plan.entity.ShipmentMoveID,
			Reason:         "Carrier backed out of the accepted tender: " + reason,
		}); err != nil {
			return nil, err
		}
	}

	now := timeutils.NowUnix()
	reopened, err := s.repo.Reopen(ctx, &tenderports.ReopenTenderRequest{
		TenantInfo:     req.TenantInfo,
		TenderID:       plan.entity.ID,
		RevokedOfferID: plan.revoked.ID,
		Reason:         reason,
		RevokedByID:    userIDPtr(req.TenantInfo),
		At:             now,
		Reoffers:       plan.reoffers,
	})
	if err != nil {
		return nil, err
	}
	if !reopened {
		return nil, errortypes.NewBusinessError("The tender has already moved on")
	}

	entity, err := s.repo.GetByID(ctx, tenderports.GetTenderByIDRequest{
		TenantInfo:    req.TenantInfo,
		TenderID:      plan.entity.ID,
		IncludeOffers: true,
	})
	if err != nil {
		return nil, err
	}
	if err = s.repo.RevokeTokensForOffer(ctx, req.TenantInfo, plan.revoked.ID, now); err != nil {
		s.l.Warn("failed to revoke tokens for bounced tender offer", zap.Error(err))
	}
	if err = s.startRound(ctx, req.TenantInfo, entity); err != nil {
		return nil, err
	}

	s.announceBounce(ctx, req, plan, entity)
	return entity, nil
}

func (s *Service) announceBounce(
	ctx context.Context,
	req *BounceTenderRequest,
	plan *bouncePlan,
	entity *tender.Tender,
) {
	offer := plan.revoked
	offer.Tender = entity
	carrierName := carrierNameOf(offer)
	reason := strings.TrimSpace(req.Reason)

	s.recordEvent(ctx, shipmenteventservice.BuildTenderBounced(
		shipmenteventservice.TenantRefFor(req.TenantInfo),
		tenderRefForOffer(offer),
		shipmenteventservice.TenderBounce{
			CarrierName: carrierName,
			Reason:      reason,
			Source:      req.Source,
			Reoffers:    len(plan.reoffers),
			Round:       entity.BounceCount,
		},
		shipmenteventservice.ActorFor(req.TenantInfo),
	))

	message := fmt.Sprintf(
		"%s backed out of the tender after accepting it (%s). The tender goes on to %d more %s",
		carrierName, reason, len(plan.reoffers), pluralCarriers(len(plan.reoffers)),
	)
	if len(plan.reoffers) == 0 {
		message = fmt.Sprintf(
			"%s backed out of the tender after accepting it (%s). Nobody is left to ask, so the move needs coverage",
			carrierName,
			reason,
		)
	}
	s.notifyDispatch(ctx, req.TenantInfo, &dispatchNotification{
		EventType:     "tender_bounced",
		Priority:      notificationPriorityHigh,
		Title:         "Carrier backed out of a tender",
		Message:       message,
		CorrelationID: entity.ID.String(),
		Link:          dispatchConsoleMoveLink(entity.ShipmentMoveID),
	})
	s.logTenderAudit(&tenderAuditParams{
		TenantInfo: req.TenantInfo,
		TenderID:   entity.ID,
		Operation:  permission.OpUpdate,
		UserID:     req.TenantInfo.UserID,
		Comment: fmt.Sprintf(
			"Tender bounced by %s via %s: %s", carrierName, req.Source.String(), reason,
		),
		Previous: plan.entity,
		Current:  entity,
	})
	shipmentports.InvalidateShipments(
		ctx,
		s.invalidator,
		shipmentports.ShipmentInvalidationByUser(
			req.TenantInfo,
			entity.ShipmentID,
			"tender_bounced",
		),
	)
}

func pluralCarriers(n int) string {
	if n == 1 {
		return "carrier"
	}
	return "carriers"
}

// bouncedTender is the tender as Bounce leaves it: live again on the next
// round, the acceptance revoked and the re-offers waiting. The stored tender
// is left as it is.
func bouncedTender(plan *bouncePlan, req *BounceTenderRequest, now int64) *tender.Tender {
	reason := strings.TrimSpace(req.Reason)
	out := copyTender(plan.entity, func(offer *tender.TenderOffer) {
		if offer.ID != plan.revoked.ID {
			return
		}
		offer.Status = tender.OfferStatusRevoked
		offer.RevokedAt = &now
		offer.RevokeReason = reason
		offer.RevokedByID = userIDPtr(req.TenantInfo)
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

// bounceOnCarrierDecline treats a carrier's EDI decline of an offer it had
// already accepted as the carrier backing out: a 990 is the carrier's own word,
// so the tender is bounced on it. Every other answer to an accepted offer is
// late and is filed as such by the caller.
func (s *Service) bounceOnCarrierDecline(
	ctx context.Context,
	req *tenderports.TenderResponseRequest,
) (bool, error) {
	if req.Source != tender.ResponseSourceEDI || req.Action != tender.ResponseActionDecline ||
		req.OfferID.IsNil() {
		return false, nil
	}
	offer, err := s.repo.GetOfferByID(ctx, tenderports.GetTenderOfferByIDRequest{
		TenantInfo:    req.TenantInfo,
		OfferID:       req.OfferID,
		IncludeTender: true,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return false, nil
		}
		return false, err
	}
	if offer.Tender == nil || offer.Status != tender.OfferStatusAccepted ||
		offer.Tender.AcceptedOfferID == nil || *offer.Tender.AcceptedOfferID != offer.ID {
		return false, nil
	}

	reason := strings.TrimSpace(req.DeclineReason)
	if reason == "" {
		reason = "The carrier declined by EDI after accepting"
	}
	reason = stringutils.TruncateRunes(reason, maxBounceReasonLength)
	_, err = s.Bounce(ctx, &BounceTenderRequest{
		TenantInfo: req.TenantInfo,
		TenderID:   offer.TenderID,
		Reason:     reason,
		Source:     tender.ResponseSourceEDI,
	})
	return true, err
}
