package costingservice_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/costingcontrol"
	"github.com/emoss08/trenova/internal/core/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/costingservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	testOrgID       = pulid.MustNew("org_")
	testBuID        = pulid.MustNew("bu_")
	testFuelIndexID = pulid.MustNew("fidx_")
	testNow         = time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
)

func testTenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: testOrgID, BuID: testBuID}
}

func benchmarkControl() *costingcontrol.CostingControl {
	control := &costingcontrol.CostingControl{
		ID:                   pulid.MustNew("cstc_"),
		BusinessUnitID:       testBuID,
		OrganizationID:       testOrgID,
		UseLiveFuelPrice:     false,
		MilesPerGallon:       costingcontrol.DefaultMilesPerGallon(),
		IncludeDeadheadMiles: true,
		GLRollingMonths:      3,
	}
	control.Categories = costingcontrol.DefaultCategories()
	for _, category := range control.Categories {
		category.BusinessUnitID = testBuID
		category.OrganizationID = testOrgID
		category.CostingControlID = control.ID
	}
	return control
}

func float64Ptr(v float64) *float64 {
	return &v
}

func testShipment(moves []*shipment.ShipmentMove, revenue string) *shipment.Shipment {
	return &shipment.Shipment{
		ID:                pulid.MustNew("shp_"),
		BusinessUnitID:    testBuID,
		OrganizationID:    testOrgID,
		Moves:             moves,
		TotalChargeAmount: decimal.NewNullDecimal(decimal.RequireFromString(revenue)),
	}
}

func newEstimatorService(
	t *testing.T,
	control *costingcontrol.CostingControl,
	entity *shipment.Shipment,
) *costingservice.Service {
	t.Helper()

	controlRepo := mocks.NewMockCostingControlRepository(t)
	controlRepo.EXPECT().
		GetByOrgID(mock.Anything, mock.Anything).
		Return(control, nil).
		Maybe()

	shipmentRepo := mocks.NewMockShipmentRepository(t)
	if entity != nil {
		shipmentRepo.EXPECT().
			GetByID(mock.Anything, mock.Anything).
			Return(entity, nil).
			Once()
	}

	return costingservice.NewTestService(costingservice.TestServiceParams{
		Repo:         controlRepo,
		ShipmentRepo: shipmentRepo,
		Now:          func() time.Time { return testNow },
	})
}

func TestEstimateShipment_BenchmarkProfile(t *testing.T) {
	t.Parallel()

	entity := testShipment([]*shipment.ShipmentMove{
		{Loaded: true, Distance: float64Ptr(500)},
		{Loaded: false, Distance: float64Ptr(100)},
	}, "2000")

	svc := newEstimatorService(t, benchmarkControl(), entity)

	estimate, err := svc.EstimateShipment(t.Context(), entity.ID, testTenantInfo())
	require.NoError(t, err)

	assert.InDelta(t, 500.0, estimate.LoadedMiles, 0.001)
	assert.InDelta(t, 100.0, estimate.DeadheadMiles, 0.001)
	assert.InDelta(t, 600.0, estimate.TotalMiles, 0.001)
	assert.False(t, estimate.MissingDistance)

	assert.True(
		t,
		estimate.Profile.TotalCPM.Equal(decimal.RequireFromString("2.435")),
		"total CPM should be ATRI benchmark sum, got %s", estimate.Profile.TotalCPM,
	)
	assert.True(
		t,
		estimate.EstimatedCost.Equal(decimal.RequireFromString("1461")),
		"cost = 2.435 x 600 = 1461, got %s", estimate.EstimatedCost,
	)
	assert.True(t, estimate.Profit.Equal(decimal.RequireFromString("539")))

	require.True(t, estimate.MarginPercent.Valid)
	assert.True(
		t,
		estimate.MarginPercent.Decimal.Equal(decimal.RequireFromString("26.95")),
		"margin should be 26.95, got %s", estimate.MarginPercent.Decimal,
	)

	require.True(t, estimate.BreakEvenRPM.Valid)
	assert.True(
		t,
		estimate.BreakEvenRPM.Decimal.Equal(decimal.RequireFromString("2.922")),
		"break-even RPM = 1461 / 500, got %s", estimate.BreakEvenRPM.Decimal,
	)

	require.True(t, estimate.RevenuePerLoadedMile.Valid)
	assert.True(t, estimate.RevenuePerLoadedMile.Decimal.Equal(decimal.RequireFromString("4")))

	require.Len(t, estimate.Breakdown, 10)
	for _, line := range estimate.Breakdown {
		assert.Equal(t, costingcontrol.EffectiveRateSourceBenchmark, line.EffectiveSource)
	}
}

