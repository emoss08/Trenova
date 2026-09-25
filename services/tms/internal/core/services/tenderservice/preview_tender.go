package tenderservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// TenderPreview is a tender as creating it would start it: the offers in the
// order carriers are asked, at their rates and over their channels, and what
// the eligibility screen said about each carrier. Nothing is saved and no
// workflow starts.
type TenderPreview struct {
	Tender       *tender.Tender
	Carriers     map[pulid.ID]string
	Guide        *tender.RoutingGuide
	Screening    *GuideScreeningSummary
	Warnings     []string
	RefusalError error
}

// PreviewWaterfall is CreateWaterfall without the save: the guide the lane
// matches and the offers its entries become, with the carriers it skips.
func (s *Service) PreviewWaterfall(
	ctx context.Context,
	req *CreateWaterfallTenderRequest,
) (*TenderPreview, error) {
	waterfall, err := s.planWaterfall(ctx, req, false)
	if err != nil {
		return nil, err
	}

	return &TenderPreview{
		Tender:    waterfall.tender,
		Carriers:  waterfall.offers.carriers,
		Guide:     waterfall.guide,
		Screening: waterfall.offers.screeningSummary(),
	}, nil
}

// PreviewSpot is CreateSpot without the save. Insurance warnings the call
// does not override are returned as the refusal CreateSpot would make.
func (s *Service) PreviewSpot(
	ctx context.Context,
	req *CreateSpotTenderRequest,
) (*TenderPreview, error) {
	spot, err := s.planSpot(ctx, req, false)
	if err != nil {
		return nil, err
	}

	return &TenderPreview{
		Tender:       spot.tender,
		Carriers:     spot.offers.carriers,
		Warnings:     spot.offers.warnings,
		RefusalError: spot.refusal(req),
	}, nil
}

// PreviewLiveTenders is what cancelling a shipment withdraws: every live
// tender on it, with the offers carriers hold.
func (s *Service) PreviewLiveTenders(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) ([]*tender.Tender, error) {
	live, err := s.liveTendersForShipment(ctx, tenantInfo, shipmentID)
	if err != nil {
		return nil, err
	}

	withOffers := make([]*tender.Tender, 0, len(live))
	for _, entity := range live {
		loaded, gErr := s.repo.GetByID(ctx, repositories.GetTenderByIDRequest{
			TenantInfo:    tenantInfo,
			TenderID:      entity.ID,
			IncludeOffers: true,
		})
		if gErr != nil {
			return nil, gErr
		}
		withOffers = append(withOffers, loaded)
	}

	return withOffers, nil
}

func (s *Service) liveTendersForShipment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) ([]*tender.Tender, error) {
	tenders, err := s.repo.ListByShipment(ctx, repositories.ListTendersByShipmentRequest{
		TenantInfo: tenantInfo,
		ShipmentID: shipmentID,
	})
	if err != nil {
		return nil, err
	}

	live := make([]*tender.Tender, 0, len(tenders))
	for _, entity := range tenders {
		if entity.IsLive() {
			live = append(live, entity)
		}
	}

	return live, nil
}

type waterfallPlan struct {
	tender *tender.Tender
	guide  *tender.RoutingGuide
	offers *guideOfferPlan
}

// planWaterfall is everything CreateWaterfall decides before it saves: the
// move is tenderable, the guide resolves, and its entries are screened into
// offers. refreshIntel refreshes carrier intelligence before screening,
// which writes, so only the write asks for it.
func (s *Service) planWaterfall(
	ctx context.Context,
	req *CreateWaterfallTenderRequest,
	refreshIntel bool,
) (*waterfallPlan, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	move, sh, err := s.loadTenderableMove(ctx, req.TenantInfo, req.ShipmentMoveID)
	if err != nil {
		return nil, err
	}

	guide, err := s.resolveGuide(ctx, req, move, sh)
	if err != nil {
		return nil, err
	}

	offers, err := s.buildGuideOffers(ctx, req.TenantInfo, guide, move, refreshIntel)
	if err != nil {
		return nil, err
	}

	guideID := guide.ID

	return &waterfallPlan{
		guide:  guide,
		offers: offers,
		tender: &tender.Tender{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			ShipmentID:     move.ShipmentID,
			ShipmentMoveID: move.ID,
			RoutingGuideID: &guideID,
			Mode:           tender.ModeWaterfall,
			Status:         tender.StatusActive,
			CreatedByID:    userIDPtr(req.TenantInfo),
			Offers:         offers.offers,
		},
	}, nil
}

type spotOfferPlan struct {
	offers   []*tender.TenderOffer
	warnings []string
	carriers map[pulid.ID]string
}

type spotPlan struct {
	tender *tender.Tender
	offers *spotOfferPlan
}

// planSpot is everything CreateSpot decides before it saves.
func (s *Service) planSpot(
	ctx context.Context,
	req *CreateSpotTenderRequest,
	refreshIntel bool,
) (*spotPlan, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	move, _, err := s.loadTenderableMove(ctx, req.TenantInfo, req.ShipmentMoveID)
	if err != nil {
		return nil, err
	}

	offers, err := s.buildSpotOffers(ctx, req, move, refreshIntel)
	if err != nil {
		return nil, err
	}

	return &spotPlan{
		offers: offers,
		tender: &tender.Tender{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			ShipmentID:     move.ShipmentID,
			ShipmentMoveID: move.ID,
			Mode:           req.Mode,
			Status:         tender.StatusActive,
			CreatedByID:    userIDPtr(req.TenantInfo),
			Offers:         offers.offers,
		},
	}, nil
}

// refusal is the error a spot tender is refused with when a carrier on it
// carries insurance warnings the call did not override.
func (p *spotPlan) refusal(req *CreateSpotTenderRequest) error {
	if len(p.offers.warnings) == 0 || req.OverrideInsuranceWarnings {
		return nil
	}

	return errortypes.NewBusinessError(
		"Carrier has insurance warnings: {0}. Confirm the override to proceed",
		strings.Join(p.offers.warnings, "; "),
	).WithParam("overridable", "true")
}

// offerWithdrawals is what withdrawing a tender does to its offers: an offer
// a carrier holds is withdrawn, and one not yet sent is skipped.
var offerWithdrawals = []struct {
	from tender.OfferStatus
	to   tender.OfferStatus
}{
	{from: tender.OfferStatusSent, to: tender.OfferStatusWithdrawn},
	{from: tender.OfferStatusPending, to: tender.OfferStatusSkipped},
}

// WithdrawnOfferStatus is the status withdrawing a tender leaves an offer
// in, and whether it moves at all.
func WithdrawnOfferStatus(status tender.OfferStatus) (tender.OfferStatus, bool) {
	for _, withdrawal := range offerWithdrawals {
		if withdrawal.from == status {
			return withdrawal.to, true
		}
	}

	return status, false
}

func carrierNames(carriers []*carrier.Carrier) map[pulid.ID]string {
	names := make(map[pulid.ID]string, len(carriers))
	for _, entity := range carriers {
		if entity != nil {
			names[entity.ID] = entity.Name
		}
	}

	return names
}
