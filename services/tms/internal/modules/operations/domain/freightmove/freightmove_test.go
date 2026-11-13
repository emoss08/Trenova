package freightmove_test

import (
	"math"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/modules/operations/domain/freightmove"
	"github.com/emoss08/trenova/shared/geoutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const mondayMidnight = int64(1_793_577_600)

var (
	dallas  = dispatchmarket.Point{Latitude: 32.7767, Longitude: -96.7970}
	okc     = dispatchmarket.Point{Latitude: 35.4676, Longitude: -97.5164}
	houston = dispatchmarket.Point{Latitude: 29.7604, Longitude: -95.3698}
	denver  = dispatchmarket.Point{Latitude: 39.7392, Longitude: -104.9903}
	acme    = pulid.ID("cus_acme")
)

func cellOf(t *testing.T, p dispatchmarket.Point) int64 {
	t.Helper()
	cell, ok := dispatchmarket.CellOf(p)
	require.True(t, ok)

	return cell
}

type fakeValuer struct {
	cents map[int64]int64
}

func (v fakeValuer) ValueAt(key demandforecast.WindowKey, _ pulid.ID) demandforecast.Valuation {
	return demandforecast.Valuation{Key: key, Cents: v.cents[key.Cell]}
}

func (v fakeValuer) Covers(cell int64) bool {
	_, ok := v.cents[cell]
	return ok
}

type world struct {
	t      *testing.T
	now    int64
	starts []int64
	in     *freightmove.Inputs
}

func newWorld(t *testing.T) *world {
	t.Helper()
	now := mondayMidnight + 9*3600
	starts := demandforecast.OutlookStarts(dispatchmarket.Grid{}, now)
	w := &world{t: t, now: now, starts: starts}
	w.in = &freightmove.Inputs{
		Now:            now,
		EmptyMileCents: 100,
		MarginPercent:  80,
		Valuer: fakeValuer{cents: map[int64]int64{
			cellOf(t, dallas):  90_000,
			cellOf(t, okc):     9_000,
			cellOf(t, houston): 30_000,
		}},
	}
	w.area(dallas, "Dallas, TX", 45_000, nil)
	w.area(okc, "Oklahoma City, OK", 30_000, nil)
	w.area(houston, "Houston, TX", 40_000, nil)

	return w
}

func (w *world) area(
	p dispatchmarket.Point,
	label string,
	revenue int64,
	set func(slots []demandforecast.SlotBalance),
) {
	slots := make([]demandforecast.SlotBalance, len(w.starts))
	for i := range slots {
		slots[i].Start = w.starts[i]
		slots[i].End = w.starts[i] + dispatchmarket.SlotSeconds
	}
	if set != nil {
		set(slots)
	}
	cell := cellOf(w.t, p)
	for i := range w.in.Areas {
		if w.in.Areas[i].Cell == cell {
			w.in.Areas[i].Slots = slots
			return
		}
	}
	w.in.Areas = append(w.in.Areas, demandforecast.AreaBalance{
		Cell: cell, Label: label, MedianRevenueCents: revenue, Slots: slots,
	})
}

func (w *world) driver(id, name string, p dispatchmarket.Point, freeAt int64) {
	w.in.Drivers = append(w.in.Drivers, freightmove.Driver{
		WorkerID: pulid.ID(id), Name: name, Point: p, Cell: cellOf(w.t, p), FreeAt: freeAt,
	})
}

func shortAt(index int, short float64) func([]demandforecast.SlotBalance) {
	return func(slots []demandforecast.SlotBalance) { slots[index].Short = short }
}

func spareAll(spare float64) func([]demandforecast.SlotBalance) {
	return func(slots []demandforecast.SlotBalance) {
		for i := range slots {
			slots[i].Spare = spare
		}
	}
}

func roadMiles(a, b dispatchmarket.Point) float64 {
	return geoutils.PlanningRoadMiles(a.Latitude, a.Longitude, b.Latitude, b.Longitude)
}

func TestPlan_SendsAnIdleDriverToAShortAreaForWhatItNets(t *testing.T) {
	t.Parallel()

	w := newWorld(t)
	w.area(dallas, "Dallas, TX", 45_000, shortAt(4, 1.2))
	w.area(okc, "Oklahoma City, OK", 30_000, spareAll(2))
	w.driver("wrk_ana", "Ana Ruiz", okc, w.now)

	steps := freightmove.Plan(w.in)

	require.Len(t, steps, 1)
	step := steps[0]
	miles := roadMiles(okc, dispatchmarket.CenterOf(cellOf(t, dallas)))
	assert.Equal(t, freightmove.KindSend, step.Kind)
	assert.Equal(t, pulid.ID("wrk_ana"), step.WorkerID)
	assert.Equal(t, "Oklahoma City, OK", step.FromLabel)
	assert.Equal(t, "Dallas, TX", step.ToLabel)
	assert.Equal(t, w.starts[4], step.SlotStart)
	assert.Equal(t, w.starts[4]+dispatchmarket.SlotSeconds, step.SlotEnd)
	assert.InDelta(t, miles, step.Miles, 1e-6)
	assert.Equal(t, int64(math.Round(miles*100)), step.EmptyCostCents)
	assert.Equal(t, int64(90_000), step.DropCents)
	assert.Equal(t, int64(9_000), step.StayCents)
	assert.Equal(t, step.DropCents-step.StayCents-step.EmptyCostCents, step.NetCents)
	assert.Equal(t, w.now+geoutils.PlanningDriveSeconds(miles), step.ArriveAt)
	assert.LessOrEqual(t, step.ArriveAt, w.starts[4]+dispatchmarket.SlotSeconds)
}

func TestPlan_SendsOnlyAsManyDriversAsTheAreaIsShortTheBestNetFirst(t *testing.T) {
	t.Parallel()

	w := newWorld(t)
	w.area(dallas, "Dallas, TX", 45_000, shortAt(4, 1.2))
	w.area(okc, "Oklahoma City, OK", 30_000, spareAll(3))
	w.area(houston, "Houston, TX", 40_000, spareAll(3))
	w.driver("wrk_houston", "Houston Driver", houston, w.now)
	w.driver("wrk_okc", "Oklahoma City Driver", okc, w.now)

	steps := freightmove.Plan(w.in)

	require.Len(t, steps, 1, "one load short takes one driver")
	assert.Equal(t, pulid.ID("wrk_okc"), steps[0].WorkerID,
		"Houston is closer, but a driver there is worth more where they are")
}

func TestPlan_MovesNobodyWhoCannotArriveInTimeOrIsNeededWhereTheyAre(t *testing.T) {
	t.Parallel()

	w := newWorld(t)
	w.area(dallas, "Dallas, TX", 45_000, shortAt(0, 2))
	w.area(houston, "Houston, TX", 40_000, shortAt(0, 1))
	w.driver("wrk_late", "Late Driver", okc, w.now+8*3600)
	w.driver("wrk_needed", "Needed Driver", houston, w.now)

	assert.Empty(
		t,
		freightmove.Plan(w.in),
		"Dallas is short now and nobody gets there before the window ends; Houston needs its own driver",
	)
}

func TestPlan_BooksALoadEndingInTheShortAreaInsteadOfRunningEmpty(t *testing.T) {
	t.Parallel()

	w := newWorld(t)
	w.area(dallas, "Dallas, TX", 45_000, shortAt(4, 1.2))
	w.area(okc, "Oklahoma City, OK", 30_000, spareAll(2))
	w.driver("wrk_ana", "Ana Ruiz", okc, w.now)
	nearOKC := dispatchmarket.Point{Latitude: 35.55, Longitude: -97.6}
	nearDallas := dispatchmarket.Point{Latitude: 32.85, Longitude: -96.85}
	w.in.Lanes = []freightmove.Lane{
		{
			MoveID: "sm_far", ProNumber: "PRO-FAR", Origin: dispatchmarket.Point{Latitude: 36.2, Longitude: -95.9},
			Destination: nearDallas, PickupEnd: w.now + 6*3600, DeliveryEnd: w.starts[4],
		},
		{
			MoveID: "sm_near", ProNumber: "PRO-NEAR", Origin: nearOKC, Destination: nearDallas,
			PickupStart: w.now + 3600, PickupEnd: w.now + 4*3600, DeliveryEnd: w.starts[4],
		},
		{
			MoveID: "sm_late", ProNumber: "PRO-LATE", Origin: nearOKC, Destination: nearDallas,
			PickupEnd: w.now + 4*3600, DeliveryEnd: w.starts[6],
		},
	}

	steps := freightmove.Plan(w.in)

	require.Len(t, steps, 1)
	step := steps[0]
	sendCost := int64(math.Round(roadMiles(okc, dispatchmarket.CenterOf(cellOf(t, dallas))) * 100))
	deadhead := roadMiles(okc, nearOKC)
	assert.Equal(t, freightmove.KindBook, step.Kind)
	assert.Equal(t, pulid.ID("sm_near"), step.MoveID)
	assert.Equal(t, "PRO-NEAR", step.ProNumber)
	assert.Equal(t, []pulid.ID{"sm_near"}, step.LaneMoveIDs,
		"a load out of reach of the driver or delivering after the window is not offered")
	assert.InDelta(t, deadhead, step.Miles, 1e-6)
	assert.Equal(t, int64(math.Round(deadhead*100)), step.EmptyCostCents)
	assert.Equal(t, sendCost-step.EmptyCostCents, step.SavedCents)
	assert.Equal(t, step.DropCents-step.StayCents-step.EmptyCostCents, step.NetCents)
	assert.Equal(t, w.starts[4], step.ArriveAt)
}

func TestPlan_KeepsADriverWhoseNextLoadWouldLeaveAShortAreaAndHandsTheLoadOn(t *testing.T) {
	t.Parallel()

	w := newWorld(t)
	w.area(dallas, "Dallas, TX", 45_000, shortAt(6, 1.4))
	w.area(okc, "Oklahoma City, OK", 30_000, spareAll(2))
	w.driver("wrk_dana", "Dana Ortiz", houston, w.starts[8])
	w.driver("wrk_joe", "Joe Kowalski", okc, w.now)
	center := dispatchmarket.CenterOf(cellOf(t, dallas))
	pickup := dispatchmarket.Point{Latitude: center.Latitude + 0.2, Longitude: center.Longitude}
	w.in.Departures = []freightmove.Departure{{
		MoveID: "sm_out", ProNumber: "PRO-OUT", WorkerID: "wrk_dana", Origin: pickup,
		PickupStart: w.starts[3], PickupEnd: w.starts[3] + 2*3600, DestinationLabel: "Houston",
	}}

	steps := freightmove.Plan(w.in)

	require.Len(t, steps, 1)
	step := steps[0]
	partnerMiles := roadMiles(okc, pickup)
	assert.Equal(t, freightmove.KindKeep, step.Kind,
		"keeping Dana and handing her load to Joe nets more than sending Joe")
	assert.Equal(t, pulid.ID("wrk_dana"), step.WorkerID)
	assert.Equal(t, "Dallas, TX", step.ToLabel)
	assert.Equal(t, w.starts[6], step.SlotStart)
	assert.Equal(t, pulid.ID("sm_out"), step.MoveID)
	assert.Equal(t, pulid.ID("wrk_joe"), step.PartnerID)
	assert.Equal(t, "Oklahoma City, OK", step.PartnerLabel)
	assert.InDelta(t, partnerMiles, step.PartnerMiles, 1e-6)
	assert.Equal(t,
		int64(90_000)-9_000-int64(math.Round(partnerMiles*100)),
		step.NetCents,
		"keeping is worth the area's driver, less what the partner was worth and its empty miles",
	)
}

func TestPlan_AsksACustomerBehindItsUsualPaceAndRanksItByWhatItCouldRecover(t *testing.T) {
	t.Parallel()

	w := newWorld(t)
	w.area(dallas, "Dallas, TX", 45_000, func(slots []demandforecast.SlotBalance) {
		slots[2].Customers = []demandforecast.CustomerSlot{
			{CustomerID: acme, Name: "Acme Foods", Usual: 3, ByNow: 2.5, Tendered: 0},
		}
		slots[3].Customers = []demandforecast.CustomerSlot{
			{CustomerID: acme, Name: "Acme Foods", Usual: 2, ByNow: 0.5, Tendered: 1},
		}
	})

	steps := freightmove.Plan(w.in)

	require.Len(t, steps, 1)
	step := steps[0]
	assert.Equal(t, freightmove.KindCall, step.Kind)
	assert.Equal(t, acme, step.CustomerID)
	assert.Equal(t, "Dallas, TX", step.ToLabel)
	assert.Equal(t, w.starts[2], step.SlotStart, "named for the customer's busiest window")
	assert.Equal(t, w.starts[2]+dispatchmarket.SlotSeconds, step.SlotEnd)
	assert.InDelta(t, 5, step.Usual, 1e-9)
	assert.InDelta(t, 3, step.ByNow, 1e-9)
	assert.Equal(t, 1, step.Tendered)
	assert.InDelta(t, 4, step.Loads, 1e-9)
	assert.Equal(t,
		int64(math.Round(4*45_000*0.8*freightmove.CallRecoveryShare)),
		step.DropCents,
	)
}

func TestStandingOf_UsesTheSameThresholdsAsTheSuggestions(t *testing.T) {
	t.Parallel()

	assert.Equal(t, freightmove.StandingBehind, freightmove.StandingOf(5, 3, 2))
	assert.Equal(t, freightmove.StandingOn, freightmove.StandingOf(5, 3, 3))
	assert.Equal(t, freightmove.StandingOn, freightmove.StandingOf(0.8, 3, 0),
		"a customer that usually sends under a load is never behind")
	assert.Equal(t, freightmove.StandingAhead, freightmove.StandingOf(6, 2.5, 4))
}

func TestReach_SaysForEveryDriverWhetherTheyCanGetThereAndWhatItNets(t *testing.T) {
	t.Parallel()

	w := newWorld(t)
	w.area(dallas, "Dallas, TX", 45_000, shortAt(4, 2))
	w.driver("wrk_here", "Here Driver", dallas, w.now)
	w.driver("wrk_ontime", "On Time", okc, w.now)
	w.driver("wrk_late", "Too Late", okc, w.starts[4]+2*3600)
	w.driver("wrk_far", "Too Far", denver, w.now)

	rows := freightmove.Reach(w.in, cellOf(t, dallas), w.starts[4])

	require.Len(t, rows, 4)
	kinds := []freightmove.Reachability{}
	for _, row := range rows {
		kinds = append(kinds, row.Reach)
	}
	assert.Equal(t, []freightmove.Reachability{
		freightmove.ReachHere, freightmove.ReachOnTime, freightmove.ReachTooLate, freightmove.ReachTooFar,
	}, kinds)
	assert.Equal(
		t,
		int64(90_000),
		rows[0].NetCents,
		"a driver already there is worth the whole area",
	)
	assert.Equal(t, "Oklahoma City, OK", rows[1].Label)
	assert.Positive(t, rows[1].NetCents)
}

func TestReach_IsEmptyForAnAreaOrWindowOutsideTheOutlook(t *testing.T) {
	t.Parallel()

	w := newWorld(t)
	assert.Empty(t, freightmove.Reach(w.in, cellOf(t, denver), w.starts[0]))
	assert.Empty(
		t,
		freightmove.Reach(w.in, cellOf(t, dallas), w.starts[11]+dispatchmarket.SlotSeconds),
	)
}

func TestPlan_ConsidersOnlyTheBiggestShortfallsWhenThereAreMoreThanItCanPrice(t *testing.T) {
	t.Parallel()

	w := newWorld(t)
	w.area(dallas, "Dallas, TX", 45_000, func(slots []demandforecast.SlotBalance) {
		for i := range slots {
			slots[i].Short = 0.6
		}
	})
	w.area(okc, "Oklahoma City, OK", 30_000, spareAll(1))
	extra := make([]demandforecast.AreaBalance, 0, freightmove.MaxTargets)
	for i := range freightmove.MaxTargets {
		slots := make([]demandforecast.SlotBalance, len(w.starts))
		for j := range slots {
			slots[j] = demandforecast.SlotBalance{Start: w.starts[j]}
		}
		slots[11].Short = 5
		extra = append(extra, demandforecast.AreaBalance{
			Cell: int64(900_000 + i), Label: "Elsewhere", Slots: slots,
		})
	}
	w.in.Areas = append(w.in.Areas, extra...)
	w.driver("wrk_ana", "Ana Ruiz", okc, w.now)

	steps := freightmove.Plan(w.in)

	assert.Empty(
		t,
		steps,
		"Dallas's small shortfalls fall past the cut once bigger ones fill it, and those are out of reach",
	)
}

func TestPlan_SendsNoMoreDriversOutOfAnAreaThanItCanSpare(t *testing.T) {
	t.Parallel()

	w := newWorld(t)
	w.area(dallas, "Dallas, TX", 45_000, shortAt(4, 3.4))
	w.area(okc, "Oklahoma City, OK", 30_000, spareAll(1.3))
	w.driver("wrk_ana", "Ana Ruiz", okc, w.now)
	w.driver("wrk_ben", "Ben Ortiz", okc, w.now)
	w.driver("wrk_cy", "Cy Park", okc, w.now)

	steps := freightmove.Plan(w.in)

	require.Len(t, steps, 1,
		"three drivers wait in an area that can spare one: the other two are its own cover")
	assert.Equal(t, freightmove.KindSend, steps[0].Kind)
	assert.Equal(t, "Oklahoma City, OK", steps[0].FromLabel)

	w.area(okc, "Oklahoma City, OK", 30_000, spareAll(2.6))
	assert.Len(t, freightmove.Plan(w.in), 3, "an area with three to spare can send all three")
}

func TestPlan_GivesAScarceWindowToTheDriverWhoCanGoNowhereElse(t *testing.T) {
	t.Parallel()

	w := newWorld(t)
	w.in.Valuer = fakeValuer{cents: map[int64]int64{
		cellOf(t, dallas):  90_000,
		cellOf(t, okc):     9_000,
		cellOf(t, houston): 100_000,
	}}
	w.area(dallas, "Dallas, TX", 45_000, shortAt(6, 1.1))
	w.area(houston, "Houston, TX", 40_000, shortAt(3, 1.1))
	w.area(okc, "Oklahoma City, OK", 30_000, spareAll(2))
	w.driver("wrk_ana", "Ana Ruiz", okc, w.now)
	w.driver("wrk_ben", "Ben Ortiz", okc, w.starts[5])

	steps := freightmove.Plan(w.in)

	require.Len(t, steps, 2,
		"Ana nets most in Dallas, but Ben can only make Dallas: sending Ana to Houston fills both")
	byWorker := map[pulid.ID]freightmove.Step{}
	for _, step := range steps {
		byWorker[step.WorkerID] = step
	}
	assert.Equal(t, "Houston, TX", byWorker["wrk_ana"].ToLabel)
	assert.Equal(t, "Dallas, TX", byWorker["wrk_ben"].ToLabel)
	assert.Greater(t, steps[0].NetCents+steps[1].NetCents,
		90_000-9_000-int64(roadMiles(okc, dispatchmarket.CenterOf(cellOf(t, dallas)))*100),
		"together they net more than the one move a best-first pick would have made")
}

func TestPlan_NeverUsesADriverTwiceAcrossSendsAndKeeps(t *testing.T) {
	t.Parallel()

	w := newWorld(t)
	w.area(dallas, "Dallas, TX", 45_000, shortAt(4, 2.2))
	w.area(okc, "Oklahoma City, OK", 30_000, spareAll(2))
	w.area(houston, "Houston, TX", 40_000, spareAll(2))
	w.driver("wrk_ana", "Ana Ruiz", okc, w.now)
	w.driver("wrk_ben", "Ben Ortiz", houston, w.now)
	w.in.Departures = []freightmove.Departure{{
		MoveID:      "sm_out",
		ProNumber:   "PRO-9",
		WorkerID:    "wrk_ana",
		Origin:      dallas,
		PickupStart: w.starts[3],
		PickupEnd:   w.starts[4],
	}}

	steps := freightmove.Plan(w.in)

	require.NotEmpty(t, steps)
	seen := map[pulid.ID]int{}
	for _, step := range steps {
		seen[step.WorkerID]++
		if !step.PartnerID.IsNil() {
			seen[step.PartnerID]++
		}
	}
	for worker, times := range seen {
		assert.Equal(t, 1, times, "%s is asked to do one thing", worker)
	}
	assert.LessOrEqual(t, len(steps), 2, "the window is short two drivers")
}