func TestEstimateShipment_ExcludeDeadheadMiles(t *testing.T) {
	t.Parallel()

	control := benchmarkControl()
	control.IncludeDeadheadMiles = false

	entity := testShipment([]*shipment.ShipmentMove{
		{Loaded: true, Distance: float64Ptr(500)},
		{Loaded: false, Distance: float64Ptr(100)},
	}, "2000")

	svc := newEstimatorService(t, control, entity)

	estimate, err := svc.EstimateShipment(t.Context(), entity.ID, testTenantInfo())
	require.NoError(t, err)

	assert.True(
		t,
		estimate.EstimatedCost.Equal(decimal.RequireFromString("1217.5")),
		"cost = 2.435 x 500 loaded miles only, got %s", estimate.EstimatedCost,
	)
}

func TestEstimateShipment_LiveFuelPrice(t *testing.T) {
	t.Parallel()

	control := benchmarkControl()
	control.UseLiveFuelPrice = true
	control.FuelIndexID = &testFuelIndexID

	entity := testShipment([]*shipment.ShipmentMove{
		{Loaded: true, Distance: float64Ptr(100)},
	}, "1000")

	controlRepo := mocks.NewMockCostingControlRepository(t)
	controlRepo.EXPECT().GetByOrgID(mock.Anything, mock.Anything).Return(control, nil)

	shipmentRepo := mocks.NewMockShipmentRepository(t)
	shipmentRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(entity, nil)

	priceRepo := mocks.NewMockFuelIndexPriceRepository(t)
	priceRepo.EXPECT().
		GetLatestOnOrBefore(mock.Anything, mock.MatchedBy(func(req *repositories.GetLatestFuelPricesRequest) bool {
			return req.FuelIndexID == testFuelIndexID && req.Date == "2026-07-20" && req.Limit == 1
		})).
		Return([]*fuelsurcharge.FuelIndexPrice{
			{Price: decimal.RequireFromString("3.90"), PriceDate: "2026-07-14"},
		}, nil)

	svc := costingservice.NewTestService(costingservice.TestServiceParams{
		Repo:         controlRepo,
		ShipmentRepo: shipmentRepo,
		PriceRepo:    priceRepo,
		Now:          func() time.Time { return testNow },
	})

	estimate, err := svc.EstimateShipment(t.Context(), entity.ID, testTenantInfo())
	require.NoError(t, err)

	require.NotNil(t, estimate.Profile.Fuel)
	assert.Equal(
		t,
		costingcontrol.EffectiveRateSourceLiveIndex,
		estimate.Profile.Fuel.Source,
	)
	assert.Equal(t, "2026-07-14", estimate.Profile.Fuel.PriceDate)

	// Fuel rate becomes 3.90 / 6.5 = 0.60 instead of the 0.48 benchmark.
	assert.True(
		t,
		estimate.Profile.TotalCPM.Equal(decimal.RequireFromString("2.555")),
		"total CPM should swap fuel benchmark for live rate, got %s", estimate.Profile.TotalCPM,
	)

	var fuelLine *costingservice.CategoryCostLine
	for _, line := range estimate.Breakdown {
		if line.Category == costingcontrol.CategoryTypeFuel {
			fuelLine = line
		}
	}
	require.NotNil(t, fuelLine)
	assert.Equal(t, costingcontrol.EffectiveRateSourceLiveIndex, fuelLine.EffectiveSource)
	assert.True(t, fuelLine.RatePerMile.Equal(decimal.RequireFromString("0.6")))
}

func TestEstimateShipment_LiveFuelFallsBackWithoutPrice(t *testing.T) {
	t.Parallel()

	control := benchmarkControl()
	control.UseLiveFuelPrice = true
	control.FuelIndexID = &testFuelIndexID

	entity := testShipment([]*shipment.ShipmentMove{
		{Loaded: true, Distance: float64Ptr(100)},
	}, "1000")

	controlRepo := mocks.NewMockCostingControlRepository(t)
	controlRepo.EXPECT().GetByOrgID(mock.Anything, mock.Anything).Return(control, nil)

	shipmentRepo := mocks.NewMockShipmentRepository(t)
	shipmentRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(entity, nil)

	priceRepo := mocks.NewMockFuelIndexPriceRepository(t)
	priceRepo.EXPECT().
		GetLatestOnOrBefore(mock.Anything, mock.Anything).
		Return([]*fuelsurcharge.FuelIndexPrice{}, nil)

	svc := costingservice.NewTestService(costingservice.TestServiceParams{
		Repo:         controlRepo,
		ShipmentRepo: shipmentRepo,
		PriceRepo:    priceRepo,
		Now:          func() time.Time { return testNow },
	})

	estimate, err := svc.EstimateShipment(t.Context(), entity.ID, testTenantInfo())
	require.NoError(t, err)

	assert.True(
		t,
		estimate.Profile.TotalCPM.Equal(decimal.RequireFromString("2.435")),
		"missing live price should fall back to fuel benchmark, got %s", estimate.Profile.TotalCPM,
	)
}

