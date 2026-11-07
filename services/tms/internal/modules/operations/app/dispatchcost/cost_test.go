package dispatchcost_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchcost"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchurgency"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatcheconomics"
	"github.com/stretchr/testify/assert"
)

func plainEconomics() dispatcheconomics.Economics {
	return dispatcheconomics.Economics{
		EmptyMileCents:     200,
		LateMinuteCents:    100,
		DriverHourCents:    3_000,
		EarlyGraceMinutes:  120,
		HomeTimeTargetDays: 10,
		HomeTimeDayCents:   7_500,
		FairnessMileCents:  3,
		Preferences: map[dispatcheconomics.Preference]int64{
			dispatcheconomics.PreferenceCustomerExperience: 0,
			dispatcheconomics.PreferenceOnTimeHistory:      0,
			dispatcheconomics.PreferenceSafety:             0,
			dispatcheconomics.PreferenceAcceptance:         0,
			dispatcheconomics.PreferenceTrailerContinuity:  0,
			dispatcheconomics.PreferenceHaulFit:            0,
			dispatcheconomics.PreferenceTimeOffBuffer:      0,
		},
		CoverageMarginPercent: 30,
		CoverageFloorCents:    40_000,
		LatePremiumCents:      150_000,
		DueNowPremiumCents:    75_000,
		CandidatesPerMove:     40,
	}
}

func TestModel_PricesTheHardCostsOfAPairingInCents(t *testing.T) {
	t.Parallel()

	model := modelOf(plainEconomics(), nil)

	cost := model.Explain(&dispatchcost.Features{
		EmptyMiles:          50,
		PickupLateMinutes:   10,
		DeliveryLateMinutes: 5,
		EarlyMinutes:        180,
		RestHours:           2,
		HomeOverdueDays:     2,
		HomewardShare:       0.5,
		WeekMilesOverMedian: 400,
	})

	assert.Equal(t, []dispatcheconomics.Line{
		{Component: dispatcheconomics.ComponentEmptyMiles, Cents: 10_000},
		{Component: dispatcheconomics.ComponentLateness, Cents: 1_500},
		{Component: dispatcheconomics.ComponentIdleWait, Cents: 3_000},
		{Component: dispatcheconomics.ComponentRest, Cents: 6_000},
		{Component: dispatcheconomics.ComponentHomeTime, Cents: 7_500},
		{Component: dispatcheconomics.ComponentFairness, Cents: 1_200},
	}, cost.Lines)
	assert.Equal(t, int64(29_200), cost.TotalCents)
	assert.Equal(t, cost.TotalCents, model.Total(&dispatchcost.Features{
		EmptyMiles:          50,
		PickupLateMinutes:   10,
		DeliveryLateMinutes: 5,
		EarlyMinutes:        180,
		RestHours:           2,
		HomeOverdueDays:     2,
		HomewardShare:       0.5,
		WeekMilesOverMedian: 400,
	}), "the solver's numeric path and the explained cost agree")
}

func TestModel_PricesMissingHistoryAsNeutralNotAsPerfect(t *testing.T) {
	t.Parallel()

	economics := plainEconomics()
	economics.Preferences[dispatcheconomics.PreferenceOnTimeHistory] = 10_000
	model := modelOf(economics, nil)

	unknown := model.Total(&dispatchcost.Features{})
	perfect := model.Total(&dispatchcost.Features{OnTimeHistory: dispatchcost.Known(1)})
	poor := model.Total(&dispatchcost.Features{OnTimeHistory: dispatchcost.Known(0)})

	assert.Equal(t, int64(5_000), unknown)
	assert.Equal(t, int64(0), perfect)
	assert.Equal(t, int64(10_000), poor)
}

func TestModel_ScalesAPreferenceByTheOrganizationsLearnedWeight(t *testing.T) {
	t.Parallel()

	economics := plainEconomics()
	economics.Preferences[dispatcheconomics.PreferenceTrailerContinuity] = 10_000
	learned := dispatchcost.Multipliers{dispatcheconomics.ComponentTrailerContinuity: 1.5}

	cost := modelOf(economics, learned).Total(&dispatchcost.Features{
		TrailerContinuity: dispatchcost.Known(0),
	})

	assert.Equal(t, int64(15_000), cost)
}

