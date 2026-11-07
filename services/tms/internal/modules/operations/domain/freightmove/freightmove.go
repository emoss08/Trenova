package freightmove

import (
	"cmp"
	"math"
	"slices"

	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/shared/geoutils"
	"github.com/emoss08/trenova/shared/pulid"
)

type Kind string

const (
	KindSend = Kind("Send")
	KindBook = Kind("Book")
	KindKeep = Kind("Keep")
	KindCall = Kind("Call")
)

const (
	MaxRepositionMiles = 600.0
	MaxReachMiles      = 900.0
	LaneRadiusMiles    = 60.0
	MaxSteps           = 12
	MaxLaneChoices     = 5
	MaxReach           = 8
	MaxTargets         = 120
	MinShort           = 0.5
	MinSpare           = 0.5
	KeepMinShort       = 1.0
	CallHorizonSlots   = 6
	CallMinUsual       = 1.0
	CallBehindBy       = 1.0
	CallRecoveryShare  = 0.25
	AheadBy            = 1.5
)

type Driver struct {
	WorkerID pulid.ID
	Name     string
	Point    dispatchmarket.Point
	Cell     int64
	FreeAt   int64
}

type Lane struct {
	MoveID      pulid.ID
	ProNumber   string
	Origin      dispatchmarket.Point
	Destination dispatchmarket.Point
	PickupStart int64
	PickupEnd   int64
	DeliveryEnd int64
}

type Departure struct {
	MoveID           pulid.ID
	ProNumber        string
	WorkerID         pulid.ID
	Origin           dispatchmarket.Point
	PickupStart      int64
	PickupEnd        int64
	DestinationLabel string
}

type Valuer interface {
	ValueAt(key demandforecast.WindowKey, workerID pulid.ID) demandforecast.Valuation
	Covers(cell int64) bool
}

type Inputs struct {
	Now            int64
	Areas          []demandforecast.AreaBalance
	Drivers        []Driver
	Lanes          []Lane
	Departures     []Departure
	EmptyMileCents int64
	MarginPercent  int64
	Valuer         Valuer
}

type Step struct {
	Kind           Kind
	WorkerID       pulid.ID
	WorkerName     string
	FromCell       int64
	FromLabel      string
	ToCell         int64
	ToLabel        string
	SlotStart      int64
	Short          float64
	FreeAt         int64
	ArriveAt       int64
	Miles          float64
	EmptyCostCents int64
	DropCents      int64
	StayCents      int64
	NetCents       int64
	SavedCents     int64
	MoveID         pulid.ID
	ProNumber      string
	LaneMoveIDs    []pulid.ID
	PartnerID      pulid.ID
	PartnerName    string
	PartnerCell    int64
	PartnerLabel   string
	PartnerFreeAt  int64
	PartnerMiles   float64
	CustomerID     pulid.ID
	CustomerName   string
	Usual          float64
	ByNow          float64
	Tendered       int
	Loads          float64
}

type target struct {
	area  *demandforecast.AreaBalance
	index int
	start int64
	end   int64
	short float64
	room  int
	point dispatchmarket.Point
}

type planner struct {
	in      *Inputs
	areas   map[int64]*demandforecast.AreaBalance
	drops   map[demandforecast.WindowKey]int64
	stays   map[demandforecast.WindowKey]int64
	drivers map[pulid.ID]*Driver
}

func newPlanner(in *Inputs) *planner {
	p := &planner{
		in:      in,
		areas:   make(map[int64]*demandforecast.AreaBalance, len(in.Areas)),
		drops:   map[demandforecast.WindowKey]int64{},
		stays:   map[demandforecast.WindowKey]int64{},
		drivers: make(map[pulid.ID]*Driver, len(in.Drivers)),
	}
	for i := range in.Areas {
		p.areas[in.Areas[i].Cell] = &in.Areas[i]
	}
	for i := range in.Drivers {
		p.drivers[in.Drivers[i].WorkerID] = &in.Drivers[i]
	}

	return p
}

func (p *planner) label(cell int64) string {
	if area, ok := p.areas[cell]; ok {
		return area.Label
	}

	return ""
}

func (p *planner) drop(t *target) int64 {
	key := demandforecast.WindowKey{Cell: t.area.Cell, Start: t.start}
	if cents, ok := p.drops[key]; ok {
		return cents
	}
	cents := int64(0)
	if p.in.Valuer.Covers(t.area.Cell) {
		cents = p.in.Valuer.ValueAt(key, pulid.Nil).Cents
	}
	p.drops[key] = cents

	return cents
}

func (p *planner) stay(d *Driver) int64 {
	key := demandforecast.WindowKey{
		Cell:  d.Cell,
		Start: dispatchmarket.SlotStart(max(d.FreeAt, p.in.Now)),
	}
	if cents, ok := p.stays[key]; ok {
		return cents
	}
	cents := int64(0)
	if p.in.Valuer.Covers(d.Cell) {
		cents = p.in.Valuer.ValueAt(key, d.WorkerID).Cents
	}
	p.stays[key] = cents

	return cents
}