func TestEstimateShipment_OverrideRate(t *testing.T) {
	t.Parallel()

	control := benchmarkControl()
	for _, category := range control.Categories {
		if category.Category == costingcontrol.CategoryTypeDriverWages {
			category.RateSource = costingcontrol.RateSourceOverride
			category.OverrideRatePerMile = decimal.NewNullDecimal(
				decimal.RequireFromString("1.00"),
			)
		}
	}

	entity := testShipment([]*shipment.ShipmentMove{
		{Loaded: true, Distance: float64Ptr(100)},
	}, "1000")

	svc := newEstimatorService(t, control, entity)

	estimate, err := svc.EstimateShipment(t.Context(), entity.ID, testTenantInfo())
	require.NoError(t, err)

	// 2.435 - 0.82 + 1.00 = 2.615
	assert.True(
		t,
		estimate.Profile.TotalCPM.Equal(decimal.RequireFromString("2.615")),
		"override should replace driver wages benchmark, got %s", estimate.Profile.TotalCPM,
	)
}

func TestEstimateShipment_GLActualRate(t *testing.T) {
	t.Parallel()

	glAccountID := pulid.MustNew("gla_")

	control := benchmarkControl()
	for _, category := range control.Categories {
		if category.Category == costingcontrol.CategoryTypeMaintenance {
			category.RateSource = costingcontrol.RateSourceGLActual
			category.GLAccounts = []*costingcontrol.CostCategoryGLAccount{
				{GLAccountID: glAccountID},
			}
		}
	}
	control.GLActualsEnabled = true

	entity := testShipment([]*shipment.ShipmentMove{
		{Loaded: true, Distance: float64Ptr(100)},
	}, "1000")

	controlRepo := mocks.NewMockCostingControlRepository(t)
	controlRepo.EXPECT().GetByOrgID(mock.Anything, mock.Anything).Return(control, nil)

	shipmentRepo := mocks.NewMockShipmentRepository(t)
	shipmentRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(entity, nil)

	actualsRepo := mocks.NewMockCostingActualsRepository(t)
	actualsRepo.EXPECT().
		FleetMiles(mock.Anything, mock.Anything).
		Return(&repositories.FleetMilesResult{TotalMiles: 10000, LoadedMiles: 9000, DeadheadMiles: 1000}, nil)
	actualsRepo.EXPECT().
		SumExpenseByAccounts(mock.Anything, mock.MatchedBy(func(req *repositories.SumExpenseByAccountsRequest) bool {
			return len(req.GLAccountIDs) == 1 && req.GLAccountIDs[0] == glAccountID
		})).
		Return(map[pulid.ID]decimal.Decimal{
			glAccountID: decimal.RequireFromString("3000"),
		}, nil)

	svc := costingservice.NewTestService(costingservice.TestServiceParams{
		Repo:         controlRepo,
		ShipmentRepo: shipmentRepo,
		ActualsRepo:  actualsRepo,
		Now:          func() time.Time { return testNow },
	})

	estimate, err := svc.EstimateShipment(t.Context(), entity.ID, testTenantInfo())
	require.NoError(t, err)

	// Maintenance becomes 3000 / 10000 = 0.30 instead of 0.215 benchmark.
	assert.True(
		t,
		estimate.Profile.TotalCPM.Equal(decimal.RequireFromString("2.52")),
		"GL actual should replace maintenance benchmark, got %s", estimate.Profile.TotalCPM,
	)

	require.NotNil(t, estimate.Profile.GLWindow)
	assert.True(t, estimate.Profile.GLWindow.HasPostings)
	assert.InDelta(t, 10000.0, estimate.Profile.GLWindow.FleetMiles, 0.001)
}

