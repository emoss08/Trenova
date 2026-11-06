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
	DefaultCapCents   = int64(40_000)
	slotMidSeconds    = dispatchmarket.SlotSeconds / 2
	maxReasonCustomer = 2
	minCustomerMean   = 0.5
)

type Booked struct {
	Cell      int64
	SlotStart int64
	Total     int
	Open      int
}

type Driver struct {
	WorkerID pulid.ID
	Cell     int64
	FreeAt   int64
}

type ForecasterParams struct {
	Now           int64
	Quality       *Quality
	Rates         []*Rate
	Areas         []*Area
	Curves        []*BookingCurve
	Booked        []Booked
	Solving       []Booked
	Supply        []Driver
	CustomerNames map[pulid.ID]string
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

type Forecaster struct {
	now       int64
	ready     bool
	areas     map[int64]*Area
	rates     map[rateKey]*Rate
	customers map[rateKey][]*Rate
	curves    map[pulid.ID][]float64
	tenant    []float64
	booked    map[absoluteKey]Booked
	solving   map[absoluteKey]int
	supply    map[int64][]Driver
	names     map[pulid.ID]string
	margin    int64
	cap       int64
}

func NewForecaster(p *ForecasterParams) *Forecaster {
	f := &Forecaster{
		now:       p.Now,
		ready:     p.Quality.Ready(),
		areas:     make(map[int64]*Area, len(p.Areas)),
		rates:     make(map[rateKey]*Rate, len(p.Rates)),
		customers: map[rateKey][]*Rate{},
		curves:    make(map[pulid.ID][]float64, len(p.Curves)),
		booked:    make(map[absoluteKey]Booked, len(p.Booked)),
		solving:   make(map[absoluteKey]int, len(p.Solving)),
		names:     p.CustomerNames,
		margin:    p.MarginPercent,
		cap:       cmp.Or(p.CapCents, DefaultCapCents),
	}
	for _, area := range p.Areas {
		f.areas[area.Cell] = area
	}
	for _, rate := range p.Rates {
		key := rateKey{cell: rate.Cell, slot: rate.Slot}
		if rate.CustomerID.IsNil() {
			f.rates[key] = rate
			continue
		}
		f.customers[key] = append(f.customers[key], rate)
	}
	for _, curve := range p.Curves {
		if curve.CustomerID.IsNil() {
			f.tenant = curve.Shares
			continue
		}
		f.curves[curve.CustomerID] = curve.Shares
	}
	for _, booked := range p.Booked {
		key := absoluteKey{cell: booked.Cell, start: dispatchmarket.SlotStart(booked.SlotStart)}
		current := f.booked[key]
		current.Total += booked.Total
		current.Open += booked.Open
		f.booked[key] = current
	}
	for _, solving := range p.Solving {
		f.solving[absoluteKey{cell: solving.Cell, start: dispatchmarket.SlotStart(solving.SlotStart)}] += solving.Open
	}
	f.supply = supplyByCell(p.Supply)

	return f
}

func (f *Forecaster) Ready() bool {
	return f != nil && f.ready
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

	return WindowKey{Cell: cell, Start: dispatchmarket.SlotStart(max(freeAt, f.now))}, true
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
	drivers := f.driversIn(key, workerID)
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
			WindowEnd:        key.Start + WindowSlots*dispatchmarket.SlotSeconds,
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
	out := windowForecast{key: key, shares: map[pulid.ID]float64{}}
	for i := range int64(WindowSlots) {
		start := key.Start + i*dispatchmarket.SlotSeconds
		slot := dispatchmarket.SlotOf(start)
		rate := f.rates[rateKey{cell: key.Cell, slot: slot}]
		if rate == nil {
			continue
		}
		booked := f.booked[absoluteKey{cell: key.Cell, start: start}]
		customers := f.customers[rateKey{cell: key.Cell, slot: slot}]
		lead := float64(start+slotMidSeconds-f.now) / 3600
		unbooked := 1 - f.bookedShare(customers, lead)

		out.typical = out.typical.Plus(GammaPoisson(rate.Alpha, rate.Beta))
		out.unbooked = out.unbooked.Plus(Nowcast(rate.Alpha, rate.Beta, booked.Total, unbooked))
		out.booked += booked.Total
		out.open += booked.Open - f.solving[absoluteKey{cell: key.Cell, start: start}]
		for _, customer := range customers {
			out.shares[customer.CustomerID] += customer.Alpha / customer.Beta
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

func (f *Forecaster) driversIn(key WindowKey, except pulid.ID) int {
	end := key.Start + WindowSlots*dispatchmarket.SlotSeconds
	count := 0
	for _, driver := range f.supply[key.Cell] {
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

func (f *Forecaster) Cells() []int64 {
	cells := make([]int64, 0, len(f.areas))
	for cell := range f.areas {
		cells = append(cells, cell)
	}
	slices.Sort(cells)

	return cells
}
