package tenderservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	portservices "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

func (s *Service) skipIneligibleOffer(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offer *tender.TenderOffer,
) (*portservices.TenderDispatchResult, error) {
	if s.intelGate == nil {
		return nil, nil
	}

	carriers, err := s.carrierRepo.GetByIDs(ctx, repositories.GetCarriersByIDsRequest{
		TenantInfo: tenantInfo,
		CarrierIDs: []pulid.ID{offer.CarrierID},
		CarrierFilterOptions: repositories.CarrierFilterOptions{
			IncludeInsurancePolicies: true,
		},
	})
	if err != nil {
		return nil, err
	}
	if len(carriers) == 0 {
		return nil, nil
	}

	gates := s.intelGates(ctx, tenantInfo, []pulid.ID{offer.CarrierID}, true)
	eligibility := carrier.EvaluateEligibility(carrier.EligibilityInput{
		Carrier: carriers[0],
		Now:     timeutils.NowUnix(),
		Intel:   gates[offer.CarrierID],
	})
	if !eligibility.IsBlocked() {
		return nil, nil
	}

	reason := "Carrier became ineligible before the offer was sent: " +
		strings.Join(eligibility.Blockers, "; ")
	moved, err := s.repo.UpdateOfferStatus(ctx, &repositories.UpdateOfferStatusRequest{
		TenantInfo:    tenantInfo,
		OfferID:       offer.ID,
		FromStatus:    []tender.OfferStatus{tender.OfferStatusPending},
		ToStatus:      tender.OfferStatusSkipped,
		DeclineReason: reason,
	})
	if err != nil {
		return nil, err
	}
	if moved {
		s.l.Info("skipped tender offer for ineligible carrier",
			zap.String("offerId", offer.ID.String()),
			zap.String("carrierId", offer.CarrierID.String()))
		s.publishTenderShipmentInvalidation(ctx, tenantInfo, offer, "tender_offer_skipped")
	}
	return &portservices.TenderDispatchResult{Delivered: false, Reason: reason}, nil
}

func (s *Service) markCarriersUsed(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offers []*tender.TenderOffer,
) {
	if s.lifecycle == nil || len(offers) == 0 {
		return
	}
	ids := make([]pulid.ID, 0, len(offers))
	for _, offer := range offers {
		ids = append(ids, offer.CarrierID)
	}
	s.lifecycle.CarriersUsed(ctx, tenantInfo, ids)
}
