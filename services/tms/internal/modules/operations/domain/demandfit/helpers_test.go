package demandfit_test

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/demandfit"
	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

const mondayMidnight = int64(1_793_577_600)

var (
	dallas = dispatchmarket.Point{Latitude: 32.7767, Longitude: -96.7970}
	acme   = pulid.ID("cus_acme")
)

func dallasCell(t *testing.T) int64 {
	t.Helper()
	cell, ok := dispatchmarket.CellOf(dallas)
	require.True(t, ok)

	return cell
}

func nowAtWeek(week int64) int64 {
	return mondayMidnight + (week-dispatchmarket.WeekOf(mondayMidnight))*dispatchmarket.WeekSeconds
}

func poisson(r *rand.Rand, mean float64) int32 {
	limit, k, p := math.Exp(-mean), int32(0), 1.0
	for {
		p *= r.Float64()
		if p <= limit {
			return k
		}
		k++
	}
}

func areaRate(t *testing.T, fitted *demandforecast.Fitted, slot int16) *demandforecast.Rate {
	t.Helper()
	for _, rate := range fitted.Rates {
		if rate.CustomerID.IsNil() && rate.Slot == slot {
			return rate
		}
	}
	t.Fatalf("no area rate for slot %d", slot)

	return nil
}

func patternedHistory(t *testing.T, weeks int64) *demandfit.History {
	t.Helper()
	r := rand.New(rand.NewPCG(3, 5))
	current := dispatchmarket.WeekOf(mondayMidnight) + weeks
	h := &demandfit.History{Now: nowAtWeek(current)}
	means := map[int16]float64{1: 4, 7: 3, 13: 5, 25: 2, 31: 1}
	for week := current - weeks; week < current; week++ {
		for slot, mean := range means {
			h.Counts = append(h.Counts, demandfit.LoadCount{
				CustomerID: acme, Cell: dallasCell(t), Week: week, Slot: slot,
				Loads: poisson(r, mean),
			})
		}
	}

	return h
}
