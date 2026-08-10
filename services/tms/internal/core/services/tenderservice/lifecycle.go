package tenderservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	portservices "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/shipmenteventservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

// LoadPlan returns the tender's ordered offer plan with live statuses, so a
// (re)started workflow resumes exactly where the rows say it left off.
func (s *Service) LoadPlan(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	tenderID pulid.ID,
) (*portservices.TenderPlan, error) {
	entity, err := s.repo.GetByID(ctx, repositories.GetTenderByIDRequest{
		TenantInfo:    tenantInfo,
		TenderID:      tenderID,
		IncludeOffers: true,
	})
	if err != nil {
		return nil, err
	}

	plan := &portservices.TenderPlan{
		TenderID: entity.ID,
		Status:   entity.Status,
		Mode:     entity.Mode,
		Offers:   make([]portservices.TenderPlanOffer, 0, len(entity.Offers)),
	}
	for _, offer := range entity.Offers {
		if offer == nil {
			continue
		}
		plan.Offers = append(plan.Offers, portservices.TenderPlanOffer{
			OfferID:         offer.ID,
			Rank:            offer.Rank,
			Status:          offer.Status,
			OfferTTLSeconds: offer.OfferTTLSeconds,
			ExpiresAt:       offer.ExpiresAt,
		})
	}

	return plan, nil
}

// ExpireOffer lapses a Sent offer whose TTL ran out. Losing the CAS means a
// response won the race, which is not an error.
func (s *Service) ExpireOffer(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offerID pulid.ID,
) error {
	offer, err := s.repo.GetOfferByID(ctx, repositories.GetTenderOfferByIDRequest{
		TenantInfo:    tenantInfo,
		OfferID:       offerID,
		IncludeTender: true,
	})
	if err != nil {
		return err
	}

	now := timeutils.NowUnix()
	moved, err := s.repo.UpdateOfferStatus(ctx, &repositories.UpdateOfferStatusRequest{
		TenantInfo: tenantInfo,
		OfferID:    offerID,
		FromStatus: []tender.OfferStatus{tender.OfferStatusSent},
		ToStatus:   tender.OfferStatusExpired,
	})
	if err != nil {
		return err
	}
	if !moved {
		return nil
	}

	if err = s.repo.RevokeTokensForOffer(ctx, tenantInfo, offerID, now); err != nil {
		s.l.Warn("failed to revoke tokens for expired tender offer", zap.Error(err))
	}

	s.recordEvent(ctx, shipmenteventservice.BuildTenderExpired(
		shipmenteventservice.TenantRefFor(tenantInfo),
		tenderRefForOffer(offer),
		carrierNameOf(offer),
		shipmenteventservice.ActorFor(tenantInfo),
	))
	s.publishTenderShipmentInvalidation(ctx, tenantInfo, offer, "tender_offer_expired")

	return nil
}

// DeclineOffer records a carrier's decline. Losing the CAS means the offer
// already lapsed or was answered elsewhere; the response is preserved as a
// late response instead.
func (s *Service) DeclineOffer(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offerID pulid.ID,
	source tender.ResponseSource,
	reason string,
) error {
	offer, err := s.repo.GetOfferByID(ctx, repositories.GetTenderOfferByIDRequest{
		TenantInfo:    tenantInfo,
		OfferID:       offerID,
		IncludeTender: true,
	})
	if err != nil {
		return err
	}

	now := timeutils.NowUnix()
	moved, err := s.repo.UpdateOfferStatus(ctx, &repositories.UpdateOfferStatusRequest{
		TenantInfo:     tenantInfo,
		OfferID:        offerID,
		FromStatus:     []tender.OfferStatus{tender.OfferStatusSent},
		ToStatus:       tender.OfferStatusDeclined,
		RespondedAt:    &now,
		ResponseSource: source,
		DeclineReason:  reason,
	})
	if err != nil {
		return err
	}
	if !moved {
		if lateErr := s.repo.RecordLateOfferResponse(
			ctx,
			&repositories.RecordLateOfferResponseRequest{
				TenantInfo: tenantInfo,
				OfferID:    offerID,
				Action:     tender.ResponseActionDecline,
				Source:     source,
				OccurredAt: now,
			},
		); lateErr != nil {
			s.l.Warn("failed to record late decline", zap.Error(lateErr))
		}
		return nil
	}

	if err = s.repo.RevokeTokensForOffer(ctx, tenantInfo, offerID, now); err != nil {
		s.l.Warn("failed to revoke tokens for declined tender offer", zap.Error(err))
	}

	s.recordEvent(ctx, shipmenteventservice.BuildTenderDeclined(
		shipmenteventservice.TenantRefFor(tenantInfo),
		tenderRefForOffer(offer),
		carrierNameOf(offer),
		reason,
		source,
		shipmenteventservice.ActorFor(tenantInfo),
	))
	s.publishTenderShipmentInvalidation(ctx, tenantInfo, offer, "tender_offer_declined")

	return nil
}