func TestModel_BoundsALearnedWeightSoOneSignalCannotDominate(t *testing.T) {
	t.Parallel()

	economics := plainEconomics()
	economics.Preferences[dispatcheconomics.PreferenceSafety] = 1_000
	learned := dispatchcost.Multipliers{dispatcheconomics.ComponentSafety: 50}

	cost := modelOf(economics, learned).Total(&dispatchcost.Features{
		Safety: dispatchcost.Known(0),
	})

	assert.Equal(t, int64(4_000), cost)
}

func TestModel_ScalesHomeTimeAndFairnessByTheLearnedWeight(t *testing.T) {
	t.Parallel()

	learned := dispatchcost.Multipliers{
		dispatcheconomics.ComponentHomeTime: 1.6,
		dispatcheconomics.ComponentFairness: 0.5,
	}

	cost := modelOf(plainEconomics(), learned).Explain(&dispatchcost.Features{
		HomeOverdueDays:     2,
		WeekMilesOverMedian: 100,
	})

	assert.Contains(t, cost.Lines, dispatcheconomics.Line{
		Component: dispatcheconomics.ComponentHomeTime,
		Cents:     24_000,
	})
	assert.Contains(t, cost.Lines, dispatcheconomics.Line{
		Component: dispatcheconomics.ComponentFairness,
		Cents:     150,
	})
}

func TestModel_NeverScalesAHardCost(t *testing.T) {
	t.Parallel()

	learned := dispatchcost.Multipliers{
		dispatcheconomics.ComponentEmptyMiles: 4,
		dispatcheconomics.ComponentEndMarket:  4,
	}

	cost := modelOf(plainEconomics(), learned).Total(&dispatchcost.Features{
		EmptyMiles:     10,
		EndMarketCents: 500,
	})

	assert.Equal(t, int64(1_500), cost)
}

func TestModel_CreditsAMoveThatEndsWhereDriversAreWorthMore(t *testing.T) {
	t.Parallel()

	cost := modelOf(plainEconomics(), nil).Explain(&dispatchcost.Features{
		EmptyMiles:     10,
		EndMarketCents: 1_500,
	})

	assert.Equal(t, int64(500), cost.TotalCents)
	assert.Contains(t, cost.Lines, dispatcheconomics.Line{
		Component: dispatcheconomics.ComponentEndMarket,
		Cents:     -1_500,
	})
}

func TestCoveragePrice_IsAShareOfRevenueAboveAFloorPlusAnUrgencyPremium(t *testing.T) {
	t.Parallel()

	model := modelOf(plainEconomics(), nil)

	assert.Equal(t, int64(40_000), model.CoveragePrice(dispatchcost.MoveValue{
		RevenueCents: 50_000,
		Urgency:      dispatchurgency.Planned,
	}), "a small load is still worth the floor")
	assert.Equal(t, int64(90_000), model.CoveragePrice(dispatchcost.MoveValue{
		RevenueCents: 300_000,
		Urgency:      dispatchurgency.Today,
	}))
	assert.Equal(t, int64(240_000), model.CoveragePrice(dispatchcost.MoveValue{
		RevenueCents: 300_000,
		Urgency:      dispatchurgency.Late,
	}))
	assert.Equal(t, int64(165_000), model.CoveragePrice(dispatchcost.MoveValue{
		RevenueCents: 300_000,
		Urgency:      dispatchurgency.Now,
	}))
}

func TestNewModel_FallsBackToDefaultsForAnUnsetOrganization(t *testing.T) {
	t.Parallel()

	model := modelOf(dispatcheconomics.Economics{}, nil)

	assert.Equal(t, dispatcheconomics.Default(), model.Economics())
}

func TestModel_SplitsHardCostsThatDecideCoverageFromSoftOnesThatChooseADriver(t *testing.T) {
	t.Parallel()

	economics := plainEconomics()
	economics.Preferences[dispatcheconomics.PreferenceHaulFit] = 6_000
	model := modelOf(economics, nil)
	features := &dispatchcost.Features{
		EmptyMiles:          10,
		PickupLateMinutes:   5,
		WeekMilesOverMedian: 100,
		HaulFit:             dispatchcost.Known(0.5),
	}

	hard, soft := model.Split(features)

	assert.Equal(t, int64(2_500), hard)
	assert.Equal(t, int64(3_300), soft)
	assert.Equal(t, model.Total(features), hard+soft)
}

func modelOf(
	economics dispatcheconomics.Economics,
	multipliers dispatchcost.Multipliers,
) *dispatchcost.Model {
	model := dispatchcost.NewModel(&economics, multipliers)
	return &model
}
