package demandforecast_test

import (
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/demandfit"
	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatcheconomics"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/shared/assignmentsolver"
	"github.com/emoss08/trenova/shared/geoutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	simDrivers      = 10
	simHistoryWeeks = 16
	simEvalWeeks    = 4
	simScale        = 1.1
	simHorizon      = int64(12 * 3600)
	simWindow       = int64(4 * 3600)
	simDwell        = int64(2 * 3600)
	simBaseCents    = 15_000
	simMileCents    = 250
	simPickupStep   = int64(30 * 60)
)

type simCity struct {
	label    string
	point    dispatchmarket.Point
	customer pulid.ID
	days     [7]float64
	lanes    [4]float64
}

var simCities = [...]simCity{
	{
		label:    "Dallas, TX",
		point:    dispatchmarket.Point{Latitude: 32.7767, Longitude: -96.7970},
		customer: pulid.ID("cus_dallas"),
		days:     [7]float64{0.6, 2, 0.6, 2, 0.6, 0.3, 0.2},
		lanes:    [4]float64{0, 4, 3, 3},
	},
	{
		label:    "Houston, TX",
		point:    dispatchmarket.Point{Latitude: 29.7604, Longitude: -95.3698},
		customer: pulid.ID("cus_houston"),
		days:     [7]float64{2, 0.6, 2, 0.6, 2, 0.3, 0.2},
		lanes:    [4]float64{5, 0, 4, 1},
	},
	{
		label:    "San Antonio, TX",
		point:    dispatchmarket.Point{Latitude: 29.4241, Longitude: -98.4936},
		customer: pulid.ID("cus_san_antonio"),
		days:     [7]float64{1, 1, 1, 1, 1, 0.3, 0.2},
		lanes:    [4]float64{4, 5, 0, 1},
	},
	{
		label:    "Oklahoma City, OK",
		point:    dispatchmarket.Point{Latitude: 35.4676, Longitude: -97.5164},
		customer: pulid.ID("cus_oklahoma_city"),
		days:     [7]float64{0.3, 0.3, 0.3, 0.3, 0.3, 0.1, 0.1},
		lanes:    [4]float64{8, 1, 1, 0},
	},
}

var (
	simHours       = [6]float64{0.5, 1, 1, 0.9, 0.7, 0.5}
	simLeadWeights = []float64{5, 3, 2}
	simLeadBands   = [][2]int64{
		{18 * 3600, 72 * 3600},
		{4 * 3600, 18 * 3600},
		{1800, 4 * 3600},
	}
)

type simLoad struct {
	city     int
	dest     int
	pickupAt int64
	bookedAt int64
	miles    float64
	revenue  int64
}

type simDriver struct {
	id     pulid.ID
	city   int
	freeAt int64
}

type simReach struct {
	empty  float64
	depart int64
	finish int64
}

type simState struct {
	drivers []simDriver
	covered []bool
	empty   []float64
}

type simMode int

const (
	simOff simMode = iota
	simDropOnly
	simRelative
)

type simResult struct {
	loads      int
	covered    int
	emptyMiles float64
	revenue    int64
}

func (r simResult) contribution(emptyMileCents int64) float64 {
	return float64(r.revenue) - r.emptyMiles*float64(emptyMileCents)
}

func (r simResult) coverage() float64 {
	return float64(r.covered) / float64(r.loads)
}

func (r simResult) emptyPerLoad() float64 {
	return r.emptyMiles / float64(r.covered)
}

type simWorld struct {
	cells   [len(simCities)]int64
	miles   [len(simCities)][len(simCities)]float64
	eval    []simLoad
	start   int64
	end     int64
	fitted  *demandforecast.Fitted
	quality *demandforecast.Quality
	econ    dispatcheconomics.Economics
}

