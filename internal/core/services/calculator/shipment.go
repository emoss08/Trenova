package calculator

import (
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ShipmentCalculatorParams struct {
	fx.In

	Logger *logger.Logger
}

type ShipmentCalculator struct {
	l *zerolog.Logger
}

func NewShipmentCalculator(p ShipmentCalculatorParams) *ShipmentCalculator {
	log := p.Logger.With().
		Str("service", "ShipmentCalculator").
		Logger()

	return &ShipmentCalculator{
		l: &log,
	}
}

func (sc *ShipmentCalculator) CalculateTotals(shp *shipment.Shipment) {
	sc.CalculateCommodityTotals(shp)
}

func (sc *ShipmentCalculator) CalculateCommodityTotals(shp *shipment.Shipment) {
	if len(shp.Commodities) == 0 {
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
