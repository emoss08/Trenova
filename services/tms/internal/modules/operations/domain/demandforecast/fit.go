package demandforecast

import (
	"cmp"
	"math"
	"slices"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	HistoryWeeks      = 26
	HalfLifeWeeks     = 8.0
	PriorWeeks        = 2.0
	SlotPriorLoads    = 5.0
	CurvePriorLoads   = 20.0
	minCustomerPrior  = 0.05
	slotsPerWeek      = int(dispatchmarket.SlotsPerWeek)
	smoothingLoadSlot = 1.0
)

var LeadHours = [...]int64{0, 2, 4, 8, 12, 24, 36, 48, 72, 96, 120, 168, 336}

type LoadCount struct {
	CustomerID pulid.ID
	Cell       int64
	Week       int64
	Slot       int16
	Loads      int32
}

type LeadCount struct {
	CustomerID pulid.ID
	Bucket     int
	Loads      int32
}

type AreaSummary struct {
	Cell               int64
	Label              string
	MedianRevenueCents int64
	Loads              int32
}

type History struct {
	Now    int64
	Counts []LoadCount
	Leads  []LeadCount
	Areas  []AreaSummary
}

type Fitted struct {
	Rates  []*Rate
	Areas  []*Area
	Curves []*BookingCurve
}

type window struct {
	first   int64
	last    int64
	weights map[int64]float64
	total   float64
}

func windowOf(counts []LoadCount, beforeWeek int64) window {
	w := window{first: beforeWeek, last: beforeWeek - 1, weights: map[int64]float64{}}
	earliest := beforeWeek
	for _, count := range counts {
		if count.Week < beforeWeek && count.Week < earliest {
			earliest = count.Week
		}
	}
	w.first = max(earliest, beforeWeek-HistoryWeeks)
	for week := w.first; week <= w.last; week++ {
		weight := math.Pow(0.5, float64(beforeWeek-1-week)/HalfLifeWeeks)
		w.weights[week] = weight
		w.total += weight
	}

	return w
}

func (w window) weeks() int {
	return int(max(w.last-w.first+1, 0))
}

type seriesKey struct {
	customer pulid.ID
	cell     int64
}

type series struct {
	slot  [slotsPerWeek]float64
	raw   [slotsPerWeek]int32
	total float64
}

func (s *series) add(slot int16, weighted float64, loads int32) {
	s.slot[slot] += weighted
	s.raw[slot] += loads
	s.total += weighted
}

type aggregate struct {
	areas     map[int64]*series
	customers map[seriesKey]*series
	tenant    [slotsPerWeek]float64
	tenantSum float64
}

func aggregateOf(counts []LoadCount, w window) *aggregate {
	agg := &aggregate{areas: map[int64]*series{}, customers: map[seriesKey]*series{}}
	for _, count := range counts {
		weight, ok := w.weights[count.Week]
		if !ok || count.Slot < 0 || int(count.Slot) >= slotsPerWeek || count.Loads <= 0 {
			continue
		}
		weighted := weight * float64(count.Loads)
		area := agg.areas[count.Cell]
		if area == nil {
			area = &series{}
			agg.areas[count.Cell] = area
		}
		area.add(count.Slot, weighted, count.Loads)
		if !count.CustomerID.IsNil() {
			key := seriesKey{customer: count.CustomerID, cell: count.Cell}
			customer := agg.customers[key]
			if customer == nil {
				customer = &series{}
				agg.customers[key] = customer
			}
			customer.add(count.Slot, weighted, count.Loads)
		}
		agg.tenant[count.Slot] += weighted
		agg.tenantSum += weighted
	}

	return agg
}

func (a *aggregate) tenantShare(slot int) float64 {
	return (a.tenant[slot] + smoothingLoadSlot/float64(slotsPerWeek)) /
		(a.tenantSum + smoothingLoadSlot)
}

func (a *aggregate) areaShare(area *series, slot int) float64 {
	return (area.slot[slot] + SlotPriorLoads*a.tenantShare(slot)) /
		(area.total + SlotPriorLoads)
}

func posterior(priorMean, weightedLoads, exposure float64) (alpha, beta float64) {
	return PriorWeeks*priorMean + weightedLoads, PriorWeeks + exposure
}

