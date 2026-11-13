package demandforecast

import (
	"math"

	"github.com/emoss08/trenova/shared/pulid"
)

const OutlookSlots = 12

type CustomerSlot struct {
	CustomerID pulid.ID
	Name       string
	Usual      float64
	ByNow      float64
	Tendered   int
}

type SlotForecast struct {
	Start     int64
	End       int64
	Demand    Count
	Open      Count
	Booked    int
	Need      float64
	Customers []CustomerSlot
}

type SlotBalance struct {
	Start     int64
	End       int64
	Demand    Count
	Open      Count
	Low       int
	High      int
	Booked    int
	Need      float64
	Drivers   int
	Short     float64
	Spare     float64
	Customers []CustomerSlot
}

type AreaBalance struct {
	Cell               int64
	Label              string
	MedianRevenueCents int64
	Slots              []SlotBalance
}

type NetworkSlot struct {
	Start     int64
	End       int64
	Low       int
	High      int
	NeedLow   int
	NeedHigh  int
	Need      float64
	Drivers   float64
	Short     float64
	Uncovered float64
	Idle      float64
}

func (f *Forecaster) Slot(cell, start int64) SlotForecast {
	key := absoluteKey{cell: cell, start: start}
	slot := f.grid.Slot(start)
	scale := f.scale(start)
	booked := f.booked[key]
	open := float64(max(booked.Open-f.solving[key], 0))
	customers := f.customers[rateKey{cell: cell, slot: slot}]
	lead := float64(start+slotMidSeconds-f.now) / 3600
	out := SlotForecast{
		Start:  start,
		End:    f.grid.SlotEnd(start),
		Demand: Count{}.Shift(booked.Total),
		Open:   Count{}.Shift(int(open)),
		Booked: booked.Total,
		Need:   open,
	}
	if rate := f.rates[rateKey{cell: cell, slot: slot}]; rate != nil {
		unbooked := Nowcast(
			rate.Alpha,
			rate.Beta/scale,
			booked.Total,
			1-f.bookedShare(customers, lead),
		)
		out.Demand = unbooked.Shift(booked.Total)
		out.Open = unbooked.Shift(int(open))
		out.Need += unbooked.Mean
	}

	out.Customers = make([]CustomerSlot, 0, len(customers))
	for _, customer := range customers {
		usual := scale * customer.Alpha / customer.Beta
		byNow := usual
		if lead > 0 {
			shares, ok := f.curves[customer.CustomerID]
			if !ok {
				shares = f.tenant
			}
			byNow = usual * BookedShare(shares, lead)
		}
		out.Customers = append(out.Customers, CustomerSlot{
			CustomerID: customer.CustomerID,
			Name:       f.names[customer.CustomerID],
			Usual:      usual,
			ByNow:      byNow,
			Tendered:   f.tendered[customerKey{absoluteKey: key, customer: customer.CustomerID}],
		})
	}

	return out
}

func (f *Forecaster) Balance(cell int64, starts []int64) AreaBalance {
	area := f.areas[cell]
	out := AreaBalance{Cell: cell, Slots: make([]SlotBalance, len(starts))}
	if area != nil {
		out.Label, out.MedianRevenueCents = area.Label, area.MedianRevenueCents
	}

	pool := 0.0
	for i, start := range starts {
		forecast := f.Slot(cell, start)
		end := forecast.End
		freed, total := 0, 0
		for _, driver := range f.supply[cell] {
			if driver.FreeAt >= end {
				continue
			}
			total++
			if i == 0 || driver.FreeAt >= start {
				freed++
			}
		}
		pool += float64(freed)
		short := math.Max(0, forecast.Need-pool)
		pool = math.Max(0, pool-forecast.Need)
		out.Slots[i] = SlotBalance{
			Start:     start,
			End:       end,
			Demand:    forecast.Demand,
			Open:      forecast.Open,
			Low:       forecast.Demand.Quantile(IntervalLow),
			High:      forecast.Demand.Quantile(IntervalHigh),
			Booked:    forecast.Booked,
			Need:      forecast.Need,
			Drivers:   total,
			Short:     short,
			Spare:     pool,
			Customers: forecast.Customers,
		}
	}

	return out
}

func (f *Forecaster) Network(areas []AreaBalance, starts []int64) []NetworkSlot {
	out := make([]NetworkSlot, len(starts))
	pool := 0.0
	for i, start := range starts {
		end := f.grid.SlotEnd(start)
		freed := 0
		for _, drivers := range f.supply {
			for _, driver := range drivers {
				if driver.FreeAt < end && (i == 0 || driver.FreeAt >= start) {
					freed++
				}
			}
		}
		pool += float64(freed)

		demand, open := Count{}, Count{}
		slot := NetworkSlot{Start: start, End: end, Drivers: pool}
		for _, area := range areas {
			balance := area.Slots[i]
			demand = demand.Plus(balance.Demand)
			open = open.Plus(balance.Open)
			slot.Need += balance.Need
			slot.Uncovered += balance.Short
			slot.Idle += balance.Spare
		}
		demand, open = demand.Widen(f.dispersion), open.Widen(f.dispersion)
		slot.Low, slot.High = demand.Quantile(IntervalLow), demand.Quantile(IntervalHigh)
		slot.NeedLow, slot.NeedHigh = open.Quantile(IntervalLow), open.Quantile(IntervalHigh)
		slot.Short = math.Max(0, slot.Need-pool)
		pool = math.Max(0, pool-slot.Need)
		out[i] = slot
	}

	return out
}

func (b *AreaBalance) Worst() (index int, short float64) {
	for i, slot := range b.Slots {
		if slot.Short > short {
			index, short = i, slot.Short
		}
	}

	return index, short
}

func (b *AreaBalance) MostSpare() (index int, spare float64) {
	for i, slot := range b.Slots {
		if slot.Spare > spare {
			index, spare = i, slot.Spare
		}
	}

	return index, spare
}

func (f *Forecaster) OneMoreDriver(b *AreaBalance) int64 {
	if len(b.Slots) == 0 || !f.Covers(b.Cell) {
		return 0
	}
	index, _ := b.Worst()

	return f.ValueAt(WindowKey{Cell: b.Cell, Start: b.Slots[index].Start}, pulid.Nil).Cents
}

func (f *Forecaster) MarginPercent() int64 {
	return f.margin
}
