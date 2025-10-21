package calculator

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/services/formulatemplate"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/statemachine"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ShipmentCalculatorParams struct {
	fx.In

	Logger              *zap.Logger
	StateMachineManager *statemachine.Manager
	FormulaService      *formulatemplate.Service
}

type ShipmentCalculator struct {
	l              *zap.Logger
	smManager      *statemachine.Manager
	formulaService *formulatemplate.Service
}

func NewShipmentCalculator(p ShipmentCalculatorParams) *ShipmentCalculator {
	return &ShipmentCalculator{
		smManager:      p.StateMachineManager,
		formulaService: p.FormulaService,
		l:              p.Logger.Named("calculator.shipment"),
	}
}

func (sc *ShipmentCalculator) CalculateTotals(
	ctx context.Context,
	shp *shipment.Shipment,
	userID pulid.ID,
) {
	sc.CalculateCommodityTotals(shp)
	sc.calculateShipmentCharge(ctx, shp, userID)
}

func (sc *ShipmentCalculator) calculateShipmentCharge(
	ctx context.Context,
	shp *shipment.Shipment,
	userID pulid.ID,
) {
	totals := sc.CalculateBillingAmounts(ctx, shp, userID)

	sc.l.Debug("calculated shipment charge",
		zap.String("shipmentID", shp.ID.String()),
		zap.String("baseCharge", totals.BaseCharge.String()),
		zap.String("otherChargeAmount", totals.OtherChargeAmount.String()),
		zap.String("totalChargeAmount", totals.TotalChargeAmount.String()),
	)

	shp.FreightChargeAmount = decimal.NewNullDecimal(totals.BaseCharge)
	shp.OtherChargeAmount = decimal.NewNullDecimal(totals.OtherChargeAmount)
	shp.TotalChargeAmount = decimal.NewNullDecimal(totals.TotalChargeAmount)
}

func (sc *ShipmentCalculator) CalculateBillingAmounts(
	ctx context.Context,
	shp *shipment.Shipment,
	userID pulid.ID,
) ShipmentTotalsResponse {
	baseCharge := sc.CalculateBaseCharge(ctx, shp, userID)
	additionalChargesTotal := sc.calculateAdditionalCharges(shp, baseCharge)
	totalCharge := baseCharge.Add(additionalChargesTotal)

	return ShipmentTotalsResponse{
		BaseCharge:        baseCharge,
		OtherChargeAmount: additionalChargesTotal,
		TotalChargeAmount: totalCharge,
	}
}

func (sc *ShipmentCalculator) CalculateBaseCharge(
	ctx context.Context,
	shp *shipment.Shipment,
	userID pulid.ID,
) decimal.Decimal {
	freightChargeAmount := decimal.Zero
	if shp.FreightChargeAmount.Valid {
		freightChargeAmount = shp.FreightChargeAmount.Decimal
	}

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

	case shipment.RatingMethodFormulaTemplate:
		return sc.calculateFormulaTemplateRate(ctx, shp, userID)

	default:
		sc.l.Warn("unsupported rating method, using zero as base charge",
			zap.String("shipmentID", shp.ID.String()),
			zap.String("ratingMethod", string(shp.RatingMethod)),
		)
		return decimal.Zero
	}
}

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

func (sc *ShipmentCalculator) calculateFormulaTemplateRate(
	ctx context.Context,
	shp *shipment.Shipment,
	userID pulid.ID,
) decimal.Decimal {
	if shp.FormulaTemplateID == nil || shp.FormulaTemplateID.IsNil() {
		sc.l.Error("formula template rating method selected but no formula template ID provided",
			zap.String("shipmentID", shp.ID.String()),
		)
		return decimal.Zero
	}

	rate, err := sc.formulaService.CalculateShipmentRate(ctx, *shp.FormulaTemplateID, shp, userID)
	if err != nil {
		sc.l.Error("failed to calculate rate using formula template",
			zap.String("shipmentID", shp.ID.String()),
			zap.String("formulaTemplateID", shp.FormulaTemplateID.String()),
			zap.Error(err),
		)
		return decimal.Zero
	}

	sc.l.Info("calculated rate using formula template",
		zap.String("shipmentID", shp.ID.String()),
		zap.String("formulaTemplateID", shp.FormulaTemplateID.String()),
		zap.String("calculatedRate", rate.String()),
	)

	return rate
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

		sc.l.Debug("calculated additional charge",
			zap.String("shipmentID", shp.ID.String()),
			zap.String("additionalChargeID", charge.ID.String()),
			zap.String("method", string(charge.Method)),
			zap.String("amount", charge.Amount.String()),
			zap.Int16("unit", charge.Unit),
			zap.String("chargeAmount", chargeAmount.String()),
		)
	}

	return totalAdditionalCharges
}

