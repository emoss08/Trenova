package freightmove

import (
	"cmp"
	"slices"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/shared/geoutils"
	"github.com/emoss08/trenova/shared/pulid"
)

type Reachability string

const (
	ReachHere     = Reachability("Here")
	ReachOnTime   = Reachability("OnTime")
	ReachTooLate  = Reachability("TooLate")
	ReachTooFar   = Reachability("TooFar")
	ReachNotWorth = Reachability("NotWorth")
)

type ReachRow struct {
	WorkerID   pulid.ID
	WorkerName string
	Cell       int64
	Label      string
	Point      dispatchmarket.Point
	FreeAt     int64
	ArriveAt   int64
	Miles      float64
	NetCents   int64
	Reach      Reachability
}

func Reach(in *Inputs, cell int64, slotStart int64) []ReachRow {
	if in == nil || in.Valuer == nil {
		return []ReachRow{}
	}
	p := newPlanner(in)
	area, ok := p.areas[cell]
	if !ok {
		return []ReachRow{}
	}
	index, ok := p.slotIndex(area, slotStart)
	if !ok {
		return []ReachRow{}
	}
	slot := area.Slots[index]
	t := &target{
		area:  area,
		index: index,
		start: slot.Start,
		end:   slot.Start + dispatchmarket.SlotSeconds,
		short: slot.Short,
		point: dispatchmarket.CenterOf(cell),
	}
	drop := p.drop(t)

	rows := []ReachRow{}
	for i := range in.Drivers {
		d := &in.Drivers[i]
		free := max(d.FreeAt, in.Now)
		row := ReachRow{
			WorkerID:   d.WorkerID,
			WorkerName: d.Name,
			Cell:       d.Cell,
			Label:      p.label(d.Cell),
			Point:      d.Point,
			FreeAt:     free,
		}
		if d.Cell == cell {
			if free >= t.end {
				continue
			}
			row.Reach, row.ArriveAt, row.NetCents = ReachHere, free, drop
			rows = append(rows, row)
			continue
		}
		row.Miles = milesBetween(d.Point, t.point)
		if row.Miles > MaxReachMiles {
			continue
		}
		row.ArriveAt = free + geoutils.PlanningDriveSeconds(row.Miles)
		row.NetCents = drop - p.stay(d) - p.emptyCost(row.Miles)
		switch {
		case row.Miles > MaxRepositionMiles:
			row.Reach = ReachTooFar
		case row.ArriveAt > t.end:
			row.Reach = ReachTooLate
		case row.NetCents <= 0:
			row.Reach = ReachNotWorth
		default:
			row.Reach = ReachOnTime
		}
		rows = append(rows, row)
	}

	order := map[Reachability]int{
		ReachHere: 0, ReachOnTime: 1, ReachNotWorth: 2, ReachTooLate: 3, ReachTooFar: 4,
	}
	slices.SortStableFunc(rows, func(a, b ReachRow) int {
		if c := cmp.Compare(order[a.Reach], order[b.Reach]); c != 0 {
			return c
		}
		if c := cmp.Compare(b.NetCents, a.NetCents); c != 0 {
			return c
		}
		return cmp.Compare(a.WorkerID, b.WorkerID)
	})

	return rows[:min(len(rows), MaxReach)]
}
