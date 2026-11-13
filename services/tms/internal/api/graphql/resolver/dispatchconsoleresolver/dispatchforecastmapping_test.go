package dispatchconsoleresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/modules/operations/domain/driverpositioning"
	"github.com/emoss08/trenova/internal/modules/operations/domain/freightmove"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/freightoutlookports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func forecastReason(
	label string,
	customers ...demandforecast.CustomerShare,
) *demandforecast.Reason {
	return &demandforecast.Reason{
		AreaLabel:        label,
		WindowStart:      1_800_000_000,
		WindowEnd:        1_800_043_200,
		TypicalLow:       2,
		TypicalHigh:      6,
		Booked:           1,
		OpenBooked:       1,
		ExpectedUnbooked: 2.5,
		DriversExpected:  3,
		Customers:        customers,
		ValueCents:       42_000,
	}
}

func TestMapDispatchCandidate_CarriesAForecastEndMarketWithEveryField(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	candidate := mapDispatchCandidate(&dispatchplanports.CandidateScore{
		WorkerID: pulid.MustNew("wrk_"),
		MoveID:   pulid.MustNew("smv_"),
		EndMarket: &dispatchplanports.EndMarket{
			Source:    dispatchplanports.EndMarketForecast,
			Cents:     30_000,
			DropCents: 42_000,
			StayCents: 12_000,
			Reason: forecastReason("Dallas, TX", demandforecast.CustomerShare{
				CustomerID: customerID,
				Name:       "Acme Foods",
				Expected:   1.75,
			}),
			StayReason: forecastReason("Waco, TX"),
		},
	})

	require.NotNil(t, candidate)
	assert.Equal(t, &gqlmodel.DispatchEndMarket{
		Source:          gqlmodel.DispatchEndMarketSourceForecast,
		ValueCents:      30_000,
		DropValueCents:  42_000,
		StayValueCents:  12_000,
		StayAreaLabel:   new("Waco, TX"),
		AreaLabel:       new("Dallas, TX"),
		WindowStart:     new(1_800_000_000),
		WindowEnd:       new(1_800_043_200),
		TypicalLow:      new(2),
		TypicalHigh:     new(6),
		Booked:          new(1),
		DriversExpected: new(3),
		Customers: []*gqlmodel.DispatchForecastCustomer{
			{CustomerID: customerID.String(), Name: "Acme Foods", ExpectedLoads: 1.75},
		},
	}, candidate.EndMarket)
}

func TestMapDispatchEndMarket_NamesTheStayAreaOnlyWhenStayingIsWorthSomething(t *testing.T) {
	t.Parallel()

	mapped := mapDispatchEndMarket(&dispatchplanports.EndMarket{
		Source:     dispatchplanports.EndMarketForecast,
		Cents:      42_000,
		DropCents:  42_000,
		Reason:     forecastReason("Dallas, TX"),
		StayReason: forecastReason("Waco, TX"),
	})

	require.NotNil(t, mapped)
	assert.Nil(t, mapped.StayAreaLabel)
	assert.Equal(t, new("Dallas, TX"), mapped.AreaLabel)
	assert.NotNil(t, mapped.Customers)
	assert.Empty(t, mapped.Customers)
}

func TestMapDispatchEndMarket_AHistoryValueHasNoForecastDetail(t *testing.T) {
	t.Parallel()

	mapped := mapDispatchEndMarket(&dispatchplanports.EndMarket{
		Source:    dispatchplanports.EndMarketHistory,
		Cents:     8_500,
		DropCents: 8_500,
	})

	assert.Equal(t, &gqlmodel.DispatchEndMarket{
		Source:         gqlmodel.DispatchEndMarketSourceHistory,
		ValueCents:     8_500,
		DropValueCents: 8_500,
		Customers:      []*gqlmodel.DispatchForecastCustomer{},
	}, mapped)
}

