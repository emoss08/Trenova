package demandforecast

import (
	"cmp"
	"math"
	"slices"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	WindowSlots       = 3
	FutureDiscount    = 0.8
	slotMidSeconds    = dispatchmarket.SlotSeconds / 2
	maxReasonCustomer = 2
	minCustomerMean   = 0.5
)

type Booked struct {
	CustomerID pulid.ID
	Cell       int64
	SlotStart  int64
	Total      int
	Open       int
}

type Driver struct {
	WorkerID pulid.ID
	Cell     int64
	FreeAt   int64
}

type ForecasterParams struct {
	Now           int64
	Index         *Index
	Booked        []Booked
	Solving       []Booked
	Supply        []Driver
	MarginPercent int64
	CapCents      int64
}

type rateKey struct {
	cell int64
	slot int16
}

type absoluteKey struct {
	cell  int64
	start int64
}

type customerKey struct {
	absoluteKey
	customer pulid.ID
}

type Forecaster struct {
	*Index

	now      int64
	booked   map[absoluteKey]Booked
	tendered map[customerKey]int
	solving  map[absoluteKey]int
	supply   map[int64][]Driver
	margin   int64
	cap      int64
}

func NewForecaster(p *ForecasterParams) *Forecaster {
	index := p.Index
	if index == nil {
		index = NewIndex(&IndexParams{})
	}
	f := &Forecaster{
		Index:    index,
		now:      p.Now,
		booked:   make(map[absoluteKey]Booked, len(p.Booked)),
		tendered: map[customerKey]int{},
		solving:  make(map[absoluteKey]int, len(p.Solving)),
		margin:   p.MarginPercent,
		cap:      p.CapCents,
	}
	if f.cap <= 0 {
		f.cap = index.automaticCap(p.MarginPercent)
	}
	for _, booked := range p.Booked {
		key := absoluteKey{cell: booked.Cell, start: f.grid.SlotStart(booked.SlotStart)}
		current := f.booked[key]
		current.Total += booked.Total
		current.Open += booked.Open
		f.booked[key] = current
		if !booked.CustomerID.IsNil() {
			f.tendered[customerKey{absoluteKey: key, customer: booked.CustomerID}] += booked.Total
		}
	}
	for _, solving := range p.Solving {
		key := absoluteKey{cell: solving.Cell, start: f.grid.SlotStart(solving.SlotStart)}
		f.solving[key] += solving.Open
	}
	f.supply = supplyByCell(p.Supply)

	return f
}

func (f *Forecaster) Ready() bool {
	return f != nil && f.Index.Ready()
}

func (f *Forecaster) OutlookStarts() []int64 {
	return OutlookStarts(f.grid, f.now)
}

func (f *Forecaster) Covers(cell int64) bool {
	if !f.Ready() {
		return false
	}
	area, ok := f.areas[cell]

	return ok && area.MedianRevenueCents > 0
}

type Query struct {
	Point    dispatchmarket.Point
	FreeAt   int64
	WorkerID pulid.ID
}

type WindowKey struct {
	Cell  int64
	Start int64
}

type CustomerShare struct {
	CustomerID pulid.ID
	Name       string
	Expected   float64
}

type Reason struct {
	AreaLabel        string
	WindowStart      int64
	WindowEnd        int64
	TypicalLow       int
	TypicalHigh      int
	LikelyLow        int
	LikelyHigh       int
	Booked           int
	OpenBooked       int
	ExpectedUnbooked float64
	DriversExpected  int
	Customers        []CustomerShare
	ValueCents       int64
}

type Valuation struct {
	Key    WindowKey
	Cents  int64
	Reason *Reason
}

type windowForecast struct {
	key      WindowKey
	end      int64
	typical  Count
	unbooked Count
	booked   int
	open     int
	shares   map[pulid.ID]float64
}

func (f *Forecaster) Window(cell, freeAt int64) (WindowKey, bool) {
	if !f.Covers(cell) {
		return WindowKey{}, false
	}

	return WindowKey{Cell: cell, Start: f.grid.SlotStart(max(freeAt, f.now))}, true
}

func (f *Forecaster) Value(q Query) (Valuation, bool) {
	cell, ok := dispatchmarket.CellOf(q.Point)
	if !ok {
		return Valuation{}, false
	}
	key, ok := f.Window(cell, q.FreeAt)
	if !ok {
		return Valuation{}, false
	}

	return f.ValueAt(key, q.WorkerID), true
}

func (f *Forecaster) WithSupply(supply []Driver) *Forecaster {
	next := *f
	next.supply = supplyByCell(supply)

	return &next
}

func supplyByCell(supply []Driver) map[int64][]Driver {
	byCell := map[int64][]Driver{}
	for _, driver := range supply {
		byCell[driver.Cell] = append(byCell[driver.Cell], driver)
	}

	return byCell
}

