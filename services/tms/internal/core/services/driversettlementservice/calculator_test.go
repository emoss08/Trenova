package driversettlementservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func decPtr(v float64) *float64 { return &v }

func i64Ptr(v int64) *int64 { return &v }

func testProfile(components ...*driverpay.PayProfileComponent) *driverpay.PayProfile {
	return &driverpay.PayProfile{
		Name:           "Test Profile",
		Classification: driverpay.PayeeClassificationCompanyDriver,
		CurrencyCode:   "USD",
		Components:     components,
	}
}

func loadedMove(distance float64, stops int) *shipment.ShipmentMove {
	move := &shipment.ShipmentMove{Loaded: true, Distance: decPtr(distance)}
	for range stops {
		move.Stops = append(move.Stops, &shipment.Stop{})
	}
	return move
}

func calcInput(
	profile *driverpay.PayProfile,
	move *shipment.ShipmentMove,
) *moveCalcInput {
	return &moveCalcInput{
		Profile:           profile,
		SplitPercent:      decimal.NewFromInt(100),
		Shipment:          &shipment.Shipment{},
		Move:              move,
		TotalTripDistance: moveDistance(move),
		MoveCount:         1,
	}
}

func TestComputeMovePayPerLoadedMile(t *testing.T) {
	profile := testProfile(&driverpay.PayProfileComponent{
		Kind:     driverpay.ComponentKindLinehaul,
		Method:   driverpay.CalcMethodPerLoadedMile,
		Rate:     decimal.RequireFromString("0.55"),
		IsActive: true,
	})
	components, gross := computeMovePay(calcInput(profile, loadedMove(500, 2)))

	require.Len(t, components, 1)
	assert.Equal(t, int64(27500), gross)
	assert.Equal(t, driverpay.CalcMethodPerLoadedMile, components[0].Method)
	assert.True(t, components[0].Quantity.Equal(decimal.NewFromInt(500)))
}

func TestComputeMovePayPerLoadedMileSkipsEmptyMove(t *testing.T) {
	profile := testProfile(&driverpay.PayProfileComponent{
		Kind:     driverpay.ComponentKindLinehaul,
		Method:   driverpay.CalcMethodPerLoadedMile,
		Rate:     decimal.RequireFromString("0.55"),
		IsActive: true,
	})
	move := loadedMove(500, 2)
	move.Loaded = false

	components, gross := computeMovePay(calcInput(profile, move))
	assert.Empty(t, components)
	assert.Zero(t, gross)
}

func TestComputeMovePayMileageBands(t *testing.T) {
	profile := testProfile(&driverpay.PayProfileComponent{
		Kind:   driverpay.ComponentKindLinehaul,
		Method: driverpay.CalcMethodPerLoadedMile,
		Rate:   decimal.RequireFromString("0.50"),
		Bands: []driverpay.MileageBand{
			{MinMiles: 0, MaxMiles: 200, Rate: decimal.RequireFromString("0.70")},
			{MinMiles: 200, MaxMiles: 500, Rate: decimal.RequireFromString("0.60")},
			{MinMiles: 500, MaxMiles: 0, Rate: decimal.RequireFromString("0.52")},
		},
		IsActive: true,
	})

	_, shortHaul := computeMovePay(calcInput(profile, loadedMove(150, 2)))
	assert.Equal(t, int64(10500), shortHaul)

	_, midHaul := computeMovePay(calcInput(profile, loadedMove(300, 2)))
	assert.Equal(t, int64(18000), midHaul)

	_, longHaul := computeMovePay(calcInput(profile, loadedMove(800, 2)))
	assert.Equal(t, int64(41600), longHaul)
}

func TestComputeMovePayPercentOfRevenue(t *testing.T) {
	profile := testProfile(&driverpay.PayProfileComponent{
		Kind:         driverpay.ComponentKindLinehaul,
		Method:       driverpay.CalcMethodPercentOfRevenue,
		Rate:         decimal.NewFromInt(25),
		RevenueBasis: driverpay.RevenueBasisLinehaul,
		IsActive:     true,
	})
	move := loadedMove(500, 2)
	input := calcInput(profile, move)
	input.Shipment = &shipment.Shipment{
		FreightChargeAmount: decimal.NewNullDecimal(decimal.NewFromInt(2000)),
	}

	_, gross := computeMovePay(input)
	assert.Equal(t, int64(50000), gross)
}

