package managedusage

import (
	"cmp"
	"slices"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
)

type MeterUsage struct {
	Meter    platformcatalog.Meter
	Unit     Unit
	Used     int64
	Included int64
	HardCap  bool
	Stopped  bool
}

type Usage struct {
	PeriodStart int64
	PeriodEnd   int64
	Meters      []MeterUsage
}

func UsageOf(state *State, levels map[platformcatalog.MeterKey]int64) *Usage {
	if state == nil || !state.HasWindow() {
		return nil
	}
	out := &Usage{
		PeriodStart: state.WindowStart,
		PeriodEnd:   state.WindowEnd,
		Meters:      make([]MeterUsage, 0, len(state.Meters)),
	}
	positions := make(map[platformcatalog.MeterKey]int, len(state.Meters))
	for _, m := range state.Meters {
		meter, position, known := platformcatalog.LookupMeter(m.Meter)
		if !known {
			continue
		}
		positions[m.Meter] = position
		used := m.Quantity
		if level, isLevel := levels[m.Meter]; isLevel {
			used = level
		}
		out.Meters = append(out.Meters, MeterUsage{
			Meter:    meter,
			Unit:     m.Unit,
			Used:     used,
			Included: m.Allowance,
			HardCap:  m.HardCap,
			Stopped:  m.Stopped,
		})
	}
	slices.SortFunc(out.Meters, func(a, b MeterUsage) int {
		return cmp.Compare(positions[a.Meter.Key], positions[b.Meter.Key])
	})
	return out
}
