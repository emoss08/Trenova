package resolver

import (
	"errors"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/shopspring/decimal"
)

var ErrNilPointer = errors.New("nil pointer")

func RegisterDefaultComputed(r *Resolver) {
	r.RegisterComputed("computeTotalDistance", computeTotalDistance)
	r.RegisterComputed("computeTotalStops", computeTotalStops)
	r.RegisterComputed("computeHasHazmat", computeHasHazmat)
	r.RegisterComputed("computeRequiresTemperatureControl", computeRequiresTemperatureControl)
	r.RegisterComputed("computeTemperatureDifferential", computeTemperatureDifferential)
	r.RegisterComputed("computeTotalWeight", computeTotalWeight)
	r.RegisterComputed("computeTotalPieces", computeTotalPieces)
	r.RegisterComputed("computeTotalLinearFeet", computeTotalLinearFeet)
	r.RegisterComputed("computeTotalHours", computeTotalHours)
	r.RegisterComputed("computeStops", computeStops)
	r.RegisterComputed("computeCommodities", computeCommodities)
	r.RegisterComputed("computeTotalCubicFeet", computeTotalCubicFeet)
	r.RegisterComputed("computeDensity", computeDensity)
	r.RegisterComputed("computePrimaryFreightClass", computePrimaryFreightClass)
	r.RegisterComputed("computeHighestFreightClass", computeHighestFreightClass)
	registerLaneComputed(r)
	r.RegisterComputed("computeBaseRate", computeBaseRate)
	r.RegisterComputed("computeFreightChargeAmount", computeFreightChargeAmount)
	r.RegisterComputed("computeOtherChargeAmount", computeOtherChargeAmount)
	r.RegisterComputed("computeCurrentTotalCharge", computeCurrentTotalCharge)
}

func fromShipment(zero any, read func(*shipment.Shipment) any) ComputedFunc {
	return func(entity any) (any, error) {
		entry, err := shipmentOf(entity)
		if err != nil {
			return zero, err
		}
		return read(entry), nil
	}
}

var (
	computeTotalDistance = fromShipment(0.0, func(s *shipment.Shipment) any {
		return s.RatingDistance().InexactFloat64()
	})
	computeTotalStops = fromShipment(0, func(s *shipment.Shipment) any {
		return s.RatedStopCount()
	})
	computeHasHazmat = fromShipment(false, func(s *shipment.Shipment) any {
		return s.HasHazardousMaterial()
	})
	computeRequiresTemperatureControl = fromShipment(false, func(s *shipment.Shipment) any {
		return s.TemperatureMin != nil || s.TemperatureMax != nil
	})
	computeTemperatureDifferential = fromShipment(0.0, func(s *shipment.Shipment) any {
		if s.TemperatureMin == nil || s.TemperatureMax == nil {
			return 0.0
		}
		return float64(*s.TemperatureMax) - float64(*s.TemperatureMin)
	})
	computeTotalWeight = fromShipment(0.0, func(s *shipment.Shipment) any {
		return s.RatingWeight().InexactFloat64()
	})
	computeTotalPieces = fromShipment(int64(0), func(s *shipment.Shipment) any {
		return s.RatingPieces()
	})
	computeTotalLinearFeet = fromShipment(0.0, func(s *shipment.Shipment) any {
		return s.RatingLinearFeet().InexactFloat64()
	})
	computeBaseRate = fromShipment(0.0, func(s *shipment.Shipment) any {
		return nullDecimalFloat(s.BaseRate)
	})
	computeFreightChargeAmount = fromShipment(0.0, func(s *shipment.Shipment) any {
		return nullDecimalFloat(s.FreightChargeAmount)
	})
	computeOtherChargeAmount = fromShipment(0.0, func(s *shipment.Shipment) any {
		return shipment.AdditionalChargesTotal(
			s.AdditionalCharges,
			s.FreightChargeAmount.Decimal,
		).InexactFloat64()
	})
	computeCurrentTotalCharge = fromShipment(0.0, func(s *shipment.Shipment) any {
		return nullDecimalFloat(s.TotalChargeAmount)
	})
)

func nullDecimalFloat(value decimal.NullDecimal) float64 {
	if !value.Valid {
		return 0
	}
	return value.Decimal.InexactFloat64()
}

func computeTotalHours(entity any) (any, error) {
	_, rated, err := stopsOf(entity)
	if err != nil {
		return 0.0, err
	}

	var (
		windowStart int64
		windowEnd   int64
		found       bool
	)

	for _, stop := range rated {
		start, ok := stopWindowStart(stop)
		if !ok {
			continue
		}

		end, hasEnd := stopWindowEnd(stop)
		if !hasEnd {
			end = start
		}

		if !found || start < windowStart {
			windowStart = start
		}
		if !found || end > windowEnd {
			windowEnd = end
		}
		found = true
	}

	if !found || windowEnd <= windowStart {
		return 0.0, nil
	}

	return float64(windowEnd-windowStart) / 3600.0, nil
}