func (p *planner) slotIndex(area *demandforecast.AreaBalance, at int64) (int, bool) {
	if len(area.Slots) == 0 {
		return 0, false
	}
	offset := (dispatchmarket.SlotStart(max(at, p.in.Now)) - area.Slots[0].Start) /
		dispatchmarket.SlotSeconds
	if offset < 0 || offset >= int64(len(area.Slots)) {
		return 0, false
	}

	return int(offset), true
}

func (p *planner) idle(d *Driver) bool {
	area, ok := p.areas[d.Cell]
	if !ok {
		return true
	}
	index, ok := p.slotIndex(area, d.FreeAt)
	if !ok {
		return false
	}
	slot := area.Slots[index]

	return slot.Short < MinShort && slot.Spare >= MinSpare
}

func (p *planner) targets() []*target {
	out := []*target{}
	for i := range p.in.Areas {
		area := &p.in.Areas[i]
		for index, slot := range area.Slots {
			if slot.Short < MinShort {
				continue
			}
			out = append(out, &target{
				area:  area,
				index: index,
				start: slot.Start,
				end:   slot.Start + dispatchmarket.SlotSeconds,
				short: slot.Short,
				room:  max(1, int(math.Round(slot.Short))),
				point: dispatchmarket.CenterOf(area.Cell),
			})
		}
	}
	slices.SortStableFunc(out, func(a, b *target) int {
		if c := cmp.Compare(b.short, a.short); c != 0 {
			return c
		}
		if c := cmp.Compare(a.start, b.start); c != 0 {
			return c
		}
		return cmp.Compare(a.area.Cell, b.area.Cell)
	})

	return out[:min(len(out), MaxTargets)]
}

func milesBetween(a, b dispatchmarket.Point) float64 {
	return geoutils.PlanningRoadMiles(a.Latitude, a.Longitude, b.Latitude, b.Longitude)
}

func (p *planner) emptyCost(miles float64) int64 {
	return int64(math.Round(miles * float64(p.in.EmptyMileCents)))
}

type option struct {
	step    Step
	worker  pulid.ID
	partner pulid.ID
	target  *target
}

func Plan(in *Inputs) []Step {
	if in == nil || in.Valuer == nil {
		return []Step{}
	}
	p := newPlanner(in)
	targets := p.targets()
	options := append(p.sends(targets), p.keeps(targets)...)
	slices.SortStableFunc(options, func(a, b option) int {
		if c := cmp.Compare(b.step.NetCents, a.step.NetCents); c != 0 {
			return c
		}
		return cmp.Compare(a.worker, b.worker)
	})

	used := map[pulid.ID]bool{}
	usedLanes := map[pulid.ID]bool{}
	steps := []Step{}
	for _, opt := range options {
		if used[opt.worker] || (!opt.partner.IsNil() && used[opt.partner]) || opt.target.room <= 0 {
			continue
		}
		used[opt.worker] = true
		if !opt.partner.IsNil() {
			used[opt.partner] = true
		}
		opt.target.room--
		step := opt.step
		if step.Kind == KindSend {
			step = p.book(step, opt.target, usedLanes)
		}
		steps = append(steps, step)
	}

	steps = append(steps, p.calls()...)
	slices.SortStableFunc(steps, func(a, b Step) int {
		return cmp.Compare(rank(b), rank(a))
	})

	return steps[:min(len(steps), MaxSteps)]
}

func rank(s Step) int64 {
	if s.Kind == KindCall {
		return s.DropCents
	}

	return s.NetCents
}

func (p *planner) sends(targets []*target) []option {
	out := []option{}
	for i := range p.in.Drivers {
		d := &p.in.Drivers[i]
		if !p.idle(d) {
			continue
		}
		stay := p.stay(d)
		free := max(d.FreeAt, p.in.Now)
		for _, t := range targets {
			if t.area.Cell == d.Cell {
				continue
			}
			miles := milesBetween(d.Point, t.point)
			if miles > MaxRepositionMiles {
				continue
			}
			arrive := free + geoutils.PlanningDriveSeconds(miles)
			if arrive > t.end {
				continue
			}
			drop := p.drop(t)
			cost := p.emptyCost(miles)
			net := drop - stay - cost
			if net <= 0 {
				continue
			}
			out = append(out, option{
				worker: d.WorkerID,
				target: t,
				step: Step{
					Kind:           KindSend,
					WorkerID:       d.WorkerID,
					WorkerName:     d.Name,
					FromCell:       d.Cell,
					FromLabel:      p.label(d.Cell),
					ToCell:         t.area.Cell,
					ToLabel:        t.area.Label,
					SlotStart:      t.start,
					Short:          t.short,
					FreeAt:         free,
					ArriveAt:       arrive,
					Miles:          miles,
					EmptyCostCents: cost,
					DropCents:      drop,
					StayCents:      stay,
					NetCents:       net,
				},
			})
		}
	}

	return out
}