func newSimWorld(t *testing.T, seed uint64) *simWorld {
	t.Helper()
	w := &simWorld{econ: dispatcheconomics.Default()}
	seen := map[int64]bool{}
	for a := range simCities {
		from := simCities[a].point
		cell, ok := dispatchmarket.CellOf(from)
		require.True(t, ok)
		require.False(t, seen[cell], "%s shares a map area with another city", simCities[a].label)
		seen[cell] = true
		w.cells[a] = cell
		for b := range simCities {
			if a != b {
				to := simCities[b].point
				w.miles[a][b] = geoutils.PlanningRoadMiles(
					from.Latitude, from.Longitude, to.Latitude, to.Longitude,
				)
			}
		}
	}

	r := rand.New(rand.NewPCG(seed, 11))
	first := dispatchmarket.WeekOf(mondayMidnight)
	w.start = nowAtWeek(first + simHistoryWeeks)
	w.end = w.start + simEvalWeeks*dispatchmarket.WeekSeconds
	history := w.history(w.generate(r, first, first+simHistoryWeeks))
	w.eval = w.generate(r, first+simHistoryWeeks, first+simHistoryWeeks+simEvalWeeks)
	w.fitted = demandfit.Fit(history)
	w.quality, _ = demandfit.Evaluate(history)

	return w
}

func simPick(r *rand.Rand, weights []float64) int {
	total := 0.0
	for _, weight := range weights {
		total += weight
	}
	roll, pick := r.Float64()*total, 0
	for i, weight := range weights {
		if weight <= 0 {
			continue
		}
		pick = i
		if roll < weight {
			return i
		}
		roll -= weight
	}

	return pick
}

func (w *simWorld) generate(r *rand.Rand, fromWeek, toWeek int64) []simLoad {
	var loads []simLoad
	for week := fromWeek; week < toWeek; week++ {
		monday := nowAtWeek(week)
		for slot := range int64(dispatchmarket.SlotsPerWeek) {
			slotStart := monday + slot*dispatchmarket.SlotSeconds
			for city := range simCities {
				mean := simScale * simHours[slot%6] * simCities[city].days[slot/6]
				for range poisson(r, mean) {
					loads = append(loads, w.load(r, city, slotStart))
				}
			}
		}
	}

	return loads
}

func (w *simWorld) load(r *rand.Rand, city int, slotStart int64) simLoad {
	dest := simPick(r, simCities[city].lanes[:])
	pickup := slotStart + r.Int64N(dispatchmarket.SlotSeconds/simPickupStep)*simPickupStep
	band := simLeadBands[simPick(r, simLeadWeights)]
	lead := band[0] + r.Int64N(band[1]-band[0])
	miles := w.miles[city][dest]

	return simLoad{
		city:     city,
		dest:     dest,
		pickupAt: pickup,
		bookedAt: pickup - lead,
		miles:    miles,
		revenue:  simBaseCents + int64(math.Round(miles*simMileCents)),
	}
}

func (w *simWorld) history(loads []simLoad) *demandfit.History {
	type countKey struct {
		city int
		week int64
		slot int16
	}
	h := &demandfit.History{Now: w.start}
	index := map[countKey]int{}
	var leads [len(simCities)][len(demandforecast.LeadHours)]int32
	var revenues [len(simCities)][]int64
	var totals [len(simCities)]int32
	for _, l := range loads {
		key := countKey{
			city: l.city,
			week: dispatchmarket.WeekOf(l.pickupAt),
			slot: dispatchmarket.SlotOf(l.pickupAt),
		}
		i, ok := index[key]
		if !ok {
			i = len(h.Counts)
			index[key] = i
			h.Counts = append(h.Counts, demandfit.LoadCount{
				CustomerID: simCities[l.city].customer,
				Cell:       w.cells[l.city],
				Week:       key.week,
				Slot:       key.slot,
			})
		}
		h.Counts[i].Loads++
		leads[l.city][demandforecast.LeadBucket(l.pickupAt-l.bookedAt)]++
		revenues[l.city] = append(revenues[l.city], l.revenue)
		totals[l.city]++
	}
	for city := range simCities {
		for bucket, n := range leads[city] {
			if n > 0 {
				h.Leads = append(h.Leads, demandfit.LeadCount{
					CustomerID: simCities[city].customer, Bucket: bucket, Loads: n,
				})
			}
		}
		slices.Sort(revenues[city])
		h.Areas = append(h.Areas, demandfit.AreaSummary{
			Cell:               w.cells[city],
			Label:              simCities[city].label,
			MedianRevenueCents: revenues[city][len(revenues[city])/2],
			Loads:              totals[city],
		})
	}

	return h
}

