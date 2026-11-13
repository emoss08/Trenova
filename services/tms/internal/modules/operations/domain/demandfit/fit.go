package demandfit

import (
	"cmp"
	"math"
	"slices"

	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"

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
	MaxAreaCustomers  = 50
)

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
	Now      int64
	Grid     dispatchmarket.Grid
	Holidays demandforecast.Holidays
	Counts   []LoadCount
	Leads    []LeadCount
	Areas    []AreaSummary
}

type window struct {
	first    int64
	last     int64
	weights  map[int64]float64
	total    float64
	exposure [slotsPerWeek]float64
}

func windowOf(counts []LoadCount, beforeWeek int64, holidays demandforecast.Holidays) window {
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
		for slot := range slotsPerWeek {
			if !holidays.On(week, int16(slot)) {
				w.exposure[slot] += weight
			}
		}
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
	w          window
	areas      map[int64]*series
	customers  map[seriesKey]*series
	tenantRate [slotsPerWeek]float64
	tenantSum  float64
}

func aggregateOf(counts []LoadCount, w window, holidays demandforecast.Holidays) *aggregate {
	agg := &aggregate{w: w, areas: map[int64]*series{}, customers: map[seriesKey]*series{}}
	var tenant [slotsPerWeek]float64
	for _, count := range counts {
		weight, ok := w.weights[count.Week]
		if !ok || count.Slot < 0 || int(count.Slot) >= slotsPerWeek || count.Loads <= 0 ||
			holidays.On(count.Week, count.Slot) {
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
		tenant[count.Slot] += weighted
	}
	for slot := range slotsPerWeek {
		agg.tenantRate[slot] = agg.rate(tenant[slot], slot)
		agg.tenantSum += agg.tenantRate[slot]
	}

	return agg
}

func (a *aggregate) rate(weighted float64, slot int) float64 {
	if a.w.exposure[slot] <= 0 {
		return 0
	}

	return weighted / a.w.exposure[slot]
}

func (a *aggregate) weekly(s *series) float64 {
	weekly := 0.0
	for slot := range slotsPerWeek {
		weekly += a.rate(s.slot[slot], slot)
	}

	return weekly
}

func (a *aggregate) tenantShare(slot int) float64 {
	if a.w.total <= 0 {
		return 1 / float64(slotsPerWeek)
	}
	smoothing := smoothingLoadSlot / a.w.total

	return (a.tenantRate[slot] + smoothing/float64(slotsPerWeek)) / (a.tenantSum + smoothing)
}

func (a *aggregate) areaShare(area *series, slot int) float64 {
	return (area.slot[slot] + SlotPriorLoads*a.tenantShare(slot)) /
		(area.total + SlotPriorLoads)
}

func (a *aggregate) posterior(priorMean float64, s *series, slot int) (alpha, beta float64) {
	return PriorWeeks*priorMean + s.slot[slot], PriorWeeks + a.w.exposure[slot]
}

func (a *aggregate) areaRate(area *series, slot int) (alpha, beta float64) {
	return a.posterior(a.weekly(area)*a.tenantShare(slot), area, slot)
}

func Fit(h *History) *demandforecast.Fitted {
	return Train(h).Fitted
}

func (m *trainedWeek) fitted(h *History) *demandforecast.Fitted {
	fitted := &demandforecast.Fitted{Curves: fitCurves(h.Leads, h.Now)}
	if m == nil {
		return fitted
	}
	agg := m.agg
	recent := recentCounts(h.Counts, m.week)

	for _, cell := range m.cells {
		area := agg.areas[cell]
		for slot := range slotsPerWeek {
			alpha, beta := agg.areaRate(area, slot)
			fitted.Rates = append(fitted.Rates, &demandforecast.Rate{
				Cell:     cell,
				Slot:     int16(slot),
				Alpha:    alpha,
				Beta:     beta / m.level,
				Loads:    area.raw[slot],
				Recent:   recent.of(cell, int16(slot)),
				FittedAt: h.Now,
			})
		}
	}

	for _, key := range leadingSeries(agg.customers) {
		customer := agg.customers[key]
		weekly := agg.weekly(customer)
		area := agg.areas[key.cell]
		for slot := range slotsPerWeek {
			priorMean := weekly * agg.areaShare(area, slot)
			if customer.slot[slot] == 0 && priorMean < minCustomerPrior {
				continue
			}
			alpha, beta := agg.posterior(priorMean, customer, slot)
			fitted.Rates = append(fitted.Rates, &demandforecast.Rate{
				CustomerID: key.customer,
				Cell:       key.cell,
				Slot:       int16(slot),
				Alpha:      alpha,
				Beta:       beta / m.level,
				Loads:      customer.raw[slot],
				FittedAt:   h.Now,
			})
		}
	}

	for _, summary := range h.Areas {
		if _, seen := agg.areas[summary.Cell]; !seen {
			continue
		}
		fitted.Areas = append(fitted.Areas, &demandforecast.Area{
			Cell:               summary.Cell,
			Label:              summary.Label,
			MedianRevenueCents: summary.MedianRevenueCents,
			Loads:              summary.Loads,
			FittedAt:           h.Now,
		})
	}

	return fitted
}

type recentWeeks struct {
	first  int64
	counts map[slotKey][]int32
}

func recentCounts(counts []LoadCount, beforeWeek int64) recentWeeks {
	r := recentWeeks{first: beforeWeek - demandforecast.RecordWeeks, counts: map[slotKey][]int32{}}
	for _, count := range counts {
		if count.Week < r.first || count.Week >= beforeWeek {
			continue
		}
		key := slotKey{cell: count.Cell, slot: count.Slot}
		weeks := r.counts[key]
		if weeks == nil {
			weeks = make([]int32, demandforecast.RecordWeeks)
			r.counts[key] = weeks
		}
		weeks[count.Week-r.first] += count.Loads
	}

	return r
}

func (r recentWeeks) of(cell int64, slot int16) []int32 {
	if weeks, ok := r.counts[slotKey{cell: cell, slot: slot}]; ok {
		return weeks
	}

	return make([]int32, demandforecast.RecordWeeks)
}

func sortedCells(areas map[int64]*series) []int64 {
	cells := make([]int64, 0, len(areas))
	for cell := range areas {
		cells = append(cells, cell)
	}
	slices.Sort(cells)

	return cells
}

func leadingSeries(customers map[seriesKey]*series) []seriesKey {
	byCell := map[int64][]seriesKey{}
	for key := range customers {
		byCell[key.cell] = append(byCell[key.cell], key)
	}
	keys := make([]seriesKey, 0, len(customers))
	for _, cell := range byCell {
		slices.SortFunc(cell, func(a, b seriesKey) int {
			if c := cmp.Compare(customers[b].total, customers[a].total); c != 0 {
				return c
			}
			return cmp.Compare(a.customer, b.customer)
		})
		keys = append(keys, cell[:min(len(cell), MaxAreaCustomers)]...)
	}
	slices.SortFunc(keys, func(a, b seriesKey) int {
		if c := cmp.Compare(a.customer, b.customer); c != 0 {
			return c
		}
		return cmp.Compare(a.cell, b.cell)
	})

	return keys
}

func fitCurves(leads []LeadCount, now int64) []*demandforecast.BookingCurve {
	tenant := make([]float64, len(demandforecast.LeadHours))
	byCustomer := map[pulid.ID][]float64{}
	tenantLoads := 0.0
	for _, lead := range leads {
		if lead.Loads <= 0 {
			continue
		}
		bucket := min(max(lead.Bucket, 0), len(demandforecast.LeadHours)-1)
		tenant[bucket] += float64(lead.Loads)
		tenantLoads += float64(lead.Loads)
		if lead.CustomerID.IsNil() {
			continue
		}
		buckets := byCustomer[lead.CustomerID]
		if buckets == nil {
			buckets = make([]float64, len(demandforecast.LeadHours))
			byCustomer[lead.CustomerID] = buckets
		}
		buckets[bucket] += float64(lead.Loads)
	}
	if tenantLoads == 0 {
		return nil
	}

	tenantShares := sharesOf(tenant, nil)
	curves := []*demandforecast.BookingCurve{{
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
		curves = append(curves, &demandforecast.BookingCurve{
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
