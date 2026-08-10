package tenderservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	portservices "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/shipmenteventservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/tenderjobs"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

// RecordResponse is the single funnel every response channel resolves
// through. It validates the response is still answerable, then signals the
// tender's workflow — which owns every state transition — so concurrent
// responses from different channels serialize in one place.
func (s *Service) RecordResponse(
	ctx context.Context,
	req *portservices.TenderResponseRequest,
) error {
	if req.OfferID.IsNil() {
		return errortypes.NewValidationError(
			"offerId", errortypes.ErrRequired, "Offer is required",
		)
	}
	if !req.Action.IsValid() {
		return errortypes.NewValidationError(
			"action", errortypes.ErrInvalid, "Response action is invalid",
		)
	}
	if !req.Source.IsValid() {
		return errortypes.NewValidationError(
			"source", errortypes.ErrInvalid, "Response source is invalid",
		)
	}

	offer, err := s.repo.GetOfferByID(ctx, repositories.GetTenderOfferByIDRequest{
		TenantInfo:    req.TenantInfo,
		OfferID:       req.OfferID,
		IncludeTender: true,
	})
	if err != nil {
		return err
	}

	if offer.Tender == nil || offer.Tender.Status != tender.StatusActive ||
		offer.Status != tender.OfferStatusSent {
		s.recordLateResponse(ctx, req, offer)
		return ErrOfferNoLongerAvailable
	}

	err = s.workflows.SignalWorkflow(
		ctx,
		offer.Tender.WorkflowID,
		"",
		tenderjobs.TenderSignalName,
		tenderjobs.Signal{
			Kind:          tenderjobs.SignalKindResponse,
			OfferID:       offer.ID,
			Action:        req.Action,
			Source:        req.Source,
			DeclineReason: req.DeclineReason,
			ActorUserID:   req.TenantInfo.UserID,
		},
	)
	if err != nil {
		s.l.Error("failed to signal tender workflow with response",
			zap.Error(err),
			zap.String("offerId", offer.ID.String()),
			zap.String("workflowId", offer.Tender.WorkflowID))
		return errortypes.NewBusinessError(
			"The response could not be recorded right now. Retry in a moment",
		)
	}

	return nil
}

// recordLateResponse appends a response that lost the race against expiry,
// supersession, or another channel — the offer's status never moves, but the
// attempt is preserved for the tender history.
func (s *Service) recordLateResponse(
	ctx context.Context,
	req *portservices.TenderResponseRequest,
	offer *tender.TenderOffer,
) {
	if offer == nil {
		return
	}

	if err := s.repo.RecordLateOfferResponse(ctx, &repositories.RecordLateOfferResponseRequest{
		TenantInfo: req.TenantInfo,
		OfferID:    offer.ID,
		Action:     req.Action,
		Source:     req.Source,
		OccurredAt: timeutils.NowUnix(),
	}); err != nil {
		s.l.Warn("failed to record late tender response", zap.Error(err))
	}

	carrierName := ""
	if offer.Carrier != nil {
		carrierName = offer.Carrier.Name
	}
	ref := shipmenteventservice.TenderRef{
		TenderID: offer.TenderID,
		OfferID:  offer.ID,
	}
	if offer.Tender != nil {
		ref.ShipmentID = offer.Tender.ShipmentID
		ref.MoveID = offer.Tender.ShipmentMoveID
	}

	s.recordEvent(ctx, shipmenteventservice.BuildTenderLateResponse(
		shipmenteventservice.TenantRefFor(req.TenantInfo),
		ref,
		carrierName,
		req.Action,
		req.Source,
		shipmenteventservice.ActorFor(req.TenantInfo),
	))
}
