package driversettlementservice

import (
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/shopspring/decimal"
)

var minorFactor = decimal.NewFromInt(100)

func toMinor(amount decimal.Decimal) int64 {
	return amount.Mul(minorFactor).Round(0).IntPart()
}

type moveCalcInput struct {
	Profile           *driverpay.PayProfile
	SplitPercent      decimal.Decimal
	RateOverrides     []driverpay.RateOverride
	Shipment          *shipment.Shipment
	Move              *shipment.ShipmentMove
	TotalTripDistance decimal.Decimal
	MoveCount         int
	HasHazmat         bool
	FuelSurcharge     decimal.Decimal
}

func (in *moveCalcInput) effectiveComponent(
	comp *driverpay.PayProfileComponent,
) (rate decimal.Decimal, bands []driverpay.MileageBand) {
	for _, override := range in.RateOverrides {
		if override.ComponentID == comp.ID {
			return override.Rate, nil
		}
	}
	return comp.Rate, comp.Bands
}

func resolveBandedRate(
	baseRate decimal.Decimal,
	bands []driverpay.MileageBand,
	totalMiles decimal.Decimal,
) decimal.Decimal {
	if len(bands) == 0 {
		return baseRate
	}
	miles := int(totalMiles.IntPart())
	for _, band := range bands {
		if miles >= band.MinMiles && (band.MaxMiles == 0 || miles < band.MaxMiles) {
			return band.Rate
		}
	}
	return baseRate
}

func moveDistance(move *shipment.ShipmentMove) decimal.Decimal {
	if move == nil || move.Distance == nil {
		return decimal.Zero
	}
	return decimal.NewFromFloat(*move.Distance)
}

func shipmentLinehaul(sp *shipment.Shipment) decimal.Decimal {
	if sp == nil || !sp.FreightChargeAmount.Valid {
		return decimal.Zero
	}
	return sp.FreightChargeAmount.Decimal
}

func shipmentTotalRevenue(sp *shipment.Shipment) decimal.Decimal {
	if sp == nil || !sp.TotalChargeAmount.Valid {
		return decimal.Zero
	}
	return sp.TotalChargeAmount.Decimal
}

func shipmentFuelSurcharge(sp *shipment.Shipment) decimal.Decimal {
	if sp == nil {
		return decimal.Zero
	}
	total := decimal.Zero
	for _, charge := range sp.AdditionalCharges {
		if charge == nil || charge.FuelSurchargeProgramID == nil ||
			charge.FuelSurchargeProgramID.IsNil() {
			continue
		}
		total = total.Add(charge.Amount)
	}
	return total
}

func shipmentHasHazmat(sp *shipment.Shipment) bool {
	if sp == nil {
		return false
	}
	for _, commodity := range sp.Commodities {
		if commodity == nil || commodity.Commodity == nil {
			continue
		}
		if !commodity.Commodity.HazardousMaterialID.IsNil() {
			return true
		}
	}
	return false
}

func (in *moveCalcInput) revenueShare() decimal.Decimal {
	distance := moveDistance(in.Move)
	if in.TotalTripDistance.IsPositive() && distance.IsPositive() {
		return distance.Div(in.TotalTripDistance)
	}
	if in.MoveCount > 0 {
		return decimal.NewFromInt(1).Div(decimal.NewFromInt(int64(in.MoveCount)))
	}
	return decimal.NewFromInt(1)
}

func (in *moveCalcInput) revenueForBasis(basis driverpay.RevenueBasis) decimal.Decimal {
	switch basis {
	case driverpay.RevenueBasisLinehaul:
		return shipmentLinehaul(in.Shipment)
	case driverpay.RevenueBasisLinehaulPlusFuelSurcharge:
		return shipmentLinehaul(in.Shipment).Add(in.FuelSurcharge)
	case driverpay.RevenueBasisTotalRevenue:
		return shipmentTotalRevenue(in.Shipment)
	default:
		return shipmentLinehaul(in.Shipment)
	}
}

