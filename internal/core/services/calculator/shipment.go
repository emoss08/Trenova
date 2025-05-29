package calculator

import (
	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/statemachine"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
)

type ShipmentCalculatorParams struct {
	fx.In

	Logger              *logger.Logger
	StateMachineManager *statemachine.Manager
}

type ShipmentCalculator struct {
	l         *zerolog.Logger
	smManager *statemachine.Manager
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

// CalculateTotals handles all calculations for a shipment
func (sc *ShipmentCalculator) CalculateTotals(shp *shipment.Shipment) {
	sc.CalculateCommodityTotals(shp)
	sc.calculateShipmentCharge(shp)
}

func (sc *ShipmentCalculator) calculateShipmentCharge(shp *shipment.Shipment) {
	totals := sc.CalculateBillingAmounts(shp)

	shp.FreightChargeAmount = decimal.NewNullDecimal(totals.BaseCharge)
	shp.OtherChargeAmount = decimal.NewNullDecimal(totals.OtherChargeAmount)
	shp.TotalChargeAmount = decimal.NewNullDecimal(totals.TotalChargeAmount)
}

// CalculateBillingAmounts calculates all billing amounts for a shipment
func (sc *ShipmentCalculator) CalculateBillingAmounts(
	shp *shipment.Shipment,
) ShipmentTotalsResponse {
	// Step 1: Calculate base charge based on rating method
	baseCharge := sc.CalculateBaseCharge(shp)

	// Step 2: Calculate additional charges total
	additionalChargesTotal := sc.calculateAdditionalCharges(shp, baseCharge)

	// Step 4: Calculate the total charge amount (Base + Other Charges)
	totalCharge := baseCharge.Add(additionalChargesTotal)

	return ShipmentTotalsResponse{
		BaseCharge:        baseCharge,
		OtherChargeAmount: additionalChargesTotal,
		TotalChargeAmount: totalCharge,
	}
}

// calculateBaseCharge determines the base charge based on the shipment's rating method
func (sc *ShipmentCalculator) CalculateBaseCharge(shp *shipment.Shipment) decimal.Decimal {
	// Get default value if FreightChargeAmount is null
	freightChargeAmount := decimal.Zero
	if shp.FreightChargeAmount.Valid {
		freightChargeAmount = shp.FreightChargeAmount.Decimal
	}

	// Convert rating unit to decimal for calculations
	ratingUnit := decimal.NewFromInt(shp.RatingUnit)

	switch shp.RatingMethod {
	case shipment.RatingMethodFlatRate:
		return freightChargeAmount

	case shipment.RatingMethodPerMile:
		if !freightChargeAmount.IsZero() {
			return ratingUnit.Mul(freightChargeAmount)
		}
		return decimal.Zero

	case shipment.RatingMethodPerStop:
		return sc.calculatePerStopRate(shp)

	case shipment.RatingMethodPerPound:
		if shp.Weight != nil && *shp.Weight > 0 {
			weight := decimal.NewFromInt(*shp.Weight)
			return ratingUnit.Mul(weight)
		}
		return decimal.Zero

	case shipment.RatingMethodPerPallet:
		if shp.Pieces != nil && *shp.Pieces > 0 {
			pieces := decimal.NewFromInt(*shp.Pieces)
			return ratingUnit.Mul(pieces)
		}
		return decimal.Zero

	case shipment.RatingMethodPerLinearFoot:
		return sc.calculatePerLinearFootRate(shp)

	case shipment.RatingMethodOther:
		if !freightChargeAmount.IsZero() {
			return ratingUnit.Mul(freightChargeAmount)
		}
		return decimal.Zero

	default:
		sc.l.Warn().
			Str("shipmentID", shp.ID.String()).
			Str("ratingMethod", string(shp.RatingMethod)).
			Msg("unsupported rating method, using zero as base charge")
		return decimal.Zero
	}
}

// calculatePerStopRate calculates the charge based on number of stops
func (sc *ShipmentCalculator) calculatePerStopRate(shp *shipment.Shipment) decimal.Decimal {
	if len(shp.Moves) == 0 {
		return decimal.Zero
	}

	totalStops := int64(0)
	for _, move := range shp.Moves {
		if move.Stops != nil {
			totalStops += int64(len(move.Stops))
		}
	}

	stopsCount := decimal.NewFromInt(totalStops)
	ratingUnit := decimal.NewFromInt(shp.RatingUnit)
	return stopsCount.Mul(ratingUnit)
}

func (sc *ShipmentCalculator) calculatePerLinearFootRate(shp *shipment.Shipment) decimal.Decimal {
	if !shp.HasCommodities() {
		return decimal.Zero
	}

	totalLinearFeet := int64(0)
	for _, commodity := range shp.Commodities {
		if commodity.Commodity != nil && commodity.Commodity.LinearFeetPerUnit != nil &&
			commodity.Pieces > 0 {
			commodityLinearFeet := decimal.NewFromFloat(*commodity.Commodity.LinearFeetPerUnit)
			linearFeet := decimal.NewFromInt(commodity.Pieces).Mul(commodityLinearFeet)
			totalLinearFeet += linearFeet.IntPart()
		}
	}

	linearFeet := decimal.NewFromInt(totalLinearFeet)
	ratingUnit := decimal.NewFromInt(shp.RatingUnit)

	return linearFeet.Div(ratingUnit)
}

func (sc *ShipmentCalculator) calculateAdditionalCharges(
	shp *shipment.Shipment,
	baseCharge decimal.Decimal,
) decimal.Decimal {
	if !shp.HasAdditionalCharge() {
		return decimal.Zero
	}

	totalAdditionalCharges := decimal.Zero

	for _, charge := range shp.AdditionalCharges {
		chargeAmount := sc.calculatSingleAdditionalCharge(charge, baseCharge)
		totalAdditionalCharges = totalAdditionalCharges.Add(chargeAmount)

		sc.l.Debug().
			Str("shipmentID", shp.ID.String()).
			Str("additionalChargeID", charge.ID.String()).
			Str("method", string(charge.Method)).
			Str("amount", charge.Amount.String()).
			Int16("unit", charge.Unit).
			Str("chargeAmount", chargeAmount.String()).
			Msg("calculated additional charge")
	}

	return totalAdditionalCharges
}

func (sc *ShipmentCalculator) calculatSingleAdditionalCharge(
	charge *shipment.AdditionalCharge,
	baseCharge decimal.Decimal,
) decimal.Decimal {
	switch charge.Method {
	case accessorialcharge.MethodFlat:
		// * Default to 1 for unit if not specified for `Flat` method
		unit := charge.Unit
		if unit == 0 {
			unit = 1
		}
		return charge.Amount.Mul(decimal.NewFromInt(int64(unit)))

	case accessorialcharge.MethodDistance:
		// * Amount per unit x number of units
		return charge.Amount.Mul(decimal.NewFromInt(int64(charge.Unit)))

	case accessorialcharge.MethodPercentage:
		// * Percentage is based on the base linehaul rate
		// * Convert percentage to decimal (divide by 100) then multiply by base
		percent := charge.Amount.Div(decimal.NewFromInt(100))
		return percent.Mul(baseCharge)
	default:
		sc.l.Warn().
			Str("chargeID", charge.ID.String()).
			Str("method", string(charge.Method)).
			Msg("unsupported additional charge method")

		return decimal.Zero
	}
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

func (sc *ShipmentCalculator) CalculateStatus(shp *shipment.Shipment) error {
	sc.l.Info().
		Str("shipmentID", shp.ID.String()).
		Msg("calculating shipment status and timestamps")

	// * use the state machine manager to calculate the status
	if err := sc.smManager.CalculateStatuses(shp); err != nil {
		sc.l.Error().
			Str("shipmentID", shp.ID.String()).
			Err(err).
			Msg("failed to calculate shipment status")

		return eris.Wrap(err, "failed to calculate shipment status")
	}

	sc.l.Info().
		Str("shipmentID", shp.ID.String()).
		Str("status", string(shp.Status)).
		Msg("calculated shipment status")

	return nil
}

func (sc *ShipmentCalculator) CalculateTimestamps(shp *shipment.Shipment) error {
	sc.l.Debug().
		Str("shipmentID", shp.ID.String()).
		Msg("calculating shipment timestamp")

	// * use the state machine manager to calculate the timestamps
	if err := sc.smManager.CalculateShipmentTimestamps(shp); err != nil {
		sc.l.Error().
			Str("shipmentID", shp.ID.String()).
			Err(err).
			Msg("failed to calculate shipment timestamps")
		return eris.Wrap(err, "failed to calculate shipment timestamps")
	}

	// Only log the timestamps if they are not nil
	logEvent := sc.l.Debug().
		Str("shipmentID", shp.ID.String())

	if shp.ActualShipDate != nil {
		logEvent = logEvent.Int64("actualShipDate", *shp.ActualShipDate)
	}

	if shp.ActualDeliveryDate != nil {
		logEvent = logEvent.Int64("actualDeliveryDate", *shp.ActualDeliveryDate)
	}

	logEvent.Msg("calculated shipment timestamps")

	return nil
}
