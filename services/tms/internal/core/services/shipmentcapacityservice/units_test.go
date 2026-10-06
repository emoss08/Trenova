package shipmentcapacityservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/carriercapacity"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/dispatchconsoleservice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const now = int64(1_790_000_000)

func driver(
	first, last string,
	availability dispatchconsoleservice.DriverAvailability,
	freeAt int64,
) *dispatchconsoleservice.BoardDriver {
	return &dispatchconsoleservice.BoardDriver{
		BoardDriver: &repositories.BoardDriver{
			WorkerID:             pulid.MustNew("wrk_"),
			FirstName:            first,
			LastName:             last,
			City:                 "Omaha",
			AvailableForDispatch: true,
			TractorID:            pulid.MustNew("trc_"),
			TractorCode:          "T-339",
		},
		Availability:           availability,
		ProjectedTimeAvailable: freeAt,
		DriveRemainingMs:       int64(3.9 * msPerHour),
		HOSRecordedAt:          now - 60,
	}
}

func names(units []*services.CapacityUnit) []string {
	out := make([]string, 0, len(units))
	for _, unit := range units {
		out = append(out, unit.Name)
	}

	return out
}

func TestDriverUnits_ReadyNowThenSoonestFreeAndNobodyElse(t *testing.T) {
	t.Parallel()

	finishingSoon := driver("Sam", "Cho", dispatchconsoleservice.AvailabilityFinishing, now+3600)
	finishingSooner := driver("Elise", "Moreau", dispatchconsoleservice.AvailabilityFinishing, now+600)
	finishingLate := driver("Brett", "Long", dispatchconsoleservice.AvailabilityFinishing, now+3*3600)
	ready := driver("Lou", "Kerr", dispatchconsoleservice.AvailabilityOpen, now)
	working := driver("Mae", "Lin", dispatchconsoleservice.AvailabilityWorking, now)
	offBoard := driver("Abe", "North", dispatchconsoleservice.AvailabilityOpen, now)
	offBoard.AvailableForDispatch = false

	units := driverUnits(
		[]*dispatchconsoleservice.BoardDriver{
			finishingSoon, finishingSooner, finishingLate, ready, working, offBoard, nil,
		},
		now,
		true,
	)

	require.Equal(t, []string{"Lou Kerr", "Elise Moreau", "Sam Cho"}, names(units))
	assert.Equal(t, services.CapacityGroupReadyNow, units[0].Group)
	assert.Nil(t, units[0].FreeAt, "a driver free now has no free-at time")
	assert.Equal(t, services.CapacityGroupWithinTwoHours, units[1].Group)
	require.NotNil(t, units[1].FreeAt)
	assert.Equal(t, now+600, *units[1].FreeAt)
	assert.Equal(t, "LK", units[0].Initials)
	assert.Equal(t, "T-339", units[0].UnitLabel)
}

func TestDriverUnits_ShowHoursOnlyWhenAnELDReportedThem(t *testing.T) {
	t.Parallel()

	fresh := driver("Lou", "Kerr", dispatchconsoleservice.AvailabilityOpen, now)
	stale := driver("Sam", "Cho", dispatchconsoleservice.AvailabilityOpen, now)
	stale.HOSIsStale = true
	never := driver("Mae", "Lin", dispatchconsoleservice.AvailabilityOpen, now)
	never.HOSRecordedAt = 0

	withELD := driverUnits([]*dispatchconsoleservice.BoardDriver{fresh, stale, never}, now, true)
	byName := map[string]*services.CapacityUnit{}
	for _, unit := range withELD {
		byName[unit.Name] = unit
	}
	require.NotNil(t, byName["Lou Kerr"].Ring)
	assert.InDelta(t, 3.9, byName["Lou Kerr"].Ring.Value, 0.001)
	assert.Equal(t, hosMaxHours, byName["Lou Kerr"].Ring.Max)
	assert.True(t, byName["Lou Kerr"].Ring.Low, "under four hours is low")
	assert.Nil(t, byName["Sam Cho"].Ring, "stale hours are not shown")
	assert.Nil(t, byName["Mae Lin"].Ring, "a driver the ELD never reported has no ring")

	withoutELD := driverUnits([]*dispatchconsoleservice.BoardDriver{fresh}, now, false)
	assert.Nil(t, withoutELD[0].Ring)
	assert.Nil(t, withoutELD[0].DriveRemainingMs)
}