func Fit(h *History) *Fitted {
	beforeWeek := dispatchmarket.WeekOf(h.Now)
	w := windowOf(h.Counts, beforeWeek)
	fitted := &Fitted{Curves: fitCurves(h.Leads, h.Now)}
	if w.weeks() == 0 {
		return fitted
	}
	agg := aggregateOf(h.Counts, w)

	for _, cell := range sortedCells(agg.areas) {
		area := agg.areas[cell]
		weekly := area.total / w.total
		for slot := range slotsPerWeek {
			alpha, beta := posterior(weekly*agg.tenantShare(slot), area.slot[slot], w.total)
			fitted.Rates = append(fitted.Rates, &Rate{
				Cell:     cell,
				Slot:     int16(slot),
				Alpha:    alpha,
				Beta:     beta,
				Loads:    area.raw[slot],
				FittedAt: h.Now,
			})
		}
	}

	for _, key := range sortedSeries(agg.customers) {
		customer := agg.customers[key]
		weekly := customer.total / w.total
		area := agg.areas[key.cell]
		for slot := range slotsPerWeek {
			priorMean := weekly * agg.areaShare(area, slot)
			if customer.slot[slot] == 0 && priorMean < minCustomerPrior {
				continue
			}
			alpha, beta := posterior(priorMean, customer.slot[slot], w.total)
			fitted.Rates = append(fitted.Rates, &Rate{
				CustomerID: key.customer,
				Cell:       key.cell,
				Slot:       int16(slot),
				Alpha:      alpha,
				Beta:       beta,
				Loads:      customer.raw[slot],
				FittedAt:   h.Now,
			})
		}
	}

	for _, summary := range h.Areas {
		if _, seen := agg.areas[summary.Cell]; !seen {
			continue
		}
		fitted.Areas = append(fitted.Areas, &Area{
			Cell:               summary.Cell,
			Label:              summary.Label,
			MedianRevenueCents: summary.MedianRevenueCents,
			Loads:              summary.Loads,
			FittedAt:           h.Now,
		})
	}
	return fitted
}

func sortedCells(areas map[int64]*series) []int64 {
	cells := make([]int64, 0, len(areas))
	for cell := range areas {
		cells = append(cells, cell)
	}
	slices.Sort(cells)

	return cells
}

func sortedSeries(customers map[seriesKey]*series) []seriesKey {
	keys := make([]seriesKey, 0, len(customers))
	for key := range customers {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, func(a, b seriesKey) int {
		if c := cmp.Compare(a.customer, b.customer); c != 0 {
			return c
		}
		return cmp.Compare(a.cell, b.cell)
	})

	return keys
}

func fitCurves(leads []LeadCount, now int64) []*BookingCurve {
	tenant := make([]float64, len(LeadHours))
	byCustomer := map[pulid.ID][]float64{}
	tenantLoads := 0.0
	for _, lead := range leads {
		if lead.Loads <= 0 {
			continue
		}
		bucket := min(max(lead.Bucket, 0), len(LeadHours)-1)
		tenant[bucket] += float64(lead.Loads)
		tenantLoads += float64(lead.Loads)
		if lead.CustomerID.IsNil() {
			continue
		}
		buckets := byCustomer[lead.CustomerID]
		if buckets == nil {
			buckets = make([]float64, len(LeadHours))
			byCustomer[lead.CustomerID] = buckets
		}
		buckets[bucket] += float64(lead.Loads)
	}
	if tenantLoads == 0 {
		return nil
	}

	tenantShares := sharesOf(tenant, nil)
	curves := []*BookingCurve{{
		Shares:   tenantShares,
		Loads:    int32(tenantLoads),
		FittedAt: now,
	}}
	customers := make([]pulid.ID, 0, len(byCustomer))
	for customer := range byCustomer {
		customers = append(customers, customer)
	}
	slices.Sort(customers)
	for _, customer := range customers {
		buckets := byCustomer[customer]
		loads := 0.0
		for _, n := range buckets {
			loads += n
		}
		curves = append(curves, &BookingCurve{
			CustomerID: customer,
			Shares:     sharesOf(buckets, tenantShares),
			Loads:      int32(loads),
			FittedAt:   now,
		})
	}

	return curves
}

func sharesOf(buckets, prior []float64) []float64 {
	shares := make([]float64, len(buckets))
	total := 0.0
	for _, n := range buckets {
		total += n
	}
	atLeast := total
	for i, n := range buckets {
		observed := 0.0
		if total > 0 {
			observed = atLeast / total
		}
		if prior == nil {
			shares[i] = observed
		} else {
			shares[i] = (atLeast + CurvePriorLoads*prior[i]) / (total + CurvePriorLoads)
		}
		atLeast -= n
	}

	return shares
}

func BookedShare(shares []float64, leadHours float64) float64 {
	if len(shares) != len(LeadHours) || leadHours <= 0 {
		if leadHours <= 0 {
			return 1
		}
		return 0
	}
	last := len(LeadHours) - 1
	if leadHours >= float64(LeadHours[last]) {
		return shares[last]
	}
	for i := 1; i <= last; i++ {
		upper := float64(LeadHours[i])
		if leadHours > upper {
			continue
		}
		lower := float64(LeadHours[i-1])
		t := (leadHours - lower) / (upper - lower)
		return shares[i-1] + t*(shares[i]-shares[i-1])
	}

	return shares[last]
}

func LeadBucket(leadSeconds int64) int {
	hours := leadSeconds / 3600
	bucket := 0
	for i, edge := range LeadHours {
		if hours >= edge {
			bucket = i
		}
	}

	return bucket
}