func TestMapDispatchEndMarket_NoValueMapsToNull(t *testing.T) {
	t.Parallel()

	assert.Nil(t, mapDispatchEndMarket(nil))
	candidate := mapDispatchCandidate(&dispatchplanports.CandidateScore{
		WorkerID: pulid.MustNew("wrk_"),
		MoveID:   pulid.MustNew("smv_"),
	})
	require.NotNil(t, candidate)
	assert.Nil(t, candidate.EndMarket)
}

func TestMapDispatchForecastQuality_MapsEveryStatus(t *testing.T) {
	t.Parallel()

	cases := []struct {
		status demandforecast.Status
		want   gqlmodel.DispatchForecastStatus
	}{
		{
			status: demandforecast.StatusInsufficientHistory,
			want:   gqlmodel.DispatchForecastStatusInsufficientHistory,
		},
		{status: demandforecast.StatusNoSkill, want: gqlmodel.DispatchForecastStatusNoSkill},
		{status: demandforecast.StatusReady, want: gqlmodel.DispatchForecastStatusReady},
	}

	for _, tc := range cases {
		t.Run(string(tc.status), func(t *testing.T) {
			t.Parallel()

			mapped := mapDispatchForecastQuality(&demandforecast.Quality{Status: tc.status}, false)
			assert.Equal(t, tc.want, mapped.Status)
		})
	}
}

func TestMapDispatchForecastQuality_CarriesTheBacktest(t *testing.T) {
	t.Parallel()

	mapped := mapDispatchForecastQuality(&demandforecast.Quality{
		Status:           demandforecast.StatusReady,
		HistoryWeeks:     12,
		HistoryLoads:     1_480,
		WeeksTested:      8,
		LoadsTested:      960,
		IntervalCoverage: 0.82,
		MAE:              1.4,
		NaiveMAE:         2.1,
		Skill:            0.33,
		FittedAt:         1_800_000_000,
	}, true)

	assert.Equal(t, &gqlmodel.DispatchForecastQuality{
		Status:                 gqlmodel.DispatchForecastStatusReady,
		HistoryWeeks:           12,
		HistoryLoads:           1_480,
		WeeksTested:            8,
		LoadsTested:            960,
		IntervalCoverage:       0.82,
		MeanAbsoluteError:      1.4,
		NaiveMeanAbsoluteError: 2.1,
		Skill:                  0.33,
		FittedAt:               new(1_800_000_000),
		Refitting:              true,
	}, mapped)
}

func TestMapDispatchForecastQuality_NeverFittedHasNoFittedAt(t *testing.T) {
	t.Parallel()

	mapped := mapDispatchForecastQuality(&demandforecast.Quality{
		Status: demandforecast.StatusInsufficientHistory,
	}, false)

	assert.Nil(t, mapped.FittedAt)
	assert.False(t, mapped.Refitting)
	assert.Equal(t, gqlmodel.DispatchForecastStatusInsufficientHistory, mapped.Status)
}

