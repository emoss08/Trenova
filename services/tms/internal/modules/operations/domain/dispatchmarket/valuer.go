package dispatchmarket

import (
	"cmp"
	"math"
	"slices"

	"github.com/emoss08/trenova/shared/pulid"
)

const (
	priorObservations = 4.0
	futureDiscount    = 0.8
	cellFallbackShare = 0.5
)

type DriverValue struct {
	Start       Point
	AvailableAt int64
	ValueCents  int64
}

type Sample struct {
	Key        Key
	SumCents   float64
	Drivers    int32
	ObservedAt int64
	Week       int64
}

func Aggregate(grid Grid, observedAt int64, values []DriverValue) []*Sample {
	byKey := make(map[Key]*Sample, len(values))
	keys := make([]Key, 0, len(values))
	week := grid.Week(observedAt)
	for _, value := range values {
		key, ok := grid.Key(value.Start, value.AvailableAt)
		if !ok || value.ValueCents < 0 {
			continue
		}
		sample, seen := byKey[key]
		if !seen {
			sample = &Sample{Key: key, ObservedAt: observedAt, Week: week}
			byKey[key] = sample
			keys = append(keys, key)
		}
		sample.SumCents += float64(value.ValueCents)
		sample.Drivers++
	}
	slices.SortFunc(keys, compareKeys)

	out := make([]*Sample, 0, len(keys))
	for _, key := range keys {
		out = append(out, byKey[key])
	}

	return out
}

func compareKeys(a, b Key) int {
	if c := cmp.Compare(a.Cell, b.Cell); c != 0 {
		return c
	}

	return cmp.Compare(a.Slot, b.Slot)
}

type estimate struct {
	cents        float64
	observations float64
}

func (e estimate) confident() float64 {
	if e.observations <= 0 {
		return 0
	}

	return e.cents * e.observations / (e.observations + priorObservations)
}

type Valuer struct {
	grid  Grid
	slots map[Key]estimate
	cells map[int64]estimate
}

func NewValuer(grid Grid, values []*MarketValue) *Valuer {
	v := &Valuer{
		grid:  grid,
		slots: make(map[Key]estimate, len(values)),
		cells: make(map[int64]estimate),
	}
	sums := make(map[int64]estimate)
	for _, value := range values {
		if value.Observations <= 0 {
			continue
		}
		cents := math.Max(value.Estimate(), 0)
		observations := float64(value.Observations)
		v.slots[value.Key()] = estimate{cents: cents, observations: observations}
		cell := sums[value.Cell]
		cell.cents += cents * observations
		cell.observations += observations
		sums[value.Cell] = cell
	}
	for cell, sum := range sums {
		v.cells[cell] = estimate{
			cents:        sum.cents / sum.observations,
			observations: sum.observations,
		}
	}

	return v
}

func (v *Valuer) Len() int {
	if v == nil {
		return 0
	}

	return len(v.slots)
}

func (v *Valuer) EndMarketCents(latitude, longitude float64, at int64) int64 {
	if v == nil || len(v.slots) == 0 {
		return 0
	}
	key, ok := v.grid.Key(Point{Latitude: latitude, Longitude: longitude}, at)
	if !ok {
		return 0
	}

	value := 0.0
	if slot, slotOK := v.slots[key]; slotOK {
		value = slot.confident()
	} else if cell, cellOK := v.cells[key.Cell]; cellOK {
		value = cell.confident() * cellFallbackShare
	}

	return int64(math.Round(value * futureDiscount))
}

func CellsOf(points []Point) []int64 {
	seen := make(map[int64]struct{}, len(points))
	cells := make([]int64, 0, len(points))
	for _, point := range points {
		cell, ok := CellOf(point)
		if !ok {
			continue
		}
		if _, dup := seen[cell]; dup {
			continue
		}
		seen[cell] = struct{}{}
		cells = append(cells, cell)
	}
	slices.Sort(cells)

	return cells
}

type FoldParams struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	Existing       []*MarketValue
	Samples        []*Sample
}

func FoldAll(p *FoldParams) []*MarketValue {
	byKey := make(map[Key]*MarketValue, len(p.Existing))
	for _, value := range p.Existing {
		byKey[value.Key()] = value
	}

	changed := make([]*MarketValue, 0, len(p.Samples))
	for _, sample := range p.Samples {
		value, ok := byKey[sample.Key]
		if !ok {
			value = &MarketValue{
				OrganizationID: p.OrganizationID,
				BusinessUnitID: p.BusinessUnitID,
				Cell:           sample.Key.Cell,
				Slot:           sample.Key.Slot,
			}
			byKey[sample.Key] = value
		}
		if value.Fold(sample) {
			changed = append(changed, value)
		}
	}

	return changed
}

func SampleCells(samples []*Sample) []int64 {
	seen := make(map[int64]struct{}, len(samples))
	cells := make([]int64, 0, len(samples))
	for _, sample := range samples {
		if _, dup := seen[sample.Key.Cell]; dup {
			continue
		}
		seen[sample.Key.Cell] = struct{}{}
		cells = append(cells, sample.Key.Cell)
	}
	slices.Sort(cells)

	return cells
}
