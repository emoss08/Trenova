package dispatchmarket

import (
	"math"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/maptile"
)

const (
	CellZoom     = maptile.Zoom(8)
	SlotSeconds  = int64(4 * 3600)
	SlotsPerWeek = int16(42)
	WeekSeconds  = int64(7 * 24 * 3600)
	mondayOffset = int64(4 * 24 * 3600)
	minAlpha     = 0.2
)

type Point struct {
	Latitude  float64
	Longitude float64
}

type Key struct {
	Cell int64
	Slot int16
}

type MarketValue struct {
	keel.Table `keel:"dispatch_market_values"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:dmv_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	Cell           int64    `json:"cell"           keel:"cell"`
	Slot           int16    `json:"slot"           keel:"slot"`
	ValueCents     float64  `json:"valueCents"     keel:"value_cents"`
	Weeks          int32    `json:"weeks"          keel:"weeks"`
	WeekIndex      int64    `json:"weekIndex"      keel:"week_index"`
	WeekSumCents   float64  `json:"weekSumCents"   keel:"week_sum_cents"`
	WeekSamples    int32    `json:"weekSamples"    keel:"week_samples"`
	Observations   int64    `json:"observations"   keel:"observations"`
	ObservedAt     int64    `json:"observedAt"     keel:"observed_at"`
	Version        int64    `json:"version"        keel:"version,version"`
	CreatedAt      int64    `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64    `json:"updatedAt"      keel:"updated_at,autoupdate"`
}

func (p Point) Valid() bool {
	return !math.IsNaN(p.Latitude) && !math.IsNaN(p.Longitude) &&
		p.Latitude >= -90 && p.Latitude <= 90 &&
		p.Longitude >= -180 && p.Longitude <= 180 &&
		(p.Latitude != 0 || p.Longitude != 0)
}

func CellOf(p Point) (int64, bool) {
	if !p.Valid() {
		return 0, false
	}

	//nolint:gosec // G115: a zoom-8 quadkey is below 4^8, far inside int64.
	return int64(maptile.At(orb.Point{p.Longitude, p.Latitude}, CellZoom).Quadkey()), true
}

func SlotOf(at int64) int16 {
	return int16(max(at-mondayOffset, 0) % WeekSeconds / SlotSeconds)
}

func WeekOf(at int64) int64 {
	return max(at-mondayOffset, 0) / WeekSeconds
}

func KeyOf(p Point, at int64) (Key, bool) {
	cell, ok := CellOf(p)
	if !ok {
		return Key{}, false
	}

	return Key{Cell: cell, Slot: SlotOf(at)}, true
}

func (v *MarketValue) Key() Key {
	return Key{Cell: v.Cell, Slot: v.Slot}
}

func Alpha(weeks int32) float64 {
	return math.Max(minAlpha, 1/float64(max(weeks, 0)+1))
}

func (v *MarketValue) Estimate() float64 {
	if v.WeekSamples <= 0 {
		return v.ValueCents
	}
	weekMean := v.WeekSumCents / float64(v.WeekSamples)
	if v.Weeks <= 0 {
		return weekMean
	}

	return v.ValueCents + Alpha(v.Weeks)*(weekMean-v.ValueCents)
}

func (v *MarketValue) Fold(sample *Sample) bool {
	if sample.Drivers <= 0 || sample.ObservedAt <= v.ObservedAt {
		return false
	}

	week := WeekOf(sample.ObservedAt)
	switch {
	case v.WeekSamples > 0 && week > v.WeekIndex:
		v.ValueCents = v.Estimate()
		v.Weeks++
		v.WeekIndex = week
		v.WeekSumCents = sample.SumCents
		v.WeekSamples = sample.Drivers
	case week < v.WeekIndex:
		return false
	default:
		v.WeekIndex = week
		v.WeekSumCents += sample.SumCents
		v.WeekSamples += sample.Drivers
	}
	v.Observations++
	v.ObservedAt = sample.ObservedAt

	return true
}
