package costingcontrol

import "github.com/shopspring/decimal"

type benchmarkSeed struct {
	category CategoryType
	name     string
	behavior CostBehavior
	rate     string
}

// ATRI Operational Costs of Trucking, 2025 marginal cost per mile line items.
var benchmarkSeeds = []benchmarkSeed{
	{CategoryTypeDriverWages, "Driver Wages", CostBehaviorVariable, "0.82"},
	{CategoryTypeDriverBenefits, "Driver Benefits", CostBehaviorVariable, "0.21"},
	{CategoryTypeFuel, "Fuel", CostBehaviorVariable, "0.48"},
	{CategoryTypeEquipmentPayments, "Truck & Trailer Payments", CostBehaviorFixed, "0.404"},
	{CategoryTypeMaintenance, "Repair & Maintenance", CostBehaviorVariable, "0.215"},
	{CategoryTypeInsurance, "Insurance Premiums", CostBehaviorFixed, "0.106"},
	{CategoryTypeTires, "Tires", CostBehaviorVariable, "0.05"},
	{CategoryTypeTolls, "Tolls", CostBehaviorVariable, "0.04"},
	{CategoryTypePermitsLicenses, "Permits & Licenses", CostBehaviorFixed, "0.01"},
	{CategoryTypeOverhead, "Overhead & Admin", CostBehaviorFixed, "0.10"},
}

func DefaultCategories() []*CostCategory {
	categories := make([]*CostCategory, 0, len(benchmarkSeeds))
	for i, seed := range benchmarkSeeds {
		categories = append(categories, &CostCategory{
			Category:             seed.category,
			Name:                 seed.name,
			CostBehavior:         seed.behavior,
			RateSource:           RateSourceBenchmark,
			BenchmarkRatePerMile: decimal.RequireFromString(seed.rate),
			IsActive:             true,
			//nolint:gosec,nolintlint // bounded by benchmarkSeeds length
			SortOrder: int16(i),
		})
	}
	return categories
}

func DefaultMilesPerGallon() decimal.Decimal {
	return decimal.RequireFromString("6.5")
}

const DefaultFleetSummaryWindowSeconds = int64(30 * 24 * 60 * 60)
