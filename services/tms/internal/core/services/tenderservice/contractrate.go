package tenderservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// offerPrice is what one guide entry should offer a carrier.
type offerPrice struct {
	method shipment.CarrierRateMethod
	rate   decimal.Decimal
	// reason is why this entry cannot be offered at all. Empty means it can.
	reason string
}

// priceGuideEntry decides what to offer, reading the carrier's contract when
// the entry asks for it.
//
// An entry that did not ask keeps the rate the guide carries, so turning this
// feature on changes nothing about the guides an organization already wrote.
//
// A contract-priced entry that cannot be priced is refused rather than offered
// at the frozen rate. Somebody marked that entry to follow the contract, and
// quietly tendering last year's number instead is the opposite of what they
// asked for. The one exception is a deployment with no engine wired at all,
// where the frozen rate is the only rate there is.
func (s *Service) priceGuideEntry(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entry *tender.RoutingGuideEntry,
	entity *shipment.Shipment,
) offerPrice {
	frozen := offerPrice{method: entry.RateMethod, rate: entry.Rate}

	if !entry.UseContractRate || s.rateEngine == nil || entity == nil {
		return frozen
	}

	rated, err := s.rateEngine.RateShipment(ctx, &services.RateShipmentRequest{
		Shipment:   entity,
		TenantInfo: tenantInfo,
		PartyType:  rateagreement.PartyTypeCarrier,
		PartyID:    entry.CarrierID,
		Purpose:    ratequote.PurposeShopping,
		UserID:     tenantInfo.UserID,
	})
	if err != nil {
		s.l.Warn("failed to price a routing guide entry from its carrier's contract",
			zap.String("carrierId", entry.CarrierID.String()),
			zap.Error(err),
		)

		return offerPrice{reason: "The carrier's contract could not be read to price this offer"}
	}

	if !rated.Outcome.Priced() {
		return offerPrice{reason: "No carrier agreement covers this lane"}
	}

	// The contract produced one number, already through its own minimums and
	// rounding. Offering it as a per-mile rate would invite the offer to be
	// recomputed into something the contract never said.
	return offerPrice{method: shipment.CarrierRateMethodFlat, rate: rated.Amount}
}

// shipmentForContractPricing loads the shipment a contract-priced entry needs.
//
// It returns nothing when no entry asks for contract pricing, so a guide that
// has always used its own rates never pays for a read it does not use. A read
// that fails is logged and treated as absent, which sends every contract-priced
// entry down the "cannot be priced" path rather than tendering the wrong number.
func (s *Service) shipmentForContractPricing(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	guide *tender.RoutingGuide,
	move *shipment.ShipmentMove,
) *shipment.Shipment {
	if s.rateEngine == nil || move == nil || move.ShipmentID.IsNil() {
		return nil
	}

	wanted := false
	for _, entry := range guide.Entries {
		if entry != nil && entry.UseContractRate {
			wanted = true
			break
		}
	}

	if !wanted {
		return nil
	}

	entity, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         move.ShipmentID,
		TenantInfo: tenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		s.l.Warn("failed to load the shipment for contract priced tender offers",
			zap.String("shipmentId", move.ShipmentID.String()),
			zap.Error(err),
		)

		return nil
	}

	return entity
}