func TestComputeMovePayPercentAllocatesAcrossMoves(t *testing.T) {
	profile := testProfile(&driverpay.PayProfileComponent{
		Kind:         driverpay.ComponentKindLinehaul,
		Method:       driverpay.CalcMethodPercentOfRevenue,
		Rate:         decimal.NewFromInt(25),
		RevenueBasis: driverpay.RevenueBasisLinehaul,
		IsActive:     true,
	})
	move := loadedMove(300, 2)
	input := calcInput(profile, move)
	input.TotalTripDistance = decimal.NewFromInt(600)
	input.MoveCount = 2
	input.Shipment = &shipment.Shipment{
		FreightChargeAmount: decimal.NewNullDecimal(decimal.NewFromInt(2000)),
	}

	_, gross := computeMovePay(input)
	assert.Equal(t, int64(25000), gross)
}

func TestComputeMovePayTeamSplit(t *testing.T) {
	profile := testProfile(&driverpay.PayProfileComponent{
		Kind:     driverpay.ComponentKindLinehaul,
		Method:   driverpay.CalcMethodPerLoadedMile,
		Rate:     decimal.RequireFromString("0.60"),
		IsActive: true,
	})
	input := calcInput(profile, loadedMove(1000, 2))
	input.SplitPercent = decimal.NewFromInt(50)

	_, gross := computeMovePay(input)
	assert.Equal(t, int64(30000), gross)
}

func TestComputeMovePayStopPay(t *testing.T) {
	profile := testProfile(&driverpay.PayProfileComponent{
		Kind:     driverpay.ComponentKindStopPay,
		Method:   driverpay.CalcMethodPerStop,
		Rate:     decimal.NewFromInt(50),
		IsActive: true,
	})

	_, twoStops := computeMovePay(calcInput(profile, loadedMove(100, 2)))
	assert.Zero(t, twoStops)

	_, fiveStops := computeMovePay(calcInput(profile, loadedMove(100, 5)))
	assert.Equal(t, int64(15000), fiveStops)
}

func TestComputeMovePayDetention(t *testing.T) {
	profile := testProfile(&driverpay.PayProfileComponent{
		Kind:            driverpay.ComponentKindDetention,
		Method:          driverpay.CalcMethodPerHour,
		Rate:            decimal.NewFromInt(40),
		FreeTimeMinutes: 120,
		IsActive:        true,
	})
	move := loadedMove(100, 0)
	arrival := int64(1000)
	departure := arrival + 4*3600
	move.Stops = []*shipment.Stop{
		{ActualArrival: &arrival, ActualDeparture: &departure},
	}

	components, gross := computeMovePay(calcInput(profile, move))
	require.Len(t, components, 1)
	assert.Equal(t, int64(8000), gross)
}

func TestComputeMovePayDetentionRespectsOverride(t *testing.T) {
	profile := testProfile(&driverpay.PayProfileComponent{
		Kind:            driverpay.ComponentKindDetention,
		Method:          driverpay.CalcMethodPerHour,
		Rate:            decimal.NewFromInt(40),
		FreeTimeMinutes: 120,
		IsActive:        true,
	})
	move := loadedMove(100, 0)
	arrival := int64(1000)
	departure := arrival + 4*3600
	skip := false
	move.Stops = []*shipment.Stop{
		{
			ActualArrival:          &arrival,
			ActualDeparture:        &departure,
			CountDetentionOverride: &skip,
		},
	}

	_, gross := computeMovePay(calcInput(profile, move))
	assert.Zero(t, gross)
}

func TestComputeMovePayHazmatPremium(t *testing.T) {
	profile := testProfile(&driverpay.PayProfileComponent{
		Kind:     driverpay.ComponentKindHazmat,
		Method:   driverpay.CalcMethodPerEvent,
		Rate:     decimal.NewFromInt(75),
		IsActive: true,
	})

	input := calcInput(profile, loadedMove(100, 2))
	_, withoutHazmat := computeMovePay(input)
	assert.Zero(t, withoutHazmat)

	input.HasHazmat = true
	_, withHazmat := computeMovePay(input)
	assert.Equal(t, int64(7500), withHazmat)
}