func computeMovePay(
	in *moveCalcInput,
) (components []driversettlement.PayEventComponent, gross int64) {
	if in.Profile == nil || in.Move == nil {
		return nil, 0
	}
	splitFactor := in.SplitPercent.Div(decimal.NewFromInt(100))
	components = make([]driversettlement.PayEventComponent, 0, len(in.Profile.Components))

	for _, comp := range in.Profile.Components {
		if comp == nil || !comp.IsActive {
			continue
		}
		amount, quantity, rate, ok := computeComponent(in, comp)
		if !ok {
			continue
		}
		amount = applyComponentCaps(comp, amount)
		amount = amount.Mul(splitFactor)
		amountMinor := toMinor(amount)
		if amountMinor == 0 {
			continue
		}
		components = append(components, driversettlement.PayEventComponent{
			Kind:        comp.Kind,
			Method:      comp.Method,
			Description: componentDescription(comp),
			Quantity:    quantity.Round(4),
			Rate:        rate,
			AmountMinor: amountMinor,
		})
		gross += amountMinor
	}

	return components, gross
}

//nolint:cyclop,funlen // exhaustive method dispatch is clearest as a single switch
func computeComponent(
	in *moveCalcInput,
	comp *driverpay.PayProfileComponent,
) (amount, quantity, rate decimal.Decimal, ok bool) {
	distance := moveDistance(in.Move)
	baseRate, bands := in.effectiveComponent(comp)

	switch comp.Method {
	case driverpay.CalcMethodPerLoadedMile:
		if !in.Move.Loaded || !distance.IsPositive() {
			return decimal.Zero, decimal.Zero, decimal.Zero, false
		}
		rate = resolveBandedRate(baseRate, bands, distance)
		return distance.Mul(rate), distance, rate, true
	case driverpay.CalcMethodPerEmptyMile:
		if in.Move.Loaded || !distance.IsPositive() {
			return decimal.Zero, decimal.Zero, decimal.Zero, false
		}
		rate = resolveBandedRate(baseRate, bands, distance)
		return distance.Mul(rate), distance, rate, true
	case driverpay.CalcMethodPerTotalMile:
		if !distance.IsPositive() {
			return decimal.Zero, decimal.Zero, decimal.Zero, false
		}
		rate = resolveBandedRate(baseRate, bands, distance)
		return distance.Mul(rate), distance, rate, true
	case driverpay.CalcMethodPercentOfRevenue:
		if componentRequiresHazmat(comp) && !in.HasHazmat {
			return decimal.Zero, decimal.Zero, decimal.Zero, false
		}
		basisAmount := in.revenueForBasis(comp.RevenueBasis)
		if comp.Kind == driverpay.ComponentKindFuelSurcharge {
			basisAmount = in.FuelSurcharge
		}
		share := in.revenueShare()
		allocated := basisAmount.Mul(share)
		pct := baseRate.Div(decimal.NewFromInt(100))
		return allocated.Mul(pct), allocated, baseRate, true
	case driverpay.CalcMethodFlatPerShipment:
		if componentRequiresHazmat(comp) && !in.HasHazmat {
			return decimal.Zero, decimal.Zero, decimal.Zero, false
		}
		share := in.revenueShare()
		return baseRate.Mul(share), share, baseRate, true
	case driverpay.CalcMethodPerStop:
		extraStops := extraStopCount(in.Move)
		if extraStops <= 0 {
			return decimal.Zero, decimal.Zero, decimal.Zero, false
		}
		qty := decimal.NewFromInt(int64(extraStops))
		return baseRate.Mul(qty), qty, baseRate, true
	case driverpay.CalcMethodPerHour:
		if comp.Kind != driverpay.ComponentKindDetention {
			return decimal.Zero, decimal.Zero, decimal.Zero, false
		}
		hours := detentionHours(in.Move, comp.FreeTimeMinutes)
		if !hours.IsPositive() {
			return decimal.Zero, decimal.Zero, decimal.Zero, false
		}
		return baseRate.Mul(hours), hours, baseRate, true
	case driverpay.CalcMethodPerEvent:
		if comp.Kind == driverpay.ComponentKindHazmat && in.HasHazmat {
			return baseRate, decimal.NewFromInt(1), baseRate, true
		}
		return decimal.Zero, decimal.Zero, decimal.Zero, false
	case driverpay.CalcMethodPerDay:
		return decimal.Zero, decimal.Zero, decimal.Zero, false
	default:
		return decimal.Zero, decimal.Zero, decimal.Zero, false
	}
}

