package tenderservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/shipmenteventservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/tenderjobs"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

type CancelTenderRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	TenderID   pulid.ID              `json:"tenderId"`
	Reason     string                `json:"reason"`
}

func (r *CancelTenderRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if r.TenderID.IsNil() {
		multiErr.Add("tenderId", errortypes.ErrRequired, "Tender is required")
	}
	if r.Reason == "" {
		multiErr.Add("reason", errortypes.ErrRequired, "Cancellation reason is required")
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

// Cancel withdraws a live tender. An Active tender is canceled through its
// workflow so an in-flight response cannot race the withdrawal; a NeedsReview
// tender no longer has a workflow and is canceled directly. The direct path
// also serves as the fallback when the workflow is lost.
func (s *Service) Cancel(ctx context.Context, req *CancelTenderRequest) error {
	if multiErr := req.Validate(); multiErr != nil {
		return multiErr
	}

	entity, err := s.repo.GetByID(ctx, repositories.GetTenderByIDRequest{
		TenantInfo:    req.TenantInfo,
		TenderID:      req.TenderID,
		IncludeOffers: true,
	})
	if err != nil {
		return err
	}

	switch entity.Status {
	case tender.StatusActive:
		signalErr := s.workflows.SignalWorkflow(
			ctx, tenderjobs.WorkflowID(entity.ID), "", tenderjobs.TenderSignalName,
			tenderjobs.Signal{
				Kind:        tenderjobs.SignalKindCancel,
				Reason:      req.Reason,
				ActorUserID: req.TenantInfo.UserID,
			},
		)
		if signalErr == nil {
			return nil
		}
		s.l.Warn("failed to signal tender workflow for cancel; canceling directly",
			zap.Error(signalErr), zap.String("tenderId", entity.ID.String()))
		return s.CancelDirect(ctx, req.TenantInfo, entity, req.Reason)
	case tender.StatusNeedsReview:
		return s.CancelDirect(ctx, req.TenantInfo, entity, req.Reason)
	default:
		return errortypes.NewBusinessError("Tender has already finished").
			WithParam("tenderId", entity.ID.String()).
			WithParam("status", entity.Status.String())
	}
}

// CancelLiveTenderForMove implements the TenderGuard hook: when a move gets
// covered or canceled outside the tender flow, any live tender on it is
// withdrawn so carriers stop receiving (and answering) dead offers.
func (s *Service) CancelLiveTenderForMove(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
	reason string,
) error {
	live, err := s.repo.GetLiveByMoveID(ctx, repositories.GetLiveTenderByMoveRequest{
		TenantInfo: tenantInfo,
		MoveID:     moveID,
	})
	if err != nil {
		return err
	}
	if live == nil {
		return nil
	}

	return s.Cancel(ctx, &CancelTenderRequest{
		TenantInfo: tenantInfo,
		TenderID:   live.ID,
		Reason:     reason,
	})
}

// CancelLiveTendersForShipment withdraws every live tender under a shipment,
// used when the shipment itself is canceled.
func (s *Service) CancelLiveTendersForShipment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
	reason string,
) error {
	tenders, err := s.repo.ListByShipment(ctx, repositories.ListTendersByShipmentRequest{
		TenantInfo: tenantInfo,
		ShipmentID: shipmentID,
	})
	if err != nil {
		return err
	}

	for _, entity := range tenders {
		if !entity.IsLive() {
			continue
		}
		if cancelErr := s.Cancel(ctx, &CancelTenderRequest{
			TenantInfo: tenantInfo,
			TenderID:   entity.ID,
			Reason:     reason,
		}); cancelErr != nil {
			s.l.Error("failed to cancel live tender for canceled shipment",
				zap.Error(cancelErr), zap.String("tenderId", entity.ID.String()))
			err = cancelErr
		}
	}

	return err
}

// CancelDirect performs the withdrawal without the workflow: CAS the tender
// terminal, withdraw outstanding offers, revoke their tokens, and record the
// event. Used for NeedsReview tenders and as the lost-workflow fallback.
func (s *Service) CancelDirect(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entity *tender.Tender,
	reason string,
) error {
	now := timeutils.NowUnix()

	moved, err := s.repo.UpdateTenderStatus(ctx, &repositories.UpdateTenderStatusRequest{
		TenantInfo: tenantInfo,
		TenderID:   entity.ID,
		FromStatus: []tender.Status{
			tender.StatusActive,
			tender.StatusNeedsReview,
		},
		ToStatus:           tender.StatusCanceled,
		CanceledByID:       userIDPtr(tenantInfo),
		CancellationReason: reason,
		CanceledAt:         &now,
	})
	if err != nil {
		return err
	}
	if !moved {
		return errortypes.NewBusinessError("Tender has already finished").
			WithParam("tenderId", entity.ID.String())
	}

	if _, err = s.repo.BulkUpdateOfferStatus(ctx, &repositories.BulkOfferStatusRequest{
		TenantInfo: tenantInfo,
		TenderID:   entity.ID,
		FromStatus: []tender.OfferStatus{tender.OfferStatusSent},
		ToStatus:   tender.OfferStatusWithdrawn,
	}); err != nil {
		return err
	}
	if _, err = s.repo.BulkUpdateOfferStatus(ctx, &repositories.BulkOfferStatusRequest{
		TenantInfo: tenantInfo,
		TenderID:   entity.ID,
		FromStatus: []tender.OfferStatus{tender.OfferStatusPending},
		ToStatus:   tender.OfferStatusSkipped,
	}); err != nil {
		return err
	}

	for _, offer := range entity.Offers {
		if offer == nil || offer.Channel != tender.ChannelEmail {
			continue
		}
		if err = s.repo.RevokeTokensForOffer(ctx, tenantInfo, offer.ID, now); err != nil {
			s.l.Warn("failed to revoke tender offer tokens on cancel",
				zap.Error(err), zap.String("offerId", offer.ID.String()))
		}
	}

	s.recordEvent(ctx, shipmenteventservice.BuildTenderWithdrawn(
		shipmenteventservice.TenantRefFor(tenantInfo),
		shipmenteventservice.TenderRef{
			ShipmentID: entity.ShipmentID,
			MoveID:     entity.ShipmentMoveID,
			TenderID:   entity.ID,
		},
		reason,
		shipmenteventservice.ActorFor(tenantInfo),
	))
	s.publishInvalidation(ctx, tenantInfo, entity.ShipmentID, "tender_canceled")

	return nil
}
