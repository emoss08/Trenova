package shipment

import (
	"cmp"
	"slices"

	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/shopspring/decimal"
)

func (sp *Shipment) OrderedStops() []*Stop {
	if sp == nil {
		return nil
	}

	type sequenced struct {
		stop    *Stop
		moveSeq int64
	}

	collected := make([]sequenced, 0, len(sp.Moves)*2)
	for _, move := range sp.Moves {
		if move == nil {
			continue
		}
		for _, stop := range move.Stops {
			if stop != nil {
				collected = append(collected, sequenced{stop: stop, moveSeq: move.Sequence})
			}
		}
	}

	slices.SortStableFunc(collected, func(a, b sequenced) int {
		return cmp.Or(
			cmp.Compare(a.moveSeq, b.moveSeq),
			cmp.Compare(a.stop.Sequence, b.stop.Sequence),
			cmp.Compare(a.stop.ID.String(), b.stop.ID.String()),
		)
	})

	stops := make([]*Stop, len(collected))
	for i, entry := range collected {
		stops[i] = entry.stop
	}

	return stops
}

func (sp *Shipment) RatedStops() []*Stop {
	return RatedStopsOf(sp.OrderedStops())
}

func RatedStopsOf(ordered []*Stop) []*Stop {
	rated := make([]*Stop, 0, len(ordered))
	for _, stop := range ordered {
		if !stop.IsCanceled() {
			rated = append(rated, stop)
		}
	}
	return rated
}

func (sp *Shipment) RatedStopCount() int {
	if sp == nil {
		return 0
	}

	count := 0
	for _, move := range sp.Moves {
		if move == nil {
			continue
		}
		for _, stop := range move.Stops {
			if stop != nil && !stop.IsCanceled() {
				count++
			}
		}
	}

	return count
}

func (sp *Shipment) RatingDistance() decimal.Decimal {
	total := decimal.Zero
	if sp == nil {
		return total
	}

	for _, move := range sp.Moves {
		if move != nil && move.Distance != nil {
			total = total.Add(decimal.NewFromFloat(*move.Distance))
		}
	}

	return total
}

func (sp *Shipment) RatingWeight() decimal.Decimal {
	if sp == nil {
		return decimal.Zero
	}

	var total int64
	for _, line := range sp.Commodities {
		if line != nil {
			total += line.Weight
		}
	}
	if total > 0 {
		return decimal.NewFromInt(total)
	}

	if sp.Weight != nil {
		return decimal.NewFromInt(*sp.Weight)
	}

	return decimal.Zero
}

func (sp *Shipment) RatingPieces() int64 {
	if sp == nil {
		return 0
	}

	var total int64
	for _, line := range sp.Commodities {
		if line != nil {
			total += line.Pieces
		}
	}
	if total > 0 {
		return total
	}

	if sp.Pieces != nil {
		return *sp.Pieces
	}

	return 0
}

func (sp *Shipment) RatingLinearFeet() decimal.Decimal {
	total := decimal.Zero
	if sp == nil {
		return total
	}

	for _, line := range sp.Commodities {
		if line == nil || line.Commodity == nil || line.Commodity.LinearFeetPerUnit == nil {
			continue
		}
		total = total.Add(
			decimal.NewFromFloat(*line.Commodity.LinearFeetPerUnit).
				Mul(decimal.NewFromInt(line.Pieces)),
		)
	}

	return total
}

func (sc *ShipmentCommodity) CubicFeet() decimal.Decimal {
	if sc == nil || !sc.HasDimensions() || sc.Pieces <= 0 {
		return decimal.Zero
	}

	return decimal.NewFromFloat(sc.ResolvedLengthFeet()).
		Mul(decimal.NewFromFloat(sc.ResolvedWidthFeet())).
		Mul(decimal.NewFromFloat(sc.ResolvedHeightFeet())).
		Mul(decimal.NewFromInt(sc.Pieces))
}

func (sp *Shipment) RatingCubicFeet() decimal.Decimal {
	if sp == nil {
		return decimal.Zero
	}

	if envelope := sp.CurrentEnvelope(); envelope.LengthFeet > 0 && envelope.WidthFeet > 0 &&
		envelope.HeightFeet > 0 {
		return decimal.NewFromFloat(envelope.LengthFeet).
			Mul(decimal.NewFromFloat(envelope.WidthFeet)).
			Mul(decimal.NewFromFloat(envelope.HeightFeet))
	}

	total := decimal.Zero
	for _, line := range sp.Commodities {
		total = total.Add(line.CubicFeet())
	}

	return total
}

func (sp *Shipment) RatingDensity() (decimal.Decimal, bool) {
	cube := sp.RatingCubicFeet()
	if !cube.IsPositive() {
		return decimal.Zero, false
	}

	return sp.RatingWeight().Div(cube), true
}

func (sp *Shipment) HasHazardousMaterial() bool {
	if sp == nil {
		return false
	}

	for _, line := range sp.Commodities {
		if line == nil || line.Commodity == nil {
			continue
		}
		if line.Commodity.HazardousMaterialID.IsNotNil() ||
			line.Commodity.HazardousMaterial != nil {
			return true
		}
	}

	return false
}

func (sp *Shipment) FreightClasses() []commodity.FreightClass {
	if sp == nil {
		return nil
	}

	classes := make([]commodity.FreightClass, 0, len(sp.Commodities))
	seen := make(map[commodity.FreightClass]struct{}, len(sp.Commodities))
	for _, line := range sp.Commodities {
		if line == nil || line.Commodity == nil || line.Commodity.FreightClass == "" {
			continue
		}
		class := line.Commodity.FreightClass
		if _, dup := seen[class]; dup {
			continue
		}
		seen[class] = struct{}{}
		classes = append(classes, class)
	}

	return classes
}

func (sp *Shipment) PrimaryFreightClass() commodity.FreightClass {
	if sp == nil {
		return ""
	}

	var primary commodity.FreightClass
	heaviest := int64(-1)
	for _, line := range sp.Commodities {
		if line == nil || line.Commodity == nil || !line.Commodity.FreightClass.IsValid() {
			continue
		}
		if line.Weight > heaviest {
			heaviest = line.Weight
			primary = line.Commodity.FreightClass
		}
	}

	return primary
}

func (sp *Shipment) HighestFreightClass() commodity.FreightClass {
	var highest commodity.FreightClass
	highestRank := -1
	for _, class := range sp.FreightClasses() {
		if rank := slices.Index(commodity.FreightClasses, class); rank > highestRank {
			highestRank = rank
			highest = class
		}
	}

	return highest
}