func componentRequiresHazmat(comp *driverpay.PayProfileComponent) bool {
	return comp.Kind == driverpay.ComponentKindHazmat
}

func applyComponentCaps(
	comp *driverpay.PayProfileComponent,
	amount decimal.Decimal,
) decimal.Decimal {
	if comp.MinAmountMinor != nil {
		minAmount := decimal.NewFromInt(*comp.MinAmountMinor).Div(minorFactor)
		if amount.LessThan(minAmount) {
			amount = minAmount
		}
	}
	if comp.MaxAmountMinor != nil {
		maxAmount := decimal.NewFromInt(*comp.MaxAmountMinor).Div(minorFactor)
		if amount.GreaterThan(maxAmount) {
			amount = maxAmount
		}
	}
	return amount
}

func extraStopCount(move *shipment.ShipmentMove) int {
	if move == nil {
		return 0
	}
	return max(len(move.Stops)-2, 0)
}

func detentionHours(move *shipment.ShipmentMove, freeTimeMinutes int) decimal.Decimal {
	if move == nil {
		return decimal.Zero
	}
	totalMinutes := decimal.Zero
	for _, stop := range move.Stops {
		if stop == nil || stop.ActualArrival == nil || stop.ActualDeparture == nil {
			continue
		}
		if stop.CountDetentionOverride != nil && !*stop.CountDetentionOverride {
			continue
		}
		dwellMinutes := (*stop.ActualDeparture - *stop.ActualArrival) / 60
		billable := dwellMinutes - int64(freeTimeMinutes)
		if billable > 0 {
			totalMinutes = totalMinutes.Add(decimal.NewFromInt(billable))
		}
	}
	if !totalMinutes.IsPositive() {
		return decimal.Zero
	}
	return totalMinutes.Div(decimal.NewFromInt(60)).Round(2)
}

func componentDescription(comp *driverpay.PayProfileComponent) string {
	if comp.Description != "" {
		return comp.Description
	}
	switch comp.Kind {
	case driverpay.ComponentKindLinehaul:
		return linehaulDescription(comp.Method)
	case driverpay.ComponentKindFuelSurcharge:
		return "Fuel surcharge pass-through"
	case driverpay.ComponentKindStopPay:
		return "Extra stop pay"
	case driverpay.ComponentKindDetention:
		return "Detention pay"
	case driverpay.ComponentKindLayover:
		return "Layover pay"
	case driverpay.ComponentKindBreakdown:
		return "Breakdown pay"
	case driverpay.ComponentKindTarp:
		return "Tarp pay"
	case driverpay.ComponentKindHazmat:
		return "Hazmat premium"
	case driverpay.ComponentKindBonus:
		return "Bonus pay"
	case driverpay.ComponentKindCustom:
		return "Custom pay"
	default:
		return string(comp.Kind)
	}
}

func linehaulDescription(method driverpay.CalcMethod) string {
	switch method {
	case driverpay.CalcMethodPerLoadedMile:
		return "Linehaul - loaded miles"
	case driverpay.CalcMethodPerEmptyMile:
		return "Linehaul - empty miles"
	case driverpay.CalcMethodPerTotalMile:
		return "Linehaul - total miles"
	case driverpay.CalcMethodPercentOfRevenue:
		return "Linehaul - percent of revenue"
	case driverpay.CalcMethodFlatPerShipment:
		return "Linehaul - flat rate"
	case driverpay.CalcMethodPerStop, driverpay.CalcMethodPerHour,
		driverpay.CalcMethodPerDay, driverpay.CalcMethodPerEvent:
		return "Linehaul"
	default:
		return "Linehaul"
	}
}

func payEventIdempotencyKey(workerID, moveID fmt.Stringer) string {
	return "pay:" + workerID.String() + ":" + moveID.String()
}