func (sc *ShipmentCalculator) calculatSingleAdditionalCharge(
	charge *shipment.AdditionalCharge,
	baseCharge decimal.Decimal,
) decimal.Decimal {
	switch charge.Method {
	case accessorialcharge.MethodFlat:
		unit := charge.Unit
		if unit == 0 {
			unit = 1
		}
		return charge.Amount.Mul(decimal.NewFromInt(int64(unit)))

	case accessorialcharge.MethodDistance:
		return charge.Amount.Mul(decimal.NewFromInt(int64(charge.Unit)))

	case accessorialcharge.MethodPercentage:
		percent := charge.Amount.Div(decimal.NewFromInt(100))
		return percent.Mul(baseCharge)
	default:
		sc.l.Warn("unsupported additional charge method",
			zap.String("chargeID", charge.ID.String()),
			zap.String("method", string(charge.Method)),
		)

		return decimal.Zero
	}
}

func (sc *ShipmentCalculator) CalculateCommodityTotals(shp *shipment.Shipment) {
	if !shp.HasCommodities() {
		sc.l.Debug("no commodities found",
			zap.String("shipmentID", shp.ID.String()),
		)
		return
	}

	var totalPieces, totalWeight int64

	for _, commodity := range shp.Commodities {
		commodityTotalWeight := commodity.Pieces * commodity.Weight

		sc.l.Debug("calculating commodity totals",
			zap.String("commodityID", commodity.ID.String()),
			zap.Int64("pieces", commodity.Pieces),
			zap.Int64("weightPerPiece", commodity.Weight),
			zap.Int64("totalWeight", commodityTotalWeight),
		)

		totalPieces += commodity.Pieces
		totalWeight += commodityTotalWeight
	}

	sc.l.Debug("calculated final shipment totals",
		zap.String("shipmentID", shp.ID.String()),
		zap.Int64("totalPieces", totalPieces),
		zap.Int64("totalWeight", totalWeight),
	)

	shp.Pieces = &totalPieces
	shp.Weight = &totalWeight
}

func (sc *ShipmentCalculator) CalculateStatus(shp *shipment.Shipment) error {
	sc.l.Info("calculating shipment status and timestamps",
		zap.String("shipmentID", shp.ID.String()),
	)

	if err := sc.smManager.CalculateStatuses(shp); err != nil {
		sc.l.Error("failed to calculate shipment status",
			zap.String("shipmentID", shp.ID.String()),
			zap.Error(err),
		)

		return fmt.Errorf("failed to calculate shipment status: %w", err)
	}

	sc.l.Info("calculated shipment status",
		zap.String("shipmentID", shp.ID.String()),
		zap.String("status", string(shp.Status)),
	)

	return nil
}

func (sc *ShipmentCalculator) CalculateTimestamps(shp *shipment.Shipment) error {
	log := sc.l.With(
		zap.String("operation", "CalculateTimestamps"),
		zap.String("shipmentID", shp.ID.String()),
	)

	log.Debug("calculating shipment timestamps")

	if err := sc.smManager.CalculateShipmentTimestamps(shp); err != nil {
		log.Error("failed to calculate shipment timestamps",
			zap.Error(err),
		)

		return fmt.Errorf("failed to calculate shipment timestamps: %w", err)
	}

	if shp.ActualShipDate != nil {
		log.Debug("calculated shipment timestamps",
			zap.Int64("actualShipDate", *shp.ActualShipDate),
		)
	}

	if shp.ActualDeliveryDate != nil {
		log.Debug("calculated shipment timestamps",
			zap.Int64("actualDeliveryDate", *shp.ActualDeliveryDate),
		)
	}

	log.Debug("calculated shipment timestamps")

	return nil
}