func TestEstimateShipment_GLActualNoPostingsFallsBack(t *testing.T) {
	t.Parallel()

	glAccountID := pulid.MustNew("gla_")

	control := benchmarkControl()
	for _, category := range control.Categories {
		if category.Category == costingcontrol.CategoryTypeMaintenance {
			category.RateSource = costingcontrol.RateSourceGLActual
			category.GLAccounts = []*costingcontrol.CostCategoryGLAccount{
				{GLAccountID: glAccountID},
			}
		}
	}
	control.GLActualsEnabled = true

	entity := testShipment([]*shipment.ShipmentMove{
		{Loaded: true, Distance: float64Ptr(100)},
	}, "1000")

	controlRepo := mocks.NewMockCostingControlRepository(t)
	controlRepo.EXPECT().GetByOrgID(mock.Anything, mock.Anything).Return(control, nil)

	shipmentRepo := mocks.NewMockShipmentRepository(t)
	shipmentRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(entity, nil)

	actualsRepo := mocks.NewMockCostingActualsRepository(t)
	actualsRepo.EXPECT().
		FleetMiles(mock.Anything, mock.Anything).
		Return(&repositories.FleetMilesResult{}, nil)
	actualsRepo.EXPECT().
		SumExpenseByAccounts(mock.Anything, mock.Anything).
		Return(map[pulid.ID]decimal.Decimal{}, nil)

	svc := costingservice.NewTestService(costingservice.TestServiceParams{
		Repo:         controlRepo,
		ShipmentRepo: shipmentRepo,
		ActualsRepo:  actualsRepo,
		Now:          func() time.Time { return testNow },
	})

	estimate, err := svc.EstimateShipment(t.Context(), entity.ID, testTenantInfo())
	require.NoError(t, err)

	assert.True(
		t,
		estimate.Profile.TotalCPM.Equal(decimal.RequireFromString("2.435")),
		"no postings should fall back to benchmark, got %s", estimate.Profile.TotalCPM,
	)
	require.NotNil(t, estimate.Profile.GLWindow)
	assert.False(t, estimate.Profile.GLWindow.HasPostings)
}

func TestEstimateShipment_ZeroRevenueNullMargin(t *testing.T) {
	t.Parallel()

	entity := testShipment([]*shipment.ShipmentMove{
		{Loaded: true, Distance: float64Ptr(100)},
	}, "0")

	svc := newEstimatorService(t, benchmarkControl(), entity)

	estimate, err := svc.EstimateShipment(t.Context(), entity.ID, testTenantInfo())
	require.NoError(t, err)

	assert.False(t, estimate.MarginPercent.Valid)
	assert.True(t, estimate.Profit.IsNegative())
}

func TestEstimateShipment_MissingDistanceFlagged(t *testing.T) {
	t.Parallel()

	entity := testShipment([]*shipment.ShipmentMove{
		{Loaded: true, Distance: float64Ptr(250)},
		{Loaded: true, Distance: nil},
	}, "1000")

	svc := newEstimatorService(t, benchmarkControl(), entity)

	estimate, err := svc.EstimateShipment(t.Context(), entity.ID, testTenantInfo())
	require.NoError(t, err)

	assert.True(t, estimate.MissingDistance)
	assert.InDelta(t, 250.0, estimate.TotalMiles, 0.001)
}

func TestEstimateShipment_CanceledMovesExcluded(t *testing.T) {
	t.Parallel()

	entity := testShipment([]*shipment.ShipmentMove{
		{Loaded: true, Distance: float64Ptr(250)},
		{Loaded: true, Distance: float64Ptr(999), Status: shipment.MoveStatusCanceled},
	}, "1000")

	svc := newEstimatorService(t, benchmarkControl(), entity)

	estimate, err := svc.EstimateShipment(t.Context(), entity.ID, testTenantInfo())
	require.NoError(t, err)

	assert.InDelta(t, 250.0, estimate.TotalMiles, 0.001)
	assert.False(t, estimate.MissingDistance)
}

func TestEstimateShipment_ZeroLoadedMilesNullRates(t *testing.T) {
	t.Parallel()

	entity := testShipment([]*shipment.ShipmentMove{
		{Loaded: false, Distance: float64Ptr(100)},
	}, "500")

	svc := newEstimatorService(t, benchmarkControl(), entity)

	estimate, err := svc.EstimateShipment(t.Context(), entity.ID, testTenantInfo())
	require.NoError(t, err)

	assert.False(t, estimate.BreakEvenRPM.Valid)
	assert.False(t, estimate.RevenuePerLoadedMile.Valid)
}

