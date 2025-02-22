package calculator

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/statemachine"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ShipmentCalculatorParams struct {
	fx.In

	Logger              *logger.Logger
	StateMachineManager *statemachine.StateMachineManager
}

type ShipmentCalculator struct {
	l         *zerolog.Logger
	smManager *statemachine.StateMachineManager
}

func NewShipmentCalculator(p ShipmentCalculatorParams) *ShipmentCalculator {
	log := p.Logger.With().
		Str("service", "ShipmentCalculator").
		Logger()

	return &ShipmentCalculator{
		smManager: p.StateMachineManager,
		l:         &log,
	}
}

func (sc *ShipmentCalculator) CalculateTotals(shp *shipment.Shipment) {
	sc.CalculateCommodityTotals(shp)
}

func (sc *ShipmentCalculator) CalculateCommodityTotals(shp *shipment.Shipment) {
	if !shp.HasCommodities() {
		sc.l.Debug().
			Str("shipmentID", shp.ID.String()).
			Msg("no commodities found")
		return
	}

	var totalPieces, totalWeight int64

	for _, commodity := range shp.Commodities {
		// Calculate total weight for this commodity (pieces * weight per piece)
		commodityTotalWeight := commodity.Pieces * commodity.Weight

		sc.l.Debug().
			Str("commodityID", commodity.ID.String()).
			Int64("pieces", commodity.Pieces).
			Int64("weightPerPiece", commodity.Weight).
			Int64("totalWeight", commodityTotalWeight).
			Msg("calculating commodity totals")

		totalPieces += commodity.Pieces
		totalWeight += commodityTotalWeight
	}

	sc.l.Debug().
		Str("shipmentID", shp.ID.String()).
		Int64("totalPieces", totalPieces).
		Int64("totalWeight", totalWeight).
		Msg("calculated final shipment totals")

	shp.Pieces = &totalPieces
	shp.Weight = &totalWeight
}

func (sc *ShipmentCalculator) CalculateStatus(ctx context.Context, shp *shipment.Shipment) error {
	sc.l.Debug().
		Str("shipmentID", shp.ID.String()).
		Msg("calculating shipment status")

	// use the state machine manager to calculate the status
	if err := sc.smManager.CalculateStatuses(ctx, shp); err != nil {
		sc.l.Error().
			Str("shipmentID", shp.ID.String()).
			Err(err).
			Msg("failed to calculate shipment status")

		return eris.Wrap(err, "failed to calculate shipment status")
	}

	sc.l.Debug().
		Str("shipmentID", shp.ID.String()).
		Str("status", string(shp.Status)).
		Msg("calculated shipment status")

	return nil
}