func performance(answered, accepted, loads int, rate string) *repositories.CarrierPerformance {
	perf := &repositories.CarrierPerformance{
		OffersAnswered: answered,
		OffersAccepted: accepted,
		PerMileLoads:   loads,
	}
	if rate != "" {
		perf.AvgRatePerMile = decimal.NewNullDecimal(decimal.RequireFromString(rate))
	}

	return perf
}

func TestCarrierUnits_PostedTrucksFirstThenCarriersThatUsuallyAccept(t *testing.T) {
	t.Parallel()

	posted := &carrier.Carrier{ID: pulid.MustNew("car_"), Name: "Heartland Express", MCNumber: "MC 10"}
	reliable := &carrier.Carrier{ID: pulid.MustNew("car_"), Name: "Werner"}
	tooFew := &carrier.Carrier{ID: pulid.MustNew("car_"), Name: "Covenant"}
	unreliable := &carrier.Carrier{ID: pulid.MustNew("car_"), Name: "Marten"}

	facts := &carrierFacts{
		postings: map[pulid.ID][]*carriercapacity.Posting{
			posted.ID: {
				{
					CarrierID:      posted.ID,
					TruckCount:     2,
					RateMethod:     carriercapacity.RateMethodPerMile,
					Rate:           decimal.NewNullDecimal(decimal.RequireFromString("2.75")),
					OriginLocation: &location.Location{City: "Kansas City"},
				},
				{CarrierID: posted.ID, TruckCount: 1, RateMethod: carriercapacity.RateMethodFlat},
			},
		},
		performance: map[pulid.ID]*repositories.CarrierPerformance{
			posted.ID:     performance(10, 3, 0, ""),
			reliable.ID:   performance(10, 9, 4, "2.35"),
			tooFew.ID:     performance(2, 2, 0, ""),
			unreliable.ID: performance(10, 5, 0, ""),
		},
		carriers: map[pulid.ID]*carrier.Carrier{
			posted.ID: posted, reliable.ID: reliable, tooFew.ID: tooFew, unreliable.ID: unreliable,
		},
	}

	units := carrierUnits(facts)

	require.Equal(t, []string{"Heartland Express", "Werner"}, names(units))
	first := units[0]
	assert.Equal(t, services.CapacityGroupTrucksPosted, first.Group)
	require.NotNil(t, first.BadgeCount)
	assert.Equal(t, 3, *first.BadgeCount, "trucks across every open posting")
	assert.Equal(t, "Kansas City", first.City)
	assert.Equal(t, "2.75", first.RatePerMile.Decimal.String(), "the posted per-mile rate wins")
	assert.Equal(t, "MC 10", first.UnitLabel)
	require.NotNil(t, first.Ring)
	assert.True(t, first.Ring.Low, "30% acceptance is low")

	second := units[1]
	assert.Equal(t, services.CapacityGroupUsuallyAccept, second.Group)
	assert.Nil(t, second.BadgeCount)
	assert.Equal(t, "2.35", second.RatePerMile.Decimal.String(), "history fills a missing posted rate")
	require.NotNil(t, second.AcceptancePercent)
	assert.InDelta(t, 90.0, *second.AcceptancePercent, 0.001)
}

