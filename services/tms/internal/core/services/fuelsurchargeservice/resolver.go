package fuelsurchargeservice

import (
	"context"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func (s *Service) ResolveShipmentCharge(
	ctx context.Context,
	req *services.ResolveShipmentChargeRequest,
) (*services.ResolvedFuelSurcharge, error) {
	if req == nil || req.Shipment == nil || req.Shipment.CustomerID.IsNil() {
		return nil, nil
	}

	entity := req.Shipment

	log := s.l.With(
		zap.String("operation", "ResolveShipmentCharge"),
		zap.String("shipmentId", entity.ID.String()),
	)

	profile, err := s.customerRepo.GetBillingProfile(ctx, entity.CustomerID)
	if err != nil {
		log.Warn("failed to load customer billing profile for fuel surcharge", zap.Error(err))
		return nil, nil
	}

	if profile == nil || !profile.AppliesFuelSurcharge() {
		return nil, nil
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}

	program, err := s.programRepo.GetByID(ctx, &repositories.GetFuelSurchargeProgramByIDRequest{
		ProgramID:    *profile.FuelSurchargeProgramID,
		TenantInfo:   tenantInfo,
		IncludeRows:  true,
		IncludeIndex: true,
	})
	if err != nil {
		log.Warn("failed to load fuel surcharge program", zap.Error(err))
		return nil, nil
	}

	basisDate := s.resolveBasisDate(ctx, entity, program)

	if !programApplies(program, entity, basisDate) {
		return nil, nil
	}

	prices, err := s.priceRepo.GetLatestOnOrBefore(ctx, &repositories.GetLatestFuelPricesRequest{
		FuelIndexID: program.FuelIndexID,
		TenantInfo:  tenantInfo,
		Date:        basisDate.Format(fuelsurcharge.PriceDateLayout),
		Limit:       3,
	})
	if err != nil {
		log.Warn("failed to load fuel index prices", zap.Error(err))
		return nil, nil
	}

	price, usedFallback, ok := SelectPrice(prices, program, basisDate)
	if !ok {
		log.Warn("no applicable fuel price for shipment",
			zap.String("programId", program.ID.String()),
			zap.String("basisDate", basisDate.Format(fuelsurcharge.PriceDateLayout)),
		)
		return nil, nil
	}

	miles := shipmentMiles(entity)
	result, err := ComputeCharge(ComputeChargeInput{
		Program:          program,
		Price:            price.Price,
		Miles:            miles,
		Linehaul:         req.Linehaul,
		AccessorialTotal: req.AccessorialTotal,
	})
	if err != nil {
		log.Warn("failed to compute fuel surcharge", zap.Error(err))
		return nil, nil
	}

	detail := buildDetail(detailParams{
		program:          program,
		price:            price,
		result:           result,
		miles:            miles,
		linehaul:         req.Linehaul,
		accessorialTotal: req.AccessorialTotal,
		basisDate:        basisDate,
		usedFallback:     usedFallback,
		now:              s.now(),
	})

	return &services.ResolvedFuelSurcharge{
		ProgramID:           program.ID,
		AccessorialChargeID: program.AccessorialChargeID,
		Amount:              result.Amount,
		Detail:              detail,
	}, nil
}

func (s *Service) resolveBasisDate(
	ctx context.Context,
	entity *shipment.Shipment,
	program *fuelsurcharge.FuelSurchargeProgram,
) time.Time {
	loc := s.tenantLocation(ctx, entity.OrganizationID)

	var basis int64
	switch program.DateBasis {
	case fuelsurcharge.DateBasisPickupDate:
		basis = pickupTimestamp(entity)
	case fuelsurcharge.DateBasisTenderDate:
		basis = entity.CreatedAt
	default:
		basis = pickupTimestamp(entity)
	}

	if basis == 0 {
		basis = s.now()
	}

	t := time.Unix(basis, 0).In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func (s *Service) tenantLocation(ctx context.Context, orgID pulid.ID) *time.Location {
	org, err := s.orgCacheRepo.GetByID(ctx, orgID)
	if err != nil || org == nil {
		return time.UTC
	}

	loc, err := time.LoadLocation(timeutils.NormalizeTimezone(org.Timezone))
	if err != nil {
		return time.UTC
	}

	return loc
}

func pickupTimestamp(entity *shipment.Shipment) int64 {
	if entity.ActualShipDate != nil && *entity.ActualShipDate > 0 {
		return *entity.ActualShipDate
	}

	var earliest int64
	for _, move := range entity.Moves {
		if move == nil || move.Status == shipment.MoveStatusCanceled {
			continue
		}

		for _, stop := range move.Stops {
			if stop == nil || stop.Status == shipment.StopStatusCanceled {
				continue
			}
			if stop.ScheduledWindowStart > 0 &&
				(earliest == 0 || stop.ScheduledWindowStart < earliest) {
				earliest = stop.ScheduledWindowStart
			}
		}
	}

	if earliest > 0 {
		return earliest
	}

	return entity.CreatedAt
}

func programApplies(
	program *fuelsurcharge.FuelSurchargeProgram,
	entity *shipment.Shipment,
	basisDate time.Time,
) bool {
	if program.Status != fuelsurcharge.ProgramStatusActive {
		return false
	}

	basisUnix := basisDate.Unix()
	if program.EffectiveStartDate != nil && basisUnix < *program.EffectiveStartDate {
		return false
	}
	if program.EffectiveEndDate != nil && basisUnix > *program.EffectiveEndDate {
		return false
	}

	return idListMatches(program.ShipmentTypeIDs, entity.ShipmentTypeID) &&
		idListMatches(program.ServiceTypeIDs, entity.ServiceTypeID) &&
		idListMatches(program.TractorTypeIDs, entity.TractorTypeID) &&
		idListMatches(program.TrailerTypeIDs, entity.TrailerTypeID)
}

func idListMatches(allowed []pulid.ID, id pulid.ID) bool {
	if len(allowed) == 0 {
		return true
	}
	if id.IsNil() {
		return false
	}
	return slices.Contains(allowed, id)
}

func shipmentMiles(entity *shipment.Shipment) decimal.Decimal {
	total := decimal.Zero
	for _, move := range entity.Moves {
		if move == nil || move.Status == shipment.MoveStatusCanceled || move.Distance == nil {
			continue
		}
		total = total.Add(decimal.NewFromFloat(*move.Distance))
	}
	return total
}

type detailParams struct {
	program          *fuelsurcharge.FuelSurchargeProgram
	price            *fuelsurcharge.FuelIndexPrice
	result           ChargeResult
	miles            decimal.Decimal
	linehaul         decimal.Decimal
	accessorialTotal decimal.Decimal
	basisDate        time.Time
	usedFallback     bool
	now              int64
}

func buildDetail(p detailParams) *shipment.FuelSurchargeDetail {
	detail := &shipment.FuelSurchargeDetail{
		ProgramID:     p.program.ID.String(),
		ProgramName:   p.program.Name,
		ProgramCode:   p.program.Code,
		Method:        p.program.Method.String(),
		IndexID:       p.program.FuelIndexID.String(),
		PriceDate:     p.price.PriceDate,
		Price:         p.price.Price.InexactFloat64(),
		Currency:      p.price.Currency,
		RawAmount:     p.result.RawAmount.InexactFloat64(),
		Amount:        p.result.Amount.InexactFloat64(),
		CapApplied:    p.result.CapApplied,
		FloorApplied:  p.result.FloorApplied,
		StepRounding:  p.program.StepRounding.String(),
		RateRounding:  p.program.RateRounding.String(),
		RatePrecision: p.program.RatePrecision,
		DateBasis:     p.program.DateBasis.String(),
		BasisDate:     p.basisDate.Format(fuelsurcharge.PriceDateLayout),
		UsedFallback:  p.usedFallback,
		Stale:         IsStalePrice(p.price, p.basisDate),
		CalculatedAt:  p.now,
	}

	if p.program.FuelIndex != nil {
		detail.IndexCode = p.program.FuelIndex.Code
		detail.IndexSource = p.program.FuelIndex.Source.String()
		detail.IndexRegion = p.program.FuelIndex.Region
		detail.IndexFuelType = p.program.FuelIndex.FuelType.String()
		detail.EIASeriesID = p.program.FuelIndex.EIASeriesID
	}

	if p.program.PegPrice.Valid {
		detail.PegPrice = floatPtr(p.program.PegPrice.Decimal)
	}
	if p.program.Increment.Valid {
		detail.Increment = floatPtr(p.program.Increment.Decimal)
	}
	if p.program.IncrementRate.Valid {
		detail.IncrementRate = floatPtr(p.program.IncrementRate.Decimal)
	}
	if p.program.MilesPerGallon.Valid {
		detail.MilesPerGallon = floatPtr(p.program.MilesPerGallon.Decimal)
	}

	if p.result.MatchedRow != nil {
		if p.result.MatchedRow.PriceMin.Valid {
			detail.BandMin = floatPtr(p.result.MatchedRow.PriceMin.Decimal)
		}
		if p.result.MatchedRow.PriceMax.Valid {
			detail.BandMax = floatPtr(p.result.MatchedRow.PriceMax.Decimal)
		}
		detail.BandValue = floatPtr(p.result.MatchedRow.Value)
	}

	if p.result.RatePerMile != nil {
		detail.RatePerMile = floatPtr(*p.result.RatePerMile)
		detail.Miles = floatPtr(p.miles)
	}
	if p.result.Percent != nil {
		detail.Percent = floatPtr(*p.result.Percent)
		detail.PercentBasis = p.program.PercentBasis.String()
		detail.LinehaulBase = floatPtr(p.linehaul)
		if p.program.PercentBasis == fuelsurcharge.PercentBasisLinehaulPlusAccessorials {
			detail.AccessorialBase = floatPtr(p.accessorialTotal)
		}
	}

	return detail
}

func floatPtr(d decimal.Decimal) *float64 {
	f := d.InexactFloat64()
	return &f
}
