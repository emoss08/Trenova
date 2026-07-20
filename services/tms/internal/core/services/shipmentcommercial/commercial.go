//nolint:gocritic // existing value-shaped APIs and hot-path helpers are intentionally stable
package shipmentcommercial

import (
	"context"
	"math"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/maputils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Formula         services.FormulaCalculator
	AccessorialRepo repositories.AccessorialChargeRepository
	FuelSurcharge   services.FuelSurchargeResolver
}

type Calculator struct {
	formula         services.FormulaCalculator
	accessorialRepo repositories.AccessorialChargeRepository
	fuelSurcharge   services.FuelSurchargeResolver
	now             func() int64
}

func New(p Params) *Calculator {
	return &Calculator{
		formula:         p.Formula,
		accessorialRepo: p.AccessorialRepo,
		fuelSurcharge:   p.FuelSurcharge,
		now:             timeutils.NowUnix,
	}
}

type chargeSyncOptions struct {
	detention bool
	fuel      bool
}

func (c *Calculator) Recalculate(
	ctx context.Context,
	entity *shipment.Shipment,
	control *tenant.ShipmentControl,
	userID pulid.ID,
) error {
	baseCharge, otherChargeAmount, ratingDetail, err := c.calculateCommercialTotals(
		ctx,
		entity,
		control,
		userID,
		chargeSyncOptions{detention: true, fuel: true},
	)
	if err != nil {
		return err
	}

	entity.FreightChargeAmount = decimal.NewNullDecimal(baseCharge)
	entity.OtherChargeAmount = decimal.NewNullDecimal(otherChargeAmount)
	entity.TotalChargeAmount = decimal.NewNullDecimal(baseCharge.Add(otherChargeAmount))
	entity.RatingDetail = ratingDetail

	return nil
}

func (c *Calculator) CalculateTotals(
	ctx context.Context,
	entity *shipment.Shipment,
	control *tenant.ShipmentControl,
	userID pulid.ID,
) (*repositories.ShipmentTotalsResponse, error) {
	baseCharge, otherChargeAmount, _, err := c.calculateCommercialTotals(
		ctx,
		entity,
		control,
		userID,
		chargeSyncOptions{fuel: true},
	)
	if err != nil {
		return nil, err
	}

	return &repositories.ShipmentTotalsResponse{
		FreightChargeAmount: baseCharge,
		OtherChargeAmount:   otherChargeAmount,
		TotalChargeAmount:   baseCharge.Add(otherChargeAmount),
		FuelSurcharge:       findGeneratedFuelSurchargeCharge(entity),
	}, nil
}

func findGeneratedFuelSurchargeCharge(entity *shipment.Shipment) *shipment.AdditionalCharge {
	for _, charge := range entity.AdditionalCharges {
		if charge != nil && charge.IsSystemGenerated && charge.FuelSurchargeProgramID != nil {
			return charge
		}
	}
	return nil
}

func CalculateAdditionalCharges(
	charges []*shipment.AdditionalCharge,
	baseCharge decimal.Decimal,
) decimal.Decimal {
	total := decimal.Zero
	for _, charge := range charges {
		if charge == nil {
			continue
		}

		total = total.Add(CalculateAdditionalCharge(charge, baseCharge))
	}

	return total
}

func (c *Calculator) calculateCommercialTotals(
	ctx context.Context,
	entity *shipment.Shipment,
	control *tenant.ShipmentControl,
	userID pulid.ID,
	sync chargeSyncOptions,
) (decimal.Decimal, decimal.Decimal, *shipment.RatingDetail, error) {
	if sync.detention {
		if err := c.syncDetentionCharge(ctx, entity, control); err != nil {
			return decimal.Zero, decimal.Zero, nil, err
		}
	}

	baseCharge, ratingDetail, err := c.calculateBaseCharge(ctx, entity, userID)
	if err != nil {
		return decimal.Zero, decimal.Zero, nil, err
	}

	if sync.fuel {
		if err = c.syncFuelSurcharge(ctx, entity, baseCharge); err != nil {
			return decimal.Zero, decimal.Zero, nil, err
		}
	}

	return baseCharge, CalculateAdditionalCharges(
		entity.AdditionalCharges,
		baseCharge,
	), ratingDetail, nil
}

