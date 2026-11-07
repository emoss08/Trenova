package volumegen

import (
	"cmp"
	"math"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
)

const (
	laneSlots        = int(dispatchmarket.SlotsPerWeek)
	laneSlotsPerDay  = 6
	laneCandidates   = 25
	laneDestinations = 4
	laneMinMiles     = 150
	laneMaxMiles     = 700
	laneMonthEndDays = 4
	lanePickupStep   = 30 * Minute
)

type LeadBand struct {
	Weight int
	Min    int64
	Max    int64
}

type LaneProgram struct {
	Name          string
	Customer      *CustomerRecord
	Origin        *LocationRecord
	Destinations  []*LocationRecord
	WeeklyLoads   float64
	SlotWeights   [laneSlots]float64
	MonthEndBoost float64
	Leads         []LeadBand
	share         float64
}

type LaneLoad struct {
	Program   *LaneProgram
	Dest      *LocationRecord
	PickupAt  int64
	CreatedAt int64
}

type LaneProgramParams struct {
	Customers  []*CustomerRecord
	Shippers   []*LocationRecord
	Consignees []*LocationRecord
	Begin      int64
	End        int64
	Now        int64
	Loads      int
}

type LanePlan struct {
	Programs []*LaneProgram
	Loads    []LaneLoad
}

type lanePeak struct {
	days   []time.Weekday
	slots  []int
	weight float64
}

type laneTemplate struct {
	name     string
	share    float64
	base     float64
	peaks    []lanePeak
	monthEnd float64
	leads    []LeadBand
}

var laneTemplates = []laneTemplate{
	{
		name:  "Monday and Friday DC",
		share: 3,
		base:  0.05,
		peaks: []lanePeak{
			{days: []time.Weekday{time.Monday, time.Friday}, slots: []int{1, 2, 3}, weight: 4},
		},
		leads: []LeadBand{
			{Weight: 50, Min: 24 * Hour, Max: 96 * Hour},
			{Weight: 40, Min: 8 * Hour, Max: 24 * Hour},
			{Weight: 10, Min: 2 * Hour, Max: 8 * Hour},
		},
	},
	{
		name:  "Tuesday morning",
		share: 2,
		base:  0.05,
		peaks: []lanePeak{
			{days: []time.Weekday{time.Tuesday}, slots: []int{1, 2}, weight: 6},
		},
		leads: []LeadBand{
			{Weight: 60, Min: 18 * Hour, Max: 72 * Hour},
			{Weight: 30, Min: 2 * Hour, Max: 18 * Hour},
			{Weight: 10, Min: 30 * Minute, Max: 2 * Hour},
		},
	},
	{
		name:  "Weekday steady",
		share: 3,
		base:  0.1,
		peaks: []lanePeak{{
			days: []time.Weekday{
				time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday,
			},
			slots:  []int{2, 3},
			weight: 2,
		}},
		leads: []LeadBand{
			{Weight: 40, Min: 24 * Hour, Max: 72 * Hour},
			{Weight: 40, Min: 4 * Hour, Max: 24 * Hour},
			{Weight: 20, Min: Hour, Max: 4 * Hour},
		},
	},
	{
		name:  "Month-end push",
		share: 2,
		base:  0.05,
		peaks: []lanePeak{{
			days:   []time.Weekday{time.Wednesday, time.Thursday},
			slots:  []int{3, 4},
			weight: 3,
		}},
		monthEnd: 3,
		leads: []LeadBand{
			{Weight: 30, Min: 72 * Hour, Max: 168 * Hour},
			{Weight: 50, Min: 24 * Hour, Max: 72 * Hour},
			{Weight: 20, Min: 4 * Hour, Max: 24 * Hour},
		},
	},
	{
		name:  "Overnight produce",
		share: 2,
		base:  0.05,
		peaks: []lanePeak{{
			days:   []time.Weekday{time.Sunday, time.Monday, time.Tuesday},
			slots:  []int{0, 5},
			weight: 3,
		}},
		leads: []LeadBand{
			{Weight: 20, Min: 12 * Hour, Max: 24 * Hour},
			{Weight: 50, Min: 4 * Hour, Max: 12 * Hour},
			{Weight: 30, Min: Hour, Max: 4 * Hour},
		},
	},
}

func PlanLanePrograms(f *Faker, p *LaneProgramParams) *LanePlan {
	plan := &LanePlan{}
	if p.Loads <= 0 || p.End <= p.Begin || len(p.Shippers) == 0 || len(p.Consignees) == 0 {
		return plan
	}

	plan.Programs = newLanePrograms(f, p)
	shares := 0.0
	for _, program := range plan.Programs {
		shares += program.share
	}
	weeks := float64(p.End-p.Begin) / float64(dispatchmarket.WeekSeconds)
	for _, program := range plan.Programs {
		target := float64(p.Loads) * program.share / shares
		program.WeeklyLoads = target / weeks
		plan.Loads = append(plan.Loads, program.schedule(f, p, target)...)
	}
	slices.SortStableFunc(plan.Loads, func(a, b LaneLoad) int {
		return cmp.Compare(a.PickupAt, b.PickupAt)
	})

	return plan
}

