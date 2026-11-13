package freightmove

import (
	"cmp"
	"math"
	"slices"

	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/shared/pulid"
)

const MaxPace = 20

type PaceSlot struct {
	Start    int64
	End      int64
	Usual    float64
	ByNow    float64
	Tendered int
}

type Pace struct {
	CustomerID   pulid.ID
	CustomerName string
	Cell         int64
	Label        string
	Slots        []PaceSlot
}

type Standing string

const (
	StandingBehind = Standing("Behind")
	StandingAhead  = Standing("Ahead")
	StandingOn     = Standing("OnPace")
)

func StandingOf(usual, byNow float64, tendered int) Standing {
	switch {
	case usual >= CallMinUsual && float64(tendered) <= byNow-CallBehindBy:
		return StandingBehind
	case float64(tendered) >= byNow+AheadBy:
		return StandingAhead
	default:
		return StandingOn
	}
}

func (p *Pace) Totals(slots int) (usual, byNow float64, tendered int) {
	for _, slot := range p.Slots[:min(slots, len(p.Slots))] {
		usual += slot.Usual
		byNow += slot.ByNow
		tendered += slot.Tendered
	}

	return usual, byNow, tendered
}

type paceKey struct {
	customer pulid.ID
	cell     int64
}

func Paces(areas []demandforecast.AreaBalance) []Pace {
	byKey := map[paceKey]*Pace{}
	for i := range areas {
		area := &areas[i]
		for index, slot := range area.Slots {
			for _, customer := range slot.Customers {
				key := paceKey{customer: customer.CustomerID, cell: area.Cell}
				pace := byKey[key]
				if pace == nil {
					pace = &Pace{
						CustomerID:   customer.CustomerID,
						CustomerName: customer.Name,
						Cell:         area.Cell,
						Label:        area.Label,
						Slots:        make([]PaceSlot, len(area.Slots)),
					}
					for j := range pace.Slots {
						pace.Slots[j].Start = area.Slots[j].Start
						pace.Slots[j].End = area.Slots[j].End
					}
					byKey[key] = pace
				}
				pace.Slots[index].Usual += customer.Usual
				pace.Slots[index].ByNow += customer.ByNow
				pace.Slots[index].Tendered += customer.Tendered
			}
		}
	}

	out := make([]Pace, 0, len(byKey))
	for _, pace := range byKey {
		usual, _, tendered := pace.Totals(len(pace.Slots))
		if usual < CallMinUsual && tendered == 0 {
			continue
		}
		out = append(out, *pace)
	}
	slices.SortFunc(out, func(a, b Pace) int {
		ua, _, _ := a.Totals(len(a.Slots))
		ub, _, _ := b.Totals(len(b.Slots))
		if c := cmp.Compare(ub, ua); c != 0 {
			return c
		}
		if c := cmp.Compare(a.CustomerID, b.CustomerID); c != 0 {
			return c
		}
		return cmp.Compare(a.Cell, b.Cell)
	})

	return out[:min(len(out), MaxPace)]
}

func (p *planner) calls() []Step {
	out := []Step{}
	for _, pace := range Paces(p.in.Areas) {
		usual, byNow, tendered := pace.Totals(CallHorizonSlots)
		if StandingOf(usual, byNow, tendered) != StandingBehind {
			continue
		}
		area := p.areas[pace.Cell]
		loads := math.Max(0, usual-float64(tendered))
		busiest := busiestSlot(pace, CallHorizonSlots)
		worth := loads * float64(area.MedianRevenueCents) * float64(p.in.MarginPercent) / 100
		out = append(out, Step{
			Kind:         KindCall,
			ToCell:       pace.Cell,
			ToLabel:      pace.Label,
			SlotStart:    busiest.Start,
			SlotEnd:      busiest.End,
			CustomerID:   pace.CustomerID,
			CustomerName: pace.CustomerName,
			Usual:        usual,
			ByNow:        byNow,
			Tendered:     tendered,
			Loads:        loads,
			DropCents:    int64(math.Round(worth * CallRecoveryShare)),
		})
	}

	return out
}

func busiestSlot(pace Pace, slots int) PaceSlot {
	busiest := PaceSlot{Usual: -1}
	for _, slot := range pace.Slots[:min(slots, len(pace.Slots))] {
		if slot.Usual > busiest.Usual {
			busiest = slot
		}
	}

	return busiest
}