// AcceptOffer lands a carrier's acceptance. The tender is moved terminal
// BEFORE the assignment is created so the out-of-tender coverage guard cannot
// withdraw the very tender being accepted. An assignment failure leaves the
// acceptance on record and parks the tender in NeedsReview.
func (s *Service) AcceptOffer(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offerID pulid.ID,
	source tender.ResponseSource,
) (*portservices.TenderAcceptResult, error) {
	offer, err := s.repo.GetOfferByID(ctx, repositories.GetTenderOfferByIDRequest{
		TenantInfo:    tenantInfo,
		OfferID:       offerID,
		IncludeTender: true,
	})
	if err != nil {
		return nil, err
	}
	if offer.Tender == nil {
		return nil, errortypes.NewBusinessError("Offer is not attached to a tender")
	}

	now := timeutils.NowUnix()

	tenderMoved, err := s.repo.UpdateTenderStatus(ctx, &repositories.UpdateTenderStatusRequest{
		TenantInfo: tenantInfo,
		TenderID:   offer.TenderID,
		FromStatus: []tender.Status{tender.StatusActive},
		ToStatus:   tender.StatusAccepted,
	})
	if err != nil {
		return nil, err
	}
	if !tenderMoved {
		s.recordLateAccept(ctx, tenantInfo, offer, source, now)
		return &portservices.TenderAcceptResult{Ignored: true}, nil
	}

	offerMoved, err := s.repo.UpdateOfferStatus(ctx, &repositories.UpdateOfferStatusRequest{
		TenantInfo:     tenantInfo,
		OfferID:        offerID,
		FromStatus:     []tender.OfferStatus{tender.OfferStatusSent},
		ToStatus:       tender.OfferStatusAccepted,
		RespondedAt:    &now,
		ResponseSource: source,
	})
	if err != nil {
		return nil, err
	}
	if !offerMoved {
		if _, revertErr := s.repo.UpdateTenderStatus(
			ctx,
			&repositories.UpdateTenderStatusRequest{
				TenantInfo: tenantInfo,
				TenderID:   offer.TenderID,
				FromStatus: []tender.Status{tender.StatusAccepted},
				ToStatus:   tender.StatusActive,
			},
		); revertErr != nil {
			s.l.Error("failed to revert tender after offer CAS loss", zap.Error(revertErr))
		}
		s.recordLateAccept(ctx, tenantInfo, offer, source, now)
		return &portservices.TenderAcceptResult{Ignored: true}, nil
	}

	offerIDCopy := offer.ID
	if _, err = s.repo.UpdateTenderStatus(ctx, &repositories.UpdateTenderStatusRequest{
		TenantInfo:      tenantInfo,
		TenderID:        offer.TenderID,
		FromStatus:      []tender.Status{tender.StatusAccepted},
		ToStatus:        tender.StatusAccepted,
		AcceptedOfferID: &offerIDCopy,
		AcceptedAt:      &now,
	}); err != nil {
		s.l.Warn("failed to stamp accepted offer on tender", zap.Error(err))
	}

	if err = s.repo.RevokeTokensForOffer(ctx, tenantInfo, offer.ID, now); err != nil {
		s.l.Warn("failed to revoke tokens for accepted tender offer", zap.Error(err))
	}

	if s.assigner == nil {
		return &portservices.TenderAcceptResult{
			NeedsReview:  true,
			ReviewReason: "Carrier assignment is unavailable; assign the carrier manually",
		}, nil
	}

	assignment, assignErr := s.assigner.AssignToMove(ctx, &repositories.AssignMoveToCarrierRequest{
		TenantInfo:     tenantInfo,
		ShipmentMoveID: offer.Tender.ShipmentMoveID,
		CarrierID:      offer.CarrierID,
		RateMethod:     offer.RateMethod,
		BaseRate:       offer.Rate,
	})
	if assignErr != nil {
		reason := assignErr.Error()
		s.l.Warn("tender acceptance could not be auto-assigned",
			zap.Error(assignErr), zap.String("offerId", offer.ID.String()))
		return &portservices.TenderAcceptResult{
			NeedsReview:  true,
			ReviewReason: reason,
		}, nil
	}

	if confirmErr := s.assigner.ConfirmAssignment(
		ctx, tenantInfo, assignment.ID,
	); confirmErr != nil {
		s.l.Error("failed to confirm tender-created assignment", zap.Error(confirmErr))
	}

	s.recordEvent(ctx, shipmenteventservice.BuildTenderAccepted(
		shipmenteventservice.TenantRefFor(tenantInfo),
		tenderRefForOffer(offer),
		carrierNameOf(offer),
		source,
		shipmenteventservice.ActorFor(tenantInfo),
	))
	s.notifyDispatch(ctx, tenantInfo, &dispatchNotification{
		EventType: "tender_accepted",
		Priority:  notificationPriorityMed,
		Title:     "Tender accepted",
		Message: fmt.Sprintf(
			"%s accepted the tender and has been assigned to the move",
			carrierNameOf(offer),
		),
		CorrelationID: offer.TenderID.String(),
	})
	s.publishTenderShipmentInvalidation(ctx, tenantInfo, offer, "tender_accepted")

	return &portservices.TenderAcceptResult{}, nil
}