func CalculateAdditionalCharge(
	charge *shipment.AdditionalCharge,
	baseCharge decimal.Decimal,
) decimal.Decimal {
	if charge == nil {
		return decimal.Zero
	}

	switch charge.Method {
	case accessorialcharge.MethodFlat:
		unit := max(charge.Unit, 1)
		return charge.Amount.Mul(decimal.NewFromInt32(int32(unit)))
	case accessorialcharge.MethodPerUnit:
		if charge.Unit < 1 {
			return decimal.Zero
		}
		return charge.Amount.Mul(decimal.NewFromInt32(int32(charge.Unit)))
	case accessorialcharge.MethodPercentage:
		return baseCharge.Mul(charge.Amount.Div(decimal.NewFromInt(100)))
	default:
		return decimal.Zero
	}
}

func (c *Calculator) calculateBaseCharge(
	ctx context.Context,
	entity *shipment.Shipment,
	userID pulid.ID,
) (decimal.Decimal, *shipment.RatingDetail, error) {
	resp, err := c.formula.Calculate(ctx, &formulatemplatetypes.CalculateRequest{
		TemplateID: entity.FormulaTemplateID,
		Entity:     entity,
		TenantInfo: pagination.TenantInfo{
			OrgID:  entity.OrganizationID,
			BuID:   entity.BusinessUnitID,
			UserID: userID,
		},
		RatingDate: ratingDate(entity, c.now),
	})
	if err != nil {
		return decimal.Zero, nil, err
	}

	result, _ := resp.Amount.Float64()
	detail := &shipment.RatingDetail{
		FormulaTemplateID:   resp.FormulaTemplateID,
		FormulaTemplateName: resp.FormulaTemplateName,
		Expression:          resp.Expression,
		ResolvedVariables:   maputils.WithoutFuncValues(resp.Variables),
		Result:              result,
		RatedAt:             c.now(),
		VersionNumber:       resp.VersionNumber,
		Breakdown:           ratingBreakdown(resp.Breakdown),
		Guardrail:           ratingGuardrail(resp.Guardrail),
	}

	return resp.Amount, detail, nil
}

func ratingDate(entity *shipment.Shipment, now func() int64) int64 {
	if entity.ActualShipDate != nil && *entity.ActualShipDate > 0 {
		return *entity.ActualShipDate
	}
	if entity.CreatedAt > 0 {
		return entity.CreatedAt
	}
	return now()
}

func ratingBreakdown(
	items []formulatemplatetypes.BreakdownAmount,
) []shipment.RatingBreakdownItem {
	if len(items) == 0 {
		return nil
	}

	breakdown := make([]shipment.RatingBreakdownItem, 0, len(items))
	for _, item := range items {
		amount, _ := item.Amount.Float64()
		breakdown = append(breakdown, shipment.RatingBreakdownItem{
			Name:   item.Name,
			Label:  item.Label,
			Amount: amount,
			Error:  item.Error,
		})
	}

	return breakdown
}

func ratingGuardrail(result *formulatemplatetypes.GuardrailResult) *shipment.RatingGuardrail {
	if result == nil {
		return nil
	}

	raw, _ := result.RawAmount.Float64()
	guardrail := &shipment.RatingGuardrail{
		Applied:   result.Applied,
		Bound:     result.Bound,
		RawResult: raw,
	}

	if result.MinCharge != nil {
		minCharge, _ := result.MinCharge.Float64()
		guardrail.MinCharge = &minCharge
	}
	if result.MaxCharge != nil {
		maxCharge, _ := result.MaxCharge.Float64()
		guardrail.MaxCharge = &maxCharge
	}

	return guardrail
}