func TestFleetSummary(t *testing.T) {
	t.Parallel()

	control := benchmarkControl()

	controlRepo := mocks.NewMockCostingControlRepository(t)
	controlRepo.EXPECT().GetByOrgID(mock.Anything, mock.Anything).Return(control, nil)

	actualsRepo := mocks.NewMockCostingActualsRepository(t)
	actualsRepo.EXPECT().
		FleetCostAggregates(mock.Anything, mock.MatchedBy(func(req *repositories.FleetCostAggregatesRequest) bool {
			return req.CostPerMile.Equal(decimal.RequireFromString("2.435")) &&
				req.IncludeDeadheadMiles
		})).
		Return(&repositories.FleetCostAggregatesResult{
			ShipmentCount:     20,
			UnprofitableCount: 4,
			TotalRevenue:      decimal.RequireFromString("60000"),
			TotalMiles:        20000,
			LoadedMiles:       17000,
			DeadheadMiles:     3000,
		}, nil)

	svc := costingservice.NewTestService(costingservice.TestServiceParams{
		Repo:        controlRepo,
		ActualsRepo: actualsRepo,
		Now:         func() time.Time { return testNow },
	})

	summary, err := svc.FleetSummary(t.Context(), testTenantInfo(), 0, testNow.Unix())
	require.NoError(t, err)

	assert.Equal(t, 20, summary.ShipmentCount)
	assert.Equal(t, 4, summary.UnprofitableCount)
	assert.True(t, summary.AvgCPM.Equal(decimal.RequireFromString("2.435")))

	// cost = 2.435 x 20000 = 48700; margin = (60000 - 48700) / 60000 = 18.83%
	assert.True(
		t,
		summary.TotalEstimatedCost.Equal(decimal.RequireFromString("48700")),
		"got %s", summary.TotalEstimatedCost,
	)
	require.True(t, summary.AvgMarginPercent.Valid)
	assert.True(
		t,
		summary.AvgMarginPercent.Decimal.Equal(decimal.RequireFromString("18.83")),
		"got %s", summary.AvgMarginPercent.Decimal,
	)
	assert.InDelta(t, 3000.0, summary.EmptyMiles, 0.001)
}

func TestEstimateShipment_GLActualFixedCategoryUsesPlannedMiles(t *testing.T) {
	t.Parallel()

	glAccountID := pulid.MustNew("gla_")
	plannedMiles := int64(10000)

	control := benchmarkControl()
	control.GLActualsEnabled = true
	control.PlannedMonthlyMiles = &plannedMiles
	for _, category := range control.Categories {
		if category.Category == costingcontrol.CategoryTypeInsurance {
			category.RateSource = costingcontrol.RateSourceGLActual
			category.GLAccounts = []*costingcontrol.CostCategoryGLAccount{
				{GLAccountID: glAccountID},
			}
		}
	}

	entity := testShipment([]*shipment.ShipmentMove{
		{Loaded: true, Distance: float64Ptr(100)},
	}, "1000")

	controlRepo := mocks.NewMockCostingControlRepository(t)
	controlRepo.EXPECT().GetByOrgID(mock.Anything, mock.Anything).Return(control, nil)

	shipmentRepo := mocks.NewMockShipmentRepository(t)
	shipmentRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(entity, nil)

	actualsRepo := mocks.NewMockCostingActualsRepository(t)
	actualsRepo.EXPECT().
		FleetMiles(mock.Anything, mock.Anything).
		Return(&repositories.FleetMilesResult{TotalMiles: 1000, LoadedMiles: 1000}, nil)
	actualsRepo.EXPECT().
		SumExpenseByAccounts(mock.Anything, mock.Anything).
		Return(map[pulid.ID]decimal.Decimal{
			glAccountID: decimal.RequireFromString("3600"),
		}, nil)

	svc := costingservice.NewTestService(costingservice.TestServiceParams{
		Repo:         controlRepo,
		ShipmentRepo: shipmentRepo,
		ActualsRepo:  actualsRepo,
		Now:          func() time.Time { return testNow },
	})

	estimate, err := svc.EstimateShipment(t.Context(), entity.ID, testTenantInfo())
	require.NoError(t, err)

	// Fixed category divisor = max(1000 fleet miles, 10000 x 3 months) = 30000.
	// Insurance rate = 3600 / 30000 = 0.12 replacing the 0.106 benchmark.
	assert.True(
		t,
		estimate.Profile.TotalCPM.Equal(decimal.RequireFromString("2.449")),
		"fixed GL category should divide by planned miles, got %s", estimate.Profile.TotalCPM,
	)
}