func (s *Service) recordLateAccept(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offer *tender.TenderOffer,
	source tender.ResponseSource,
	now int64,
) {
	if err := s.repo.RecordLateOfferResponse(ctx, &repositories.RecordLateOfferResponseRequest{
		TenantInfo: tenantInfo,
		OfferID:    offer.ID,
		Action:     tender.ResponseActionAccept,
		Source:     source,
		OccurredAt: now,
	}); err != nil {
		s.l.Warn("failed to record late accept", zap.Error(err))
	}
}

// FinalizeAccepted closes out the rest of the plan after an acceptance:
// undispatched ranks are skipped, outstanding broadcast siblings are
// superseded and their tokens revoked.
func (s *Service) FinalizeAccepted(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	tenderID pulid.ID,
	acceptedOfferID pulid.ID,
) error {
	if _, err := s.repo.BulkUpdateOfferStatus(ctx, &repositories.BulkOfferStatusRequest{
		TenantInfo: tenantInfo,
		TenderID:   tenderID,
		FromStatus: []tender.OfferStatus{tender.OfferStatusPending},
		ToStatus:   tender.OfferStatusSkipped,
		ExceptID:   acceptedOfferID,
	}); err != nil {
		return err
	}
	if _, err := s.repo.BulkUpdateOfferStatus(ctx, &repositories.BulkOfferStatusRequest{
		TenantInfo: tenantInfo,
		TenderID:   tenderID,
		FromStatus: []tender.OfferStatus{tender.OfferStatusSent},
		ToStatus:   tender.OfferStatusSuperseded,
		ExceptID:   acceptedOfferID,
	}); err != nil {
		return err
	}

	entity, err := s.repo.GetByID(ctx, repositories.GetTenderByIDRequest{
		TenantInfo:    tenantInfo,
		TenderID:      tenderID,
		IncludeOffers: true,
	})
	if err != nil {
		return err
	}

	now := timeutils.NowUnix()
	for _, offer := range entity.Offers {
		if offer == nil || offer.ID == acceptedOfferID {
			continue
		}
		if offer.Status != tender.OfferStatusSuperseded {
			continue
		}
		if err = s.repo.RevokeTokensForOffer(ctx, tenantInfo, offer.ID, now); err != nil {
			s.l.Warn("failed to revoke tokens for superseded tender offer",
				zap.Error(err), zap.String("offerId", offer.ID.String()))
		}
	}

	return nil
}

