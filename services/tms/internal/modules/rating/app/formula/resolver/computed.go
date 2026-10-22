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
	computeTotalDistance = fromShipment(decimal.Zero, func(s *shipment.Shipment) any {
		return s.RatingDistance()
	})
	computeTotalStops = fromShipment(int64(0), func(s *shipment.Shipment) any {
		return int64(s.RatedStopCount())
	})
	computeHasHazmat = fromShipment(false, func(s *shipment.Shipment) any {
		return s.HasHazardousMaterial()
	})
	computeRequiresTemperatureControl = fromShipment(false, func(s *shipment.Shipment) any {
		return s.TemperatureMin != nil || s.TemperatureMax != nil
	})
	computeTemperatureDifferential = fromShipment(int64(0), func(s *shipment.Shipment) any {
		if s.TemperatureMin == nil || s.TemperatureMax == nil {
			return int64(0)
		}
		return int64(*s.TemperatureMax) - int64(*s.TemperatureMin)
	})
	computeTotalWeight = fromShipment(decimal.Zero, func(s *shipment.Shipment) any {
		return s.RatingWeight()
	})
	computeTotalPieces = fromShipment(int64(0), func(s *shipment.Shipment) any {
		return s.RatingPieces()
	})
	computeTotalLinearFeet = fromShipment(decimal.Zero, func(s *shipment.Shipment) any {
		return s.RatingLinearFeet()
	})
	computeBaseRate = fromShipment(decimal.Zero, func(s *shipment.Shipment) any {
		return decimalOrZero(s.BaseRate)
	})
	computeFreightChargeAmount = fromShipment(decimal.Zero, func(s *shipment.Shipment) any {
		return decimalOrZero(s.FreightChargeAmount)
	})
	computeOtherChargeAmount = fromShipment(decimal.Zero, func(s *shipment.Shipment) any {
		return decimalOrZero(s.OtherChargeAmount)
	})
	computeCurrentTotalCharge = fromShipment(decimal.Zero, func(s *shipment.Shipment) any {
		return decimalOrZero(s.TotalChargeAmount)
	})
)

var secondsPerHour = decimal.NewFromInt(3600)

func decimalOrZero(value decimal.NullDecimal) decimal.Decimal {
	if !value.Valid {
		return decimal.Zero
	}
	return value.Decimal
}

func computeTotalHours(entity any) (any, error) {
	_, rated, err := stopsOf(entity)
	if err != nil {
		return decimal.Zero, err
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
		return decimal.Zero, nil
	}

	return decimal.NewFromInt(windowEnd - windowStart).Div(secondsPerHour), nil
}
