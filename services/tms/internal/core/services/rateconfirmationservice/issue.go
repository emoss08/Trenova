package rateconfirmationservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/rateconfirmation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

var _ services.RateConfirmationIssuer = (*Service)(nil)

// IssueForTenderAcceptance documents an accepted tender offer as a Confirmed
// rate confirmation. The carrier's affirmative acceptance of the offered rate
// IS the agreement, so no signature round-trip is pending: the revision is
// born with tender provenance and lands Confirmed immediately.
//
// The method is an idempotent state machine over the assignment's active
// revision, because the accepting workflow retries it:
//   - already Confirmed from this offer: no-op;
//   - tender-generated but not yet Confirmed (a crash between generate and
//     confirm): resume by confirming and sending;
//   - a dispatcher-managed revision stands: left untouched, reported as not
//     issued so dispatch hears about it;
//   - nothing stands: full generate, confirm, send.
func (s *Service) IssueForTenderAcceptance(
	ctx context.Context,
	req *services.IssueTenderRateConfirmationRequest,
) (*services.IssueTenderRateConfirmationResult, error) {
	if req == nil || req.ShipmentMoveID.IsNil() || req.OfferID.IsNil() {
		return nil, errortypes.NewValidationError(
			"offerId", errortypes.ErrRequired,
			"Issuing a rate confirmation requires the accepted offer and its move",
		)
	}

	assignment, err := s.carrierAssignmentRepo.GetActiveByMoveID(
		ctx, req.TenantInfo, req.ShipmentMoveID,
	)
	if err != nil {
		return nil, err
	}
	if assignment == nil {
		return &services.IssueTenderRateConfirmationResult{
			Reason: "The move has no active carrier assignment to confirm",
		}, nil
	}
	if assignment.CarrierID != req.CarrierID {
		return &services.IssueTenderRateConfirmationResult{
			Reason: "The move is covered by a different carrier than the accepted offer",
		}, nil
	}

	active, err := s.repo.GetActiveByAssignmentID(ctx, req.TenantInfo, assignment.ID)
	if err != nil {
		return nil, err
	}

	switch {
	case active == nil:
		created, genErr := s.generate(ctx, req.TenantInfo, req.ShipmentMoveID, nil,
			&generateOptions{
				Via:                 rateconfirmation.ViaTenderAcceptance,
				SourceTenderOfferID: &req.OfferID,
			})
		if genErr != nil {
			return nil, genErr
		}
		return s.completeTenderIssued(ctx, req, created)
	case issuedFromOffer(active, req):
		if active.Status == rateconfirmation.StatusConfirmed {
			return &services.IssueTenderRateConfirmationResult{
				RateConfirmationID: active.ID,
				Issued:             true,
			}, nil
		}
		return s.completeTenderIssued(ctx, req, active)
	default:
		return &services.IssueTenderRateConfirmationResult{
			RateConfirmationID: active.ID,
			Reason: "A dispatcher-managed rate confirmation already covers this move; " +
				"no automatic revision was issued",
		}, nil
	}
}

func issuedFromOffer(
	entity *rateconfirmation.RateConfirmation,
	req *services.IssueTenderRateConfirmationRequest,
) bool {
	return entity.GeneratedVia == rateconfirmation.ViaTenderAcceptance &&
		entity.SourceTenderOfferID != nil &&
		*entity.SourceTenderOfferID == req.OfferID
}

// completeTenderIssued confirms the tender-generated revision with acceptance
// provenance and emails the executed record copy best-effort — a failed email
// never unwinds the agreement, and Send can be retried by a dispatcher.
func (s *Service) completeTenderIssued(
	ctx context.Context,
	req *services.IssueTenderRateConfirmationRequest,
	entity *rateconfirmation.RateConfirmation,
) (*services.IssueTenderRateConfirmationResult, error) {
	at := req.AcceptedAt
	if at <= 0 {
		at = timeutils.NowUnix()
	}

	confirmed, err := s.markConfirmed(ctx, req.TenantInfo, entity, confirmParams{
		Name:  s.tenderConfirmingName(ctx, req, entity),
		Title: tenderConfirmingTitle(req),
		Via:   rateconfirmation.ViaTenderAcceptance,
		At:    at,
	})
	if err != nil {
		return nil, err
	}

	if _, sendErr := s.Send(ctx, req.TenantInfo, confirmed.ID); sendErr != nil {
		s.l.Warn("failed to email the auto-issued rate confirmation record copy",
			zap.Error(sendErr),
			zap.String("rateConfirmationId", confirmed.ID.String()))
	}

	return &services.IssueTenderRateConfirmationResult{
		RateConfirmationID: confirmed.ID,
		Issued:             true,
	}, nil
}

func (s *Service) tenderConfirmingName(
	ctx context.Context,
	req *services.IssueTenderRateConfirmationRequest,
	entity *rateconfirmation.RateConfirmation,
) string {
	carrierEntity, err := s.carrierRepo.GetByID(ctx, repositories.GetCarrierByIDRequest{
		ID:         entity.CarrierID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil || carrierEntity == nil || carrierEntity.Name == "" {
		return "Carrier"
	}
	return carrierEntity.Name
}

func tenderConfirmingTitle(req *services.IssueTenderRateConfirmationRequest) string {
	if req.Source == "" {
		return "Tender offer acceptance"
	}
	return fmt.Sprintf("Tender offer acceptance (%s)", req.Source)
}