// MarkNeedsReview parks the tender for a dispatcher after an acceptance that
// could not be auto-assigned. The waterfall never advances past an
// acceptance.
func (s *Service) MarkNeedsReview(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	tenderID pulid.ID,
	offerID pulid.ID,
	reason string,
) error {
	moved, err := s.repo.UpdateTenderStatus(ctx, &repositories.UpdateTenderStatusRequest{
		TenantInfo: tenantInfo,
		TenderID:   tenderID,
		FromStatus: []tender.Status{tender.StatusAccepted, tender.StatusActive},
		ToStatus:   tender.StatusNeedsReview,
	})
	if err != nil {
		return err
	}
	if !moved {
		return nil
	}

	offer, err := s.repo.GetOfferByID(ctx, repositories.GetTenderOfferByIDRequest{
		TenantInfo:    tenantInfo,
		OfferID:       offerID,
		IncludeTender: true,
	})
	if err != nil {
		return err
	}

	s.recordEvent(ctx, shipmenteventservice.BuildTenderNeedsReview(
		shipmenteventservice.TenantRefFor(tenantInfo),
		tenderRefForOffer(offer),
		carrierNameOf(offer),
		reason,
		shipmenteventservice.ActorFor(tenantInfo),
	))
	s.notifyDispatch(ctx, tenantInfo, &dispatchNotification{
		EventType: "tender_needs_review",
		Priority:  notificationPriorityHigh,
		Title:     "Tender acceptance needs review",
		Message: fmt.Sprintf(
			"%s accepted the tender but could not be assigned automatically: %s",
			carrierNameOf(offer), reason,
		),
		CorrelationID: tenderID.String(),
	})
	s.publishTenderShipmentInvalidation(ctx, tenantInfo, offer, "tender_needs_review")

	return nil
}

// MarkExhausted closes a tender whose plan ran dry with nobody accepting.
func (s *Service) MarkExhausted(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	tenderID pulid.ID,
) error {
	now := timeutils.NowUnix()
	moved, err := s.repo.UpdateTenderStatus(ctx, &repositories.UpdateTenderStatusRequest{
		TenantInfo:  tenantInfo,
		TenderID:    tenderID,
		FromStatus:  []tender.Status{tender.StatusActive},
		ToStatus:    tender.StatusExhausted,
		ExhaustedAt: &now,
	})
	if err != nil {
		return err
	}
	if !moved {
		return nil
	}

	if _, err = s.repo.BulkUpdateOfferStatus(ctx, &repositories.BulkOfferStatusRequest{
		TenantInfo: tenantInfo,
		TenderID:   tenderID,
		FromStatus: []tender.OfferStatus{tender.OfferStatusPending},
		ToStatus:   tender.OfferStatusSkipped,
	}); err != nil {
		return err
	}

	entity, err := s.repo.GetByID(ctx, repositories.GetTenderByIDRequest{
		TenantInfo: tenantInfo,
		TenderID:   tenderID,
	})
	if err != nil {
		return err
	}

	s.recordEvent(ctx, shipmenteventservice.BuildRoutingGuideExhausted(
		shipmenteventservice.TenantRefFor(tenantInfo),
		shipmenteventservice.TenderRef{
			ShipmentID: entity.ShipmentID,
			MoveID:     entity.ShipmentMoveID,
			TenderID:   entity.ID,
		},
		entity.Mode,
		shipmenteventservice.ActorFor(tenantInfo),
	))
	s.notifyDispatch(ctx, tenantInfo, &dispatchNotification{
		EventType: "tender_waterfall_exhausted",
		Priority:  notificationPriorityHigh,
		Title:     "Tender exhausted with no acceptance",
		Message: fmt.Sprintf(
			"Every carrier on the tender for move %s declined, expired, or failed. The move still needs coverage",
			entity.ShipmentMoveID.String(),
		),
		CorrelationID: entity.ID.String(),
	})
	s.publishInvalidation(ctx, tenantInfo, entity.ShipmentID, "tender_exhausted")

	return nil
}

// CancelFromWorkflow is the workflow-side arm of Cancel: it runs the direct
// withdrawal with the canceling user attributed.
func (s *Service) CancelFromWorkflow(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	tenderID pulid.ID,
	reason string,
	actorUserID pulid.ID,
) error {
	entity, err := s.repo.GetByID(ctx, repositories.GetTenderByIDRequest{
		TenantInfo:    tenantInfo,
		TenderID:      tenderID,
		IncludeOffers: true,
	})
	if err != nil {
		return err
	}
	if entity.Status.IsTerminal() && entity.Status != tender.StatusNeedsReview {
		return nil
	}

	actorTenant := tenantInfo
	actorTenant.UserID = actorUserID
	return s.CancelDirect(ctx, actorTenant, entity, reason)
}