func (w *simWorld) run(mode simMode) simResult {
	s := &simState{covered: make([]bool, len(w.eval)), empty: make([]float64, len(w.eval))}
	for i := range simDrivers {
		s.drivers = append(s.drivers, simDriver{
			id:     pulid.ID(fmt.Sprintf("wrk_%02d", i)),
			city:   i % len(simCities),
			freeAt: w.start,
		})
	}
	for now := w.start; now < w.end; now += dispatchmarket.SlotSeconds {
		w.tick(s, now, mode)
	}

	var result simResult
	for i, l := range w.eval {
		if l.pickupAt < w.start || l.pickupAt >= w.end-simHorizon {
			continue
		}
		result.loads++
		if s.covered[i] {
			result.covered++
			result.emptyMiles += s.empty[i]
			result.revenue += l.revenue
		}
	}

	return result
}

func (w *simWorld) tick(s *simState, now int64, mode simMode) {
	var rows []int
	for i, l := range w.eval {
		if !s.covered[i] && l.bookedAt <= now && l.pickupAt < now+simHorizon &&
			l.pickupAt+simWindow > now {
			rows = append(rows, i)
		}
	}
	if len(rows) == 0 {
		return
	}

	var plan []int
	if mode == simOff {
		plan = w.solve(s, now, rows, nil, mode)
	} else {
		base := w.forecaster(s, now, rows)
		first := w.solve(s, now, rows, base.WithSupply(w.supply(s, now, rows, nil)), mode)
		plan = w.solve(s, now, rows, base.WithSupply(w.supply(s, now, rows, first)), mode)
	}

	for row, col := range plan {
		if col == assignmentsolver.Unassigned {
			continue
		}
		l := w.eval[rows[row]]
		driver := &s.drivers[col]
		reach, ok := w.reach(*driver, l, now)
		if !ok || reach.depart >= now+dispatchmarket.SlotSeconds {
			continue
		}
		s.covered[rows[row]] = true
		s.empty[rows[row]] = reach.empty
		driver.city, driver.freeAt = l.dest, reach.finish
	}
}

func (w *simWorld) reach(d simDriver, l simLoad, now int64) (simReach, bool) {
	empty := w.miles[d.city][l.city]
	drive := geoutils.PlanningDriveSeconds(empty)
	arrive := max(d.freeAt, now) + drive
	if arrive > l.pickupAt+simWindow {
		return simReach{}, false
	}
	start := max(arrive, l.pickupAt)

	return simReach{
		empty:  empty,
		depart: start - drive,
		finish: start + simDwell + geoutils.PlanningDriveSeconds(l.miles),
	}, true
}

func (w *simWorld) forecaster(s *simState, now int64, rows []int) *demandforecast.Forecaster {
	booked := make([]demandforecast.Booked, 0, len(w.eval))
	for i, l := range w.eval {
		if l.bookedAt > now || l.pickupAt+simWindow <= now {
			continue
		}
		open := 1
		if s.covered[i] {
			open = 0
		}
		booked = append(booked, demandforecast.Booked{
			Cell: w.cells[l.city], SlotStart: l.pickupAt, Total: 1, Open: open,
		})
	}
	solving := make([]demandforecast.Booked, 0, len(rows))
	for _, i := range rows {
		l := w.eval[i]
		solving = append(solving, demandforecast.Booked{
			Cell: w.cells[l.city], SlotStart: l.pickupAt, Open: 1,
		})
	}

	return demandforecast.NewForecaster(&demandforecast.ForecasterParams{
		Now: now,
		Index: demandforecast.NewIndex(&demandforecast.IndexParams{
			Quality: w.quality,
			Rates:   w.fitted.Rates,
			Areas:   w.fitted.Areas,
			Curves:  w.fitted.Curves,
		}),
		Booked:        booked,
		Solving:       solving,
		MarginPercent: w.econ.CoverageMarginPercent,
		CapCents:      w.econ.AnticipationMaxCents,
	})
}