func TestMapDispatchFreightOutlook_MapsEverySection(t *testing.T) {
	t.Parallel()

	worker := pulid.MustNew("wrk_")
	partner := pulid.MustNew("wrk_")
	move := pulid.MustNew("sm_")
	customer := pulid.MustNew("cus_")
	positioning := pulid.MustNew("dpo_")
	mapped := mapDispatchFreightOutlook(&freightoutlookports.Outlook{
		Quality:     &demandforecast.Quality{Status: demandforecast.StatusReady, FittedAt: 1},
		GeneratedAt: 1_800_000_000,
		Starts:      []int64{1_800_000_000, 1_800_014_400},
		Network: []demandforecast.NetworkSlot{{
			Start: 1_800_000_000, End: 1_800_018_000,
			Low: 20, High: 30, NeedLow: 9, NeedHigh: 14, Need: 25, Drivers: 24,
			Short: 1, Uncovered: 6, Idle: 4,
		}},
		Areas: []*freightoutlookports.Area{{
			Balance: demandforecast.AreaBalance{
				Cell:  12971,
				Label: "Chicago, IL",
				Slots: []demandforecast.SlotBalance{{
					Start: 1_800_000_000, End: 1_800_018_000,
					Low: 2, High: 5, Booked: 2, Need: 3.5,
					Drivers: 1, Short: 2.5,
				}},
			},
			Point:      dispatchmarket.Point{Latitude: 41.9, Longitude: -87.6},
			ValueCents: 61_000,
			Customers: []freightoutlookports.AreaCustomer{
				{CustomerID: customer, Name: "Patriot Brewing", Share: 0.4},
			},
		}},
		AreaCount: 9,
		Steps: []freightmove.Step{
			{
				Kind: freightmove.KindKeep, WorkerID: worker, WorkerName: "Dana Ortiz",
				ToCell: 12971, ToLabel: "Chicago, IL",
				SlotStart: 1_800_014_400, SlotEnd: 1_800_025_200,
				NetCents: 54_000, MoveID: move, ProNumber: "PRO-1",
				PartnerID: partner, PartnerName: "Joe Kowalski", PartnerCell: 12802,
				PartnerFreeAt: 1_800_003_600,
			},
			{
				Kind: freightmove.KindCall, ToCell: 12971, ToLabel: "Chicago, IL",
				SlotStart: 1_800_014_400, CustomerID: customer, CustomerName: "Patriot Brewing",
				Usual: 5, ByNow: 3, Tendered: 1, Loads: 4,
			},
		},
		Days: []*demandforecast.Day{
			{DayStart: 1_799_913_600, Low: 80, High: 104, Mean: 92, Actual: 96},
		},
		Positionings: []*freightoutlookports.Positioning{{
			Positioning: &driverpositioning.Positioning{
				ID: positioning, WorkerID: worker, ToCell: 12971, ToLabel: "Chicago, IL",
				Status: driverpositioning.StatusActive, ArriveAt: 1_800_007_200,
			},
			WorkerName: "Dana Ortiz",
		}},
	})

	assert.Equal(t, []int{1_800_000_000, 1_800_014_400}, mapped.SlotStarts)
	assert.Equal(t, 9, mapped.AreaCount)
	assert.Equal(t, []*gqlmodel.DispatchFreightNetworkSlot{{
		Start: 1_800_000_000, End: 1_800_018_000,
		LikelyLow: 20, LikelyHigh: 30, NeedLow: 9, NeedHigh: 14,
		Need: 25, Drivers: 24,
		Short: 1, Uncovered: 6, Idle: 4,
	}}, mapped.Network)
	require.Len(t, mapped.Areas, 1)
	area := mapped.Areas[0]
	assert.Equal(t, 12971, area.Cell)
	assert.Equal(t, 61_000, area.ValueCents)
	assert.InDelta(t, 41.9, area.Latitude, 1e-9)
	assert.Equal(t, []*gqlmodel.DispatchFreightSlot{{
		Start: 1_800_000_000, End: 1_800_018_000,
		LikelyLow: 2, LikelyHigh: 5, Booked: 2, Need: 3.5,
		Drivers: 1, Short: 2.5,
	}}, area.Slots)
	assert.Equal(t, "Patriot Brewing", area.Customers[0].Name)

	require.Len(t, mapped.Moves, 2)
	keep := mapped.Moves[0]
	assert.Equal(t, 1_800_014_400, keep.SlotStart)
	assert.Equal(t, 1_800_025_200, keep.SlotEnd,
		"a window's end is sent as it is, three hours on here, never assumed")
	assert.Equal(t, gqlmodel.DispatchFreightMoveKindKeep, keep.Kind)
	assert.Equal(t, worker.String(), *keep.WorkerID)
	assert.Equal(t, move.String(), *keep.MoveID)
	assert.Equal(t, partner.String(), *keep.PartnerID)
	assert.Equal(t, 1_800_003_600, *keep.PartnerFreeAt)
	assert.Equal(t, 12802, keep.PartnerCell, "a keep is drawn from where its partner frees up")
	assert.Nil(t, keep.CustomerID, "a keep names no customer")
	assert.Nil(t, keep.FreeAt, "an unset time maps to null, not zero")
	call := mapped.Moves[1]
	assert.Equal(t, gqlmodel.DispatchFreightMoveKindCall, call.Kind)
	assert.Nil(t, call.WorkerID, "a call names no driver")
	assert.Equal(t, customer.String(), *call.CustomerID)
	assert.Equal(t, 1, call.Tendered)
	assert.InDelta(t, 4, call.Loads, 1e-9)
	assert.Equal(t, []string{}, call.LaneMoveIds)

	assert.Equal(t, []*gqlmodel.DispatchForecastDay{{
		DayStart: 1_799_913_600, Low: 80, High: 104, Mean: 92, Actual: 96,
	}}, mapped.Record)
	require.Len(t, mapped.Positionings, 1)
	assert.Equal(t, positioning.String(), mapped.Positionings[0].ID)
	assert.Equal(t, "Dana Ortiz", mapped.Positionings[0].WorkerName)
	assert.Equal(t, gqlmodel.DispatchPositioningStatusActive, mapped.Positionings[0].Status)
}