func TestComputeMovePayComponentCaps(t *testing.T) {
	profile := testProfile(&driverpay.PayProfileComponent{
		Kind:           driverpay.ComponentKindLinehaul,
		Method:         driverpay.CalcMethodPerLoadedMile,
		Rate:           decimal.RequireFromString("0.50"),
		MinAmountMinor: i64Ptr(10000),
		MaxAmountMinor: i64Ptr(50000),
		IsActive:       true,
	})

	_, floored := computeMovePay(calcInput(profile, loadedMove(50, 2)))
	assert.Equal(t, int64(10000), floored)

	_, capped := computeMovePay(calcInput(profile, loadedMove(2000, 2)))
	assert.Equal(t, int64(50000), capped)
}

func TestComputeMovePaySkipsInactiveComponents(t *testing.T) {
	profile := testProfile(&driverpay.PayProfileComponent{
		Kind:     driverpay.ComponentKindLinehaul,
		Method:   driverpay.CalcMethodPerLoadedMile,
		Rate:     decimal.RequireFromString("0.55"),
		IsActive: false,
	})

	components, gross := computeMovePay(calcInput(profile, loadedMove(500, 2)))
	assert.Empty(t, components)
	assert.Zero(t, gross)
}

func TestComputeMovePayFuelSurchargePassThrough(t *testing.T) {
	profile := testProfile(&driverpay.PayProfileComponent{
		Kind:         driverpay.ComponentKindFuelSurcharge,
		Method:       driverpay.CalcMethodPercentOfRevenue,
		Rate:         decimal.NewFromInt(100),
		RevenueBasis: driverpay.RevenueBasisLinehaul,
		IsActive:     true,
	})
	input := calcInput(profile, loadedMove(500, 2))
	input.FuelSurcharge = decimal.NewFromInt(150)

	_, gross := computeMovePay(input)
	assert.Equal(t, int64(15000), gross)
}

func TestComputeMovePayRateOverrides(t *testing.T) {
	componentID := pulid.MustNew("dppc_")
	profile := testProfile(&driverpay.PayProfileComponent{
		ID:     componentID,
		Kind:   driverpay.ComponentKindLinehaul,
		Method: driverpay.CalcMethodPerLoadedMile,
		Rate:   decimal.RequireFromString("0.50"),
		Bands: []driverpay.MileageBand{
			{MinMiles: 0, MaxMiles: 0, Rate: decimal.RequireFromString("0.70")},
		},
		IsActive: true,
	})
	input := calcInput(profile, loadedMove(1000, 2))
	input.RateOverrides = []driverpay.RateOverride{
		{ComponentID: componentID, Rate: decimal.RequireFromString("0.62")},
	}

	components, gross := computeMovePay(input)
	require.Len(t, components, 1)
	assert.Equal(t, int64(62000), gross)
	assert.True(t, components[0].Rate.Equal(decimal.RequireFromString("0.62")))
}

func TestComputeMovePayOverrideIgnoresOtherComponents(t *testing.T) {
	linehaulID := pulid.MustNew("dppc_")
	stopPayID := pulid.MustNew("dppc_")
	profile := testProfile(
		&driverpay.PayProfileComponent{
			ID:       linehaulID,
			Kind:     driverpay.ComponentKindLinehaul,
			Method:   driverpay.CalcMethodPerLoadedMile,
			Rate:     decimal.RequireFromString("0.50"),
			IsActive: true,
		},
		&driverpay.PayProfileComponent{
			ID:       stopPayID,
			Kind:     driverpay.ComponentKindStopPay,
			Method:   driverpay.CalcMethodPerStop,
			Rate:     decimal.NewFromInt(50),
			IsActive: true,
		},
	)
	input := calcInput(profile, loadedMove(100, 4))
	input.RateOverrides = []driverpay.RateOverride{
		{ComponentID: linehaulID, Rate: decimal.RequireFromString("0.60")},
	}

	_, gross := computeMovePay(input)
	assert.Equal(t, int64(6000+10000), gross)
}
