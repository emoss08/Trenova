package dispatchconsoleresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/ports/demandforecastports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
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

			mapped := mapDispatchForecastQuality(&demandforecast.Quality{Status: tc.status})
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
	})

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
	}, mapped)
}

func TestMapDispatchForecastQuality_NeverFittedHasNoFittedAt(t *testing.T) {
	t.Parallel()

	mapped := mapDispatchForecastQuality(&demandforecast.Quality{
		Status: demandforecast.StatusInsufficientHistory,
	})

	assert.Nil(t, mapped.FittedAt)
	assert.Equal(t, gqlmodel.DispatchForecastStatusInsufficientHistory, mapped.Status)
}

func TestMapDispatchFreightOutlook_MapsEachWindow(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	mapped := mapDispatchFreightOutlook(&demandforecastports.Outlook{
		Quality:     &demandforecast.Quality{Status: demandforecast.StatusReady, FittedAt: 1},
		GeneratedAt: 1_800_000_100,
		Windows: []*demandforecastports.OutlookWindow{{
			Cell:             7,
			AreaLabel:        "Dallas, TX",
			WindowStart:      1_800_000_000,
			WindowEnd:        1_800_014_400,
			TypicalLow:       3,
			TypicalHigh:      9,
			Booked:           2,
			ExpectedUnbooked: 4.25,
			DriversExpected:  1,
			Shortfall:        5.25,
			ValueCents:       61_000,
			Customers: []demandforecast.CustomerShare{
				{CustomerID: customerID, Name: "Acme Foods", Expected: 3.5},
			},
			Explanation: "Ending near Dallas, TX is worth $610.00.",
		}},
	})

	require.NotNil(t, mapped)
	assert.Equal(t, gqlmodel.DispatchForecastStatusReady, mapped.Quality.Status)
	assert.Equal(t, 1_800_000_100, mapped.GeneratedAt)
	assert.Equal(t, []*gqlmodel.DispatchFreightWindow{{
		AreaLabel:        "Dallas, TX",
		WindowStart:      1_800_000_000,
		WindowEnd:        1_800_014_400,
		TypicalLow:       3,
		TypicalHigh:      9,
		Booked:           2,
		ExpectedUnbooked: 4.25,
		DriversExpected:  1,
		Shortfall:        5.25,
		ValueCents:       61_000,
		Customers: []*gqlmodel.DispatchForecastCustomer{
			{CustomerID: customerID.String(), Name: "Acme Foods", ExpectedLoads: 3.5},
		},
		Explanation: "Ending near Dallas, TX is worth $610.00.",
	}}, mapped.Windows)
}

func TestMapDispatchFreightOutlook_AnUnreadyForecastHasNoWindows(t *testing.T) {
	t.Parallel()

	mapped := mapDispatchFreightOutlook(&demandforecastports.Outlook{
		Quality: &demandforecast.Quality{Status: demandforecast.StatusNoSkill},
		Windows: []*demandforecastports.OutlookWindow{},
	})

	assert.Equal(t, gqlmodel.DispatchForecastStatusNoSkill, mapped.Quality.Status)
	assert.NotNil(t, mapped.Windows)
	assert.Empty(t, mapped.Windows)
}