func (f *Forecaster) ValueAt(key WindowKey, workerID pulid.ID) Valuation {
	forecast := f.forecast(key)
	drivers := f.driversIn(key.Cell, forecast.end, workerID)
	demand := forecast.unbooked.Shift(max(forecast.open, 0))
	probability := demand.Exceeds(drivers)
	area := f.areas[key.Cell]
	value := float64(area.MedianRevenueCents) * float64(f.margin) / 100
	cents := min(int64(math.Round(FutureDiscount*value*probability)), f.cap)

	return Valuation{
		Key:   key,
		Cents: max(cents, 0),
		Reason: &Reason{
			AreaLabel:        area.Label,
			WindowStart:      key.Start,
			WindowEnd:        forecast.end,
			TypicalLow:       forecast.typical.Quantile(IntervalLow),
			TypicalHigh:      forecast.typical.Quantile(IntervalHigh),
			LikelyLow:        forecast.unbooked.Shift(forecast.booked).Quantile(IntervalLow),
			LikelyHigh:       forecast.unbooked.Shift(forecast.booked).Quantile(IntervalHigh),
			Booked:           forecast.booked,
			OpenBooked:       max(forecast.open, 0),
			ExpectedUnbooked: forecast.unbooked.Mean,
			DriversExpected:  drivers,
			Customers:        f.topCustomers(forecast.shares),
			ValueCents:       max(cents, 0),
		},
	}
}

func (f *Forecaster) forecast(key WindowKey) windowForecast {
	starts := f.grid.Starts(key.Start, WindowSlots+1)
	out := windowForecast{key: key, end: starts[WindowSlots], shares: map[pulid.ID]float64{}}
	for _, start := range starts[:WindowSlots] {
		slot := f.grid.Slot(start)
		scale := f.scale(start)
		booked := f.booked[absoluteKey{cell: key.Cell, start: start}]
		out.booked += booked.Total
		out.open += booked.Open - f.solving[absoluteKey{cell: key.Cell, start: start}]
		rate := f.rates[rateKey{cell: key.Cell, slot: slot}]
		if rate == nil {
			continue
		}
		customers := f.customers[rateKey{cell: key.Cell, slot: slot}]
		lead := float64(start+slotMidSeconds-f.now) / 3600
		unbooked := 1 - f.bookedShare(customers, lead)

		beta := rate.Beta / scale
		out.typical = out.typical.Plus(GammaPoisson(rate.Alpha, beta))
		out.unbooked = out.unbooked.Plus(Nowcast(rate.Alpha, beta, booked.Total, unbooked))
		for _, customer := range customers {
			out.shares[customer.CustomerID] += scale * customer.Alpha / customer.Beta
		}
	}

	return out
}

func (f *Forecaster) bookedShare(customers []*Rate, leadHours float64) float64 {
	if leadHours <= 0 {
		return 1
	}
	weighted, weight := 0.0, 0.0
	for _, customer := range customers {
		shares, ok := f.curves[customer.CustomerID]
		if !ok {
			shares = f.tenant
		}
		mean := customer.Alpha / customer.Beta
		weighted += mean * BookedShare(shares, leadHours)
		weight += mean
	}
	if weight == 0 {
		return BookedShare(f.tenant, leadHours)
	}

	return weighted / weight
}

func (f *Forecaster) driversIn(cell, end int64, except pulid.ID) int {
	count := 0
	for _, driver := range f.supply[cell] {
		if driver.WorkerID == except {
			continue
		}
		if driver.FreeAt < end {
			count++
		}
	}

	return count
}

func (f *Forecaster) topCustomers(shares map[pulid.ID]float64) []CustomerShare {
	out := make([]CustomerShare, 0, len(shares))
	for customer, expected := range shares {
		if expected < minCustomerMean {
			continue
		}
		out = append(out, CustomerShare{
			CustomerID: customer,
			Name:       f.names[customer],
			Expected:   expected,
		})
	}
	slices.SortFunc(out, func(a, b CustomerShare) int {
		if c := cmp.Compare(b.Expected, a.Expected); c != 0 {
			return c
		}
		return cmp.Compare(a.CustomerID, b.CustomerID)
	})

	return out[:min(len(out), maxReasonCustomer)]
}

func (r *Reason) Shortfall() float64 {
	if r == nil {
		return 0
	}

	return math.Max(0, float64(r.OpenBooked)+r.ExpectedUnbooked-float64(r.DriversExpected))
}

func (i *Index) automaticCap(marginPercent int64) int64 {
	areas := make([]*Area, 0, len(i.areas))
	for _, cell := range i.cells {
		areas = append(areas, i.areas[cell])
	}

	return AutomaticCapCents(areas, marginPercent)
}

func AutomaticCapCents(areas []*Area, marginPercent int64) int64 {
	priced := make([]*Area, 0, len(areas))
	total := int64(0)
	for _, area := range areas {
		if area.MedianRevenueCents > 0 {
			priced = append(priced, area)
			total += areaWeight(area)
		}
	}
	if total == 0 {
		return 0
	}
	slices.SortFunc(priced, func(a, b *Area) int {
		return cmp.Compare(a.MedianRevenueCents, b.MedianRevenueCents)
	})
	seen := int64(0)
	for _, area := range priced {
		seen += areaWeight(area)
		if 2*seen >= total {
			return area.MedianRevenueCents * marginPercent / 100
		}
	}

	return priced[len(priced)-1].MedianRevenueCents * marginPercent / 100
}

func (f *Forecaster) CapCents() int64 {
	if f == nil {
		return 0
	}

	return f.cap
}

func areaWeight(area *Area) int64 {
	return max(int64(area.Loads), 1)
}