func (w *simWorld) supply(s *simState, now int64, rows, plan []int) []demandforecast.Driver {
	out := make([]demandforecast.Driver, len(s.drivers))
	for col, d := range s.drivers {
		out[col] = demandforecast.Driver{WorkerID: d.id, Cell: w.cells[d.city], FreeAt: d.freeAt}
	}
	for row, col := range plan {
		if col == assignmentsolver.Unassigned {
			continue
		}
		l := w.eval[rows[row]]
		if reach, ok := w.reach(s.drivers[col], l, now); ok {
			out[col].Cell, out[col].FreeAt = w.cells[l.dest], reach.finish
		}
	}

	return out
}

func (w *simWorld) solve(
	s *simState,
	now int64,
	rows []int,
	f *demandforecast.Forecaster,
	mode simMode,
) []int {
	problem := assignmentsolver.Problem{
		Cols:      len(s.drivers),
		Rows:      make([][]assignmentsolver.Arc, len(rows)),
		Uncovered: make([]int64, len(rows)),
	}
	for row, i := range rows {
		l := w.eval[i]
		problem.Uncovered[row] = l.revenue
		for col, d := range s.drivers {
			reach, ok := w.reach(d, l, now)
			if !ok {
				continue
			}
			cost := int64(math.Round(reach.empty * float64(w.econ.EmptyMileCents)))
			if f != nil {
				value, valued := f.Value(demandforecast.Query{
					Point: simCities[l.dest].point, FreeAt: reach.finish, WorkerID: d.id,
				})
				if valued {
					cost -= value.Cents
				}
				if stay, stayed := f.Value(demandforecast.Query{
					Point: simCities[d.city].point, FreeAt: max(d.freeAt, now), WorkerID: d.id,
				}); stayed && mode == simRelative {
					cost += stay.Cents
				}
			}
			problem.Rows[row] = append(
				problem.Rows[row],
				assignmentsolver.Arc{Col: col, Cost: cost},
			)
		}
	}

	return assignmentsolver.Solve(problem).RowAssignment
}

/*
Anticipation earns by covering loads a myopic plan leaves, which costs some empty driving, so
the measure is contribution (revenue covered less empty miles at the organization's rate), not
empty miles per load. Valuing a move by where it leaves the driver less what the driver was
worth staying put charges for pulling a truck out of an area about to get busy. Six random
worlds, not one: a single world once showed a gain a counting bug had made.
*/
func TestSimulation_AnticipationEarnsMoreWithoutLosingCoverage(t *testing.T) {
	t.Parallel()

	var off, drop, relative float64
	for _, seed := range []uint64{2026, 7, 42, 99, 314, 1234} {
		w := newSimWorld(t, seed)
		require.True(t, w.quality.Ready(), "sixteen weeks of a steady rhythm pass the backtest")
		rate := w.econ.EmptyMileCents
		o, d, r := w.run(simOff), w.run(simDropOnly), w.run(simRelative)
		require.Equal(t, o.loads, r.loads)
		require.Positive(
			t,
			o.emptyMiles,
			"the world needs repositioning for anticipation to matter",
		)
		t.Logf("seed %4d: off %.1f%% $%.0f, drop %.1f%% $%.0f, relative %.1f%% $%.0f", seed,
			100*o.coverage(), o.contribution(rate)/100, 100*d.coverage(), d.contribution(rate)/100,
			100*r.coverage(), r.contribution(rate)/100)
		assert.GreaterOrEqual(t, r.coverage(), o.coverage(),
			"anticipation covers no fewer loads (seed %d)", seed)
		off += o.contribution(rate)
		drop += d.contribution(rate)
		relative += r.contribution(rate)
	}

	assert.GreaterOrEqual(t, relative, 1.01*off,
		"anticipation earns at least 1%% more than a myopic plan across the worlds")
	assert.Greater(t, relative, drop,
		"charging for the value left behind beats valuing the drop alone across the worlds")
}