func TestWeightedRatePerMile_WeighsByLoads(t *testing.T) {
	t.Parallel()

	rate := weightedRatePerMile(map[pulid.ID]*repositories.CarrierPerformance{
		pulid.MustNew("car_"): performance(0, 0, 3, "2.00"),
		pulid.MustNew("car_"): performance(0, 0, 1, "3.00"),
		pulid.MustNew("car_"): performance(0, 0, 0, ""),
	})

	require.True(t, rate.Valid)
	assert.Equal(t, "2.25", rate.Decimal.StringFixed(2))
	assert.False(t, weightedRatePerMile(nil).Valid)
}

func boardMove(originState string, pickup int64, distance float64) *dispatchconsoleservice.BoardMove {
	revenue := 2000.0
	return &dispatchconsoleservice.BoardMove{
		BoardMove: &repositories.BoardMove{
			MoveID:            pulid.MustNew("smv_"),
			ShipmentID:        pulid.MustNew("shp_"),
			OriginCity:        "Omaha",
			OriginState:       originState,
			DestinationCity:   "Chicago",
			DestinationState:  "IL",
			OriginWindowStart: pickup,
			Distance:          &distance,
			Revenue:           &revenue,
		},
	}
}

func TestRankCarrierMatches_PrefersPostingsOnTheLaneAndQuotesThem(t *testing.T) {
	t.Parallel()

	posting := &carriercapacity.Posting{
		AvailableFrom: now,
		AvailableTo:   now + 24*3600,
		OriginState:   &usstate.UsState{Abbreviation: "NE"},
		RateMethod:    carriercapacity.RateMethodPerMile,
		Rate:          decimal.NewNullDecimal(decimal.RequireFromString("2.50")),
	}
	onLane := boardMove("NE", now+3600, 400)
	offLane := boardMove("TX", now+1800, 400)
	outsideWindow := boardMove("NE", now+48*3600, 400)
	covered := boardMove("NE", now+600, 400)
	covered.IsCovered = true
	tendered := boardMove("NE", now+600, 400)
	tendered.LiveTender = &dispatchconsoleservice.MoveTenderSummary{}

	matches := rankCarrierMatches(
		[]*dispatchconsoleservice.BoardMove{offLane, outsideWindow, covered, tendered, onLane},
		[]*carriercapacity.Posting{posting},
		performance(10, 8, 0, ""),
		5,
	)

	require.Len(t, matches, 3, "covered and already-tendered loads are left out")
	assert.Equal(t, onLane.MoveID, matches[0].MoveID, "a posting on the lane outranks one off it")
	require.True(t, matches[0].Quote.Valid)
	assert.Equal(t, "1000", matches[0].Quote.Decimal.String(), "$2.50 x 400 mi")
	require.NotNil(t, matches[0].MarginPercent)
	assert.InDelta(t, 50.0, *matches[0].MarginPercent, 0.01)
	require.NotNil(t, matches[0].FitPercent)
	assert.InDelta(t, fitWindow+fitOrigin+80*fitAcceptance, *matches[0].FitPercent, 0.001)
	assert.Equal(t, offLane.MoveID, matches[1].MoveID, "inside a window but off the lane")
	assert.Equal(t, outsideWindow.MoveID, matches[2].MoveID, "outside every window: history only")
	assert.InDelta(t, 80*fitHistoryOnly, *matches[2].FitPercent, 0.001)
}

func TestRankCarrierMatches_WithoutPostingsLeansOnHistory(t *testing.T) {
	t.Parallel()

	move := boardMove("NE", now+3600, 100)
	matches := rankCarrierMatches(
		[]*dispatchconsoleservice.BoardMove{move},
		nil,
		performance(10, 9, 5, "3.00"),
		2,
	)
	require.Len(t, matches, 1)
	assert.InDelta(t, 90*fitHistoryOnly, *matches[0].FitPercent, 0.001)
	assert.Equal(t, "300", matches[0].Quote.Decimal.String())

	assert.Empty(t, rankCarrierMatches([]*dispatchconsoleservice.BoardMove{move}, nil, nil, 2),
		"a carrier with neither postings nor history has nothing to offer")
}