func newLanePrograms(f *Faker, p *LaneProgramParams) []*LaneProgram {
	candidates := laneCandidatesOf(f, p.Customers)
	programs := make([]*LaneProgram, 0, len(laneTemplates))
	usedCells := make(map[int64]bool, len(laneTemplates))
	next := 0
	for i := range laneTemplates {
		tmpl := &laneTemplates[i]
		for next < len(candidates) {
			customer := candidates[next]
			next++
			origin := laneOrigin(customer, p.Shippers)
			if origin == nil {
				continue
			}
			cell, ok := dispatchmarket.CellOf(dispatchmarket.Point{
				Latitude:  origin.City.Lat,
				Longitude: origin.City.Lon,
			})
			if !ok || usedCells[cell] {
				continue
			}
			destinations := laneDestinationsOf(f, origin, p.Consignees)
			if len(destinations) == 0 {
				continue
			}
			usedCells[cell] = true
			programs = append(programs, &LaneProgram{
				Name:          tmpl.name,
				Customer:      customer,
				Origin:        origin,
				Destinations:  destinations,
				SlotWeights:   tmpl.slotWeights(),
				MonthEndBoost: tmpl.monthEnd,
				Leads:         tmpl.leads,
				share:         tmpl.share,
			})
			break
		}
	}

	return programs
}

func laneCandidatesOf(f *Faker, customers []*CustomerRecord) []*CustomerRecord {
	pool := make([]*CustomerRecord, 0, len(customers))
	for _, customer := range customers {
		if customer.Weight > 1 && customer.Home != nil {
			pool = append(pool, customer)
		}
	}
	slices.SortStableFunc(pool, func(a, b *CustomerRecord) int {
		return cmp.Compare(b.Weight, a.Weight)
	})
	pool = pool[:min(len(pool), laneCandidates)]
	f.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })

	return pool
}

func laneOrigin(customer *CustomerRecord, shippers []*LocationRecord) *LocationRecord {
	for _, facility := range customer.Facilities {
		if facility.Shipper {
			return facility
		}
	}
	var best *LocationRecord
	bestMiles := math.MaxFloat64
	for _, location := range shippers {
		if miles := HaversineMiles(customer.Home, location.City); miles < bestMiles {
			best, bestMiles = location, miles
		}
	}

	return best
}

func laneDestinationsOf(
	f *Faker,
	origin *LocationRecord,
	consignees []*LocationRecord,
) []*LocationRecord {
	pool := make([]*LocationRecord, 0, len(consignees))
	for _, location := range consignees {
		miles := HaversineMiles(origin.City, location.City)
		if location != origin && miles >= laneMinMiles && miles <= laneMaxMiles {
			pool = append(pool, location)
		}
	}
	f.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })

	return pool[:min(len(pool), laneDestinations)]
}

func (t *laneTemplate) slotWeights() [laneSlots]float64 {
	var weights [laneSlots]float64
	for i := range weights {
		weights[i] = t.base
	}
	for _, peak := range t.peaks {
		for _, day := range peak.days {
			for _, slot := range peak.slots {
				weights[laneSlot(day, slot)] += peak.weight
			}
		}
	}
	total := 0.0
	for _, weight := range weights {
		total += weight
	}
	for i := range weights {
		weights[i] /= total
	}

	return weights
}

func laneSlot(day time.Weekday, slotOfDay int) int {
	return (int(day)+6)%7*laneSlotsPerDay + slotOfDay
}

func (lp *LaneProgram) schedule(f *Faker, p *LaneProgramParams, target float64) []LaneLoad {
	first := dispatchmarket.SlotStart(p.Begin)
	if first < p.Begin {
		first += dispatchmarket.SlotSeconds
	}
	raw := 0.0
	for at := first; at < p.End; at += dispatchmarket.SlotSeconds {
		raw += lp.slotMean(at)
	}
	if raw == 0 {
		return nil
	}

	scale := target / raw
	steps := dispatchmarket.SlotSeconds / lanePickupStep
	loads := make([]LaneLoad, 0, int(target))
	for at := first; at < p.End; at += dispatchmarket.SlotSeconds {
		for range f.Poisson(scale * lp.slotMean(at)) {
			pickup := at + f.Between64(0, steps-1)*lanePickupStep
			created := pickup - lp.lead(f)
			dest := Pick(f, lp.Destinations)
			if created >= p.Now {
				continue
			}
			loads = append(loads, LaneLoad{
				Program:   lp,
				Dest:      dest,
				PickupAt:  pickup,
				CreatedAt: created,
			})
		}
	}

	return loads
}

func (lp *LaneProgram) slotMean(at int64) float64 {
	mean := lp.SlotWeights[dispatchmarket.SlotOf(at)]
	if lp.MonthEndBoost > 0 && monthEnd(at) {
		mean *= lp.MonthEndBoost
	}

	return mean
}

func monthEnd(at int64) bool {
	day := time.Unix(at, 0).UTC()

	return day.AddDate(0, 0, laneMonthEndDays).Month() != day.Month()
}

func (lp *LaneProgram) lead(f *Faker) int64 {
	weights := make([]int, len(lp.Leads))
	for i, band := range lp.Leads {
		weights[i] = band.Weight
	}
	band := lp.Leads[f.Weighted(weights)]

	return f.Between64(band.Min, band.Max)
}