func (c *Calculator) syncDetentionCharge(
	ctx context.Context,
	entity *shipment.Shipment,
	control *tenant.ShipmentControl,
) error {
	if entity == nil || control == nil {
		return nil
	}

	if !control.TrackDetentionTime ||
		!control.AutoGenerateDetentionCharges ||
		control.DetentionChargeID == nil ||
		control.DetentionChargeID.IsNil() {
		return nil
	}

	accessorial, err := c.accessorialRepo.GetByID(ctx, repositories.GetAccessorialChargeByIDRequest{
		ID: *control.DetentionChargeID,
		TenantInfo: &pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
	if err != nil {
		return err
	}

	thresholdMinutes := shipmentstate.DefaultDelayThresholdMinutes
	if control.DetentionThreshold != nil {
		thresholdMinutes = shipmentstate.ResolveDelayThresholdMinutes(*control.DetentionThreshold)
	}

	totalExcessSeconds, detainedStopCount := detentionExposure(entity, c.now(), thresholdMinutes)
	if totalExcessSeconds <= 0 || detainedStopCount == 0 {
		removeGeneratedDetentionCharges(entity, accessorial.ID)
		return nil
	}

	unit := max(detentionUnits(accessorial.RateUnit, totalExcessSeconds, detainedStopCount), 1)

	ensureGeneratedDetentionCharge(entity, accessorial, unit)
	return nil
}

func detentionExposure(
	entity *shipment.Shipment,
	currentTime int64,
	thresholdMinutes int16,
) (int64, int16) {
	thresholdSeconds := int64(shipmentstate.ResolveDelayThresholdMinutes(thresholdMinutes)) * 60
	var totalExcessSeconds int64
	var detainedStopCount int16

	for _, move := range entity.Moves {
		if move == nil {
			continue
		}

		for _, stop := range move.Stops {
			if stop == nil || stop.Status == shipment.StopStatusCanceled ||
				stop.ActualArrival == nil {
				continue
			}

			endTime := currentTime
			if stop.ActualDeparture != nil && *stop.ActualDeparture > 0 {
				endTime = *stop.ActualDeparture
			}

			if endTime <= *stop.ActualArrival {
				continue
			}

			excessSeconds := endTime - *stop.ActualArrival - thresholdSeconds
			if excessSeconds <= 0 {
				continue
			}

			totalExcessSeconds += excessSeconds
			detainedStopCount++
		}
	}

	return totalExcessSeconds, detainedStopCount
}

func detentionUnits(
	rateUnit accessorialcharge.RateUnit,
	totalExcessSeconds int64,
	detainedStopCount int16,
) int16 {
	//nolint:exhaustive // only actionable enum states require explicit handling here
	switch rateUnit {
	case accessorialcharge.RateUnitHour:
		return int16(math.Ceil(float64(totalExcessSeconds) / 3600))
	case accessorialcharge.RateUnitDay:
		return int16(math.Ceil(float64(totalExcessSeconds) / 86400))
	case accessorialcharge.RateUnitStop:
		return detainedStopCount
	default:
		return 1
	}
}

func (c *Calculator) syncFuelSurcharge(
	ctx context.Context,
	entity *shipment.Shipment,
	baseCharge decimal.Decimal,
) error {
	if entity == nil || c.fuelSurcharge == nil {
		return nil
	}

	if entity.FuelSurchargeLocked {
		return nil
	}

	resolved, err := c.fuelSurcharge.ResolveShipmentCharge(
		ctx,
		&services.ResolveShipmentChargeRequest{
			Shipment:         entity,
			Linehaul:         baseCharge,
			AccessorialTotal: nonFuelSurchargeChargeTotal(entity, baseCharge),
		},
	)
	if err != nil {
		return err
	}

	if resolved == nil || resolved.Amount.LessThanOrEqual(decimal.Zero) {
		removeGeneratedFuelSurchargeCharges(entity)
		return nil
	}

	ensureGeneratedFuelSurchargeCharge(entity, resolved)
	return nil
}

func nonFuelSurchargeChargeTotal(
	entity *shipment.Shipment,
	baseCharge decimal.Decimal,
) decimal.Decimal {
	total := decimal.Zero
	for _, charge := range entity.AdditionalCharges {
		if charge == nil || (charge.IsSystemGenerated && charge.FuelSurchargeProgramID != nil) {
			continue
		}
		total = total.Add(CalculateAdditionalCharge(charge, baseCharge))
	}
	return total
}

func removeGeneratedFuelSurchargeCharges(entity *shipment.Shipment) {
	filtered := entity.AdditionalCharges[:0]
	for _, charge := range entity.AdditionalCharges {
		if charge == nil {
			continue
		}
		if charge.IsSystemGenerated && charge.FuelSurchargeProgramID != nil {
			continue
		}
		filtered = append(filtered, charge)
	}
	entity.AdditionalCharges = filtered
}

func ensureGeneratedFuelSurchargeCharge(
	entity *shipment.Shipment,
	resolved *services.ResolvedFuelSurcharge,
) {
	var generated *shipment.AdditionalCharge
	filtered := make([]*shipment.AdditionalCharge, 0, len(entity.AdditionalCharges))

	for _, charge := range entity.AdditionalCharges {
		if charge == nil {
			continue
		}

		if !charge.IsSystemGenerated || charge.FuelSurchargeProgramID == nil {
			filtered = append(filtered, charge)
			continue
		}

		if generated == nil {
			generated = charge
		}
	}

	if generated == nil {
		generated = &shipment.AdditionalCharge{
			OrganizationID:    entity.OrganizationID,
			BusinessUnitID:    entity.BusinessUnitID,
			ShipmentID:        entity.ID,
			IsSystemGenerated: true,
		}
	}

	programID := resolved.ProgramID
	generated.IsSystemGenerated = true
	generated.AccessorialChargeID = resolved.AccessorialChargeID
	generated.Method = accessorialcharge.MethodFlat
	generated.Amount = resolved.Amount
	generated.Unit = 1
	generated.FuelSurchargeProgramID = &programID
	generated.FuelSurchargeDetail = resolved.Detail

	filtered = append(filtered, generated)
	entity.AdditionalCharges = filtered
}

func removeGeneratedDetentionCharges(entity *shipment.Shipment, detentionChargeID pulid.ID) {
	filtered := entity.AdditionalCharges[:0]
	for _, charge := range entity.AdditionalCharges {
		if charge == nil {
			continue
		}
		if charge.AccessorialChargeID == detentionChargeID && charge.IsSystemGenerated {
			continue
		}
		filtered = append(filtered, charge)
	}
	entity.AdditionalCharges = filtered
}

func ensureGeneratedDetentionCharge(
	entity *shipment.Shipment,
	accessorial *accessorialcharge.AccessorialCharge,
	unit int16,
) {
	var generated *shipment.AdditionalCharge
	filtered := make([]*shipment.AdditionalCharge, 0, len(entity.AdditionalCharges))

	for _, charge := range entity.AdditionalCharges {
		if charge == nil {
			continue
		}

		if charge.AccessorialChargeID != accessorial.ID || !charge.IsSystemGenerated {
			filtered = append(filtered, charge)
			continue
		}

		if generated == nil {
			generated = charge
		}
	}

	if generated == nil {
		generated = &shipment.AdditionalCharge{
			OrganizationID:      entity.OrganizationID,
			BusinessUnitID:      entity.BusinessUnitID,
			ShipmentID:          entity.ID,
			AccessorialChargeID: accessorial.ID,
			IsSystemGenerated:   true,
			Method:              accessorial.Method,
			Amount:              accessorial.Amount,
		}
	}

	generated.IsSystemGenerated = true
	generated.Unit = unit

	filtered = append(filtered, generated)
	entity.AdditionalCharges = filtered
}