func (p *planner) book(step Step, t *target, usedLanes map[pulid.ID]bool) Step {
	d := p.drivers[step.WorkerID]
	if d == nil {
		return step
	}
	type choice struct {
		lane  *Lane
		miles float64
	}
	choices := []choice{}
	for i := range p.in.Lanes {
		lane := &p.in.Lanes[i]
		if usedLanes[lane.MoveID] || lane.DeliveryEnd > t.end ||
			milesBetween(lane.Destination, t.point) > LaneRadiusMiles {
			continue
		}
		miles := milesBetween(d.Point, lane.Origin)
		if miles > LaneRadiusMiles ||
			step.FreeAt+geoutils.PlanningDriveSeconds(miles) > lane.PickupEnd {
			continue
		}
		choices = append(choices, choice{lane: lane, miles: miles})
	}
	if len(choices) == 0 {
		return step
	}
	slices.SortStableFunc(choices, func(a, b choice) int {
		if c := cmp.Compare(a.miles, b.miles); c != 0 {
			return c
		}
		return cmp.Compare(a.lane.PickupStart, b.lane.PickupStart)
	})

	best := choices[0]
	usedLanes[best.lane.MoveID] = true
	cost := p.emptyCost(best.miles)
	step.Kind = KindBook
	step.MoveID = best.lane.MoveID
	step.ProNumber = best.lane.ProNumber
	step.SavedCents = step.EmptyCostCents - cost
	step.Miles = best.miles
	step.EmptyCostCents = cost
	step.ArriveAt = best.lane.DeliveryEnd
	step.NetCents = step.DropCents - step.StayCents - cost
	for _, c := range choices[:min(len(choices), MaxLaneChoices)] {
		step.LaneMoveIDs = append(step.LaneMoveIDs, c.lane.MoveID)
	}

	return step
}

func (p *planner) keeps(targets []*target) []option {
	out := []option{}
	for i := range p.in.Departures {
		m := &p.in.Departures[i]
		cell, ok := dispatchmarket.CellOf(m.Origin)
		if !ok {
			continue
		}
		w := p.drivers[m.WorkerID]
		if w == nil || w.Cell == cell {
			continue
		}
		t := firstShortAfter(targets, cell, m.PickupStart)
		if t == nil {
			continue
		}
		value := p.drop(t)
		best, bestNet, bestMiles := (*Driver)(nil), int64(0), 0.0
		for j := range p.in.Drivers {
			q := &p.in.Drivers[j]
			if q.WorkerID == w.WorkerID || q.Cell == cell || !p.idle(q) {
				continue
			}
			miles := milesBetween(q.Point, m.Origin)
			if miles > MaxRepositionMiles ||
				max(q.FreeAt, p.in.Now)+geoutils.PlanningDriveSeconds(miles) > m.PickupEnd {
				continue
			}
			net := value - p.stay(q) - p.emptyCost(miles)
			if net > bestNet || (net == bestNet && best != nil && q.WorkerID < best.WorkerID) {
				best, bestNet, bestMiles = q, net, miles
			}
		}
		if best == nil || bestNet <= 0 {
			continue
		}
		out = append(out, option{
			worker:  w.WorkerID,
			partner: best.WorkerID,
			target:  t,
			step: Step{
				Kind:           KindKeep,
				WorkerID:       w.WorkerID,
				WorkerName:     w.Name,
				FromCell:       cell,
				FromLabel:      p.label(cell),
				ToCell:         cell,
				ToLabel:        t.area.Label,
				SlotStart:      t.start,
				Short:          t.short,
				FreeAt:         m.PickupStart,
				DropCents:      value,
				StayCents:      p.stay(best),
				EmptyCostCents: p.emptyCost(bestMiles),
				NetCents:       bestNet,
				MoveID:         m.MoveID,
				ProNumber:      m.ProNumber,
				PartnerID:      best.WorkerID,
				PartnerName:    best.Name,
				PartnerCell:    best.Cell,
				PartnerLabel:   p.label(best.Cell),
				PartnerFreeAt:  max(best.FreeAt, p.in.Now),
				PartnerMiles:   bestMiles,
			},
		})
	}

	return out
}

func firstShortAfter(targets []*target, cell, after int64) *target {
	var found *target
	for _, t := range targets {
		if t.area.Cell != cell || t.end <= after || t.short < KeepMinShort {
			continue
		}
		if found == nil || t.start < found.start {
			found = t
		}
	}

	return found
}