func TestMapDispatchFreightOutlook_AnUnreadyForecastHasEmptySections(t *testing.T) {
	t.Parallel()

	mapped := mapDispatchFreightOutlook(&freightoutlookports.Outlook{
		Quality: &demandforecast.Quality{Status: demandforecast.StatusNoSkill},
	})

	assert.Equal(t, gqlmodel.DispatchForecastStatusNoSkill, mapped.Quality.Status)
	assert.NotNil(t, mapped.Areas)
	assert.Empty(t, mapped.Areas)
	assert.NotNil(t, mapped.Moves)
	assert.NotNil(t, mapped.Positionings)
}

func TestMapDispatchFreightAreaDetail_MapsReachAndTheRecentWeeks(t *testing.T) {
	t.Parallel()

	worker := pulid.MustNew("wrk_")
	mapped := mapDispatchFreightAreaDetail(&freightoutlookports.AreaDetail{
		Quality: &demandforecast.Quality{Status: demandforecast.StatusReady},
		Area:    &freightoutlookports.Area{Balance: demandforecast.AreaBalance{Cell: 12971}},
		Bounds: dispatchmarket.Bounds{
			South: 39.63, West: -87.19, North: 40.98, East: -85.78,
		},
		SlotStart: 1_800_014_400,
		SlotEnd:   1_800_032_400,
		Reach: []freightmove.ReachRow{{
			WorkerID: worker, WorkerName: "Joe Kowalski", Label: "Indianapolis, IN",
			Point:  dispatchmarket.Point{Latitude: 39.8, Longitude: -86.2},
			FreeAt: 1_800_000_000, ArriveAt: 1_800_010_000, Miles: 318, NetCents: 37_100,
			Reach: freightmove.ReachOnTime,
		}},
		Recent:     []int32{5, 4, 6},
		RecentLow:  3,
		RecentHigh: 7,
	})

	assert.Equal(t, 1_800_014_400, mapped.SlotStart)
	assert.Equal(t, 1_800_032_400, mapped.SlotEnd)
	assert.Equal(t, &gqlmodel.DispatchFreightAreaBounds{
		South: 39.63, West: -87.19, North: 40.98, East: -85.78,
	}, mapped.Bounds)
	assert.Equal(t, []int{5, 4, 6}, mapped.Recent)
	assert.Equal(t, 3, mapped.RecentLow)
	assert.Equal(t, 7, mapped.RecentHigh)
	assert.Equal(t, []*gqlmodel.DispatchFreightReach{{
		WorkerID: worker.String(), WorkerName: "Joe Kowalski", Label: "Indianapolis, IN",
		Latitude: 39.8, Longitude: -86.2, FreeAt: 1_800_000_000, ArriveAt: 1_800_010_000,
		Miles: 318, NetCents: 37_100, Reach: gqlmodel.DispatchFreightReachKindOnTime,
	}}, mapped.Reach)
}
