package tenderservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenteventservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/jobs/tenderjobs/tendercontract"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

// maxDeclineReasonLength bounds carrier-supplied text before it travels into
// workflow signals, events, and notifications; the client caps at the same
// length.
const maxDeclineReasonLength = 500

// RecordResponse is the single funnel every response channel resolves
// through. It validates the response is still answerable, then signals the
// tender's workflow — which owns every state transition — so concurrent
// responses from different channels serialize in one place.
func (s *Service) RecordResponse(
	ctx context.Context,
	req *tenderports.TenderResponseRequest,
) error {
	if bounced, err := s.bounceOnCarrierDecline(ctx, req); bounced || err != nil {
		return err
	}
	if decision, err := s.counterDecisionFor(ctx, req); decision != nil || err != nil {
		if err != nil {
			return err
		}
		return s.DecideCounter(ctx, decision)
	}

	offer, err := s.planResponse(ctx, req)
	if isNoLongerAvailable(err) {
		s.recordLateResponse(ctx, req, offer)
		return err
	}
	if err != nil {
		return err
	}

	workflowID := tendercontract.WorkflowID(offer.TenderID, offer.Tender.BounceCount)
	err = s.workflows.SignalWorkflow(
		ctx,
		workflowID,
		"",
		tendercontract.TenderSignalName,
		tendercontract.Signal{
			Kind:          tendercontract.SignalKindResponse,
			OfferID:       offer.ID,
			Action:        req.Action,
			Source:        req.Source,
			DeclineReason: req.DeclineReason,
			CounterRate:   req.CounterRate,
			CounterNote:   strings.TrimSpace(req.CounterNote),
			ActorUserID:   req.TenantInfo.UserID,
		},
	)
	if err != nil {
		s.l.Error("failed to signal tender workflow with response",
			zap.Error(err),
			zap.String("offerId", offer.ID.String()),
			zap.String("workflowId", workflowID))
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
	req *tenderports.TenderResponseRequest,
	offer *tender.TenderOffer,
) {
	if offer == nil {
		return
	}
	s.recordLateOfferResponse(ctx, req.TenantInfo, offer, req.Action, req.Source)
}

// RecordLateResponse preserves a response the workflow drained after its
// offer had already been expired, superseded, or answered — the race the
// service-side gate cannot see because the signal was already in flight.
func (s *Service) RecordLateResponse(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offerID pulid.ID,
	action tender.ResponseAction,
	source tender.ResponseSource,
) error {
	offer, err := s.repo.GetOfferByID(ctx, tenderports.GetTenderOfferByIDRequest{
		TenantInfo:    tenantInfo,
		OfferID:       offerID,
		IncludeTender: true,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil
		}
		return err
	}
	if offer.Status.IsOutstanding() || offer.RespondedAt != nil {
		return nil
	}

	s.recordLateOfferResponse(ctx, tenantInfo, offer, action, source)
	return nil
}

func (s *Service) recordLateOfferResponse(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offer *tender.TenderOffer,
	action tender.ResponseAction,
	source tender.ResponseSource,
) {
	if err := s.repo.RecordLateOfferResponse(ctx, &tenderports.RecordLateOfferResponseRequest{
		TenantInfo: tenantInfo,
		OfferID:    offer.ID,
		Action:     action,
		Source:     source,
		OccurredAt: timeutils.NowUnix(),
	}); err != nil {
		s.l.Warn("failed to record late tender response", zap.Error(err))
	}

	s.recordEvent(ctx, shipmenteventservice.BuildTenderLateResponse(
		shipmenteventservice.TenantRefFor(tenantInfo),
		tenderRefForOffer(offer),
		carrierNameOf(offer),
		action,
		source,
		shipmenteventservice.ActorFor(tenantInfo),
	))
}
