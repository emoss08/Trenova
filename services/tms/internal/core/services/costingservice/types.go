package costingservice

import (
	"github.com/emoss08/trenova/internal/core/domain/costingcontrol"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type FuelResolution struct {
	PricePerGallon decimal.NullDecimal
	PriceDate      string
	FuelIndexID    *pulid.ID
	MilesPerGallon decimal.Decimal
	Source         costingcontrol.EffectiveRateSource
}

type ResolvedCategoryRate struct {
	Category        costingcontrol.CategoryType
	Name            string
	CostBehavior    costingcontrol.CostBehavior
	RatePerMile     decimal.Decimal
	EffectiveSource costingcontrol.EffectiveRateSource
}

type GLWindowInfo struct {
	FromDate    int64
	ToDate      int64
	FleetMiles  float64
	HasPostings bool
}

type ResolvedCostProfile struct {
	TotalCPM             decimal.Decimal
	VariableCPM          decimal.Decimal
	FixedCPM             decimal.Decimal
	Categories           []*ResolvedCategoryRate
	Fuel                 *FuelResolution
	TargetMarginPercent  decimal.NullDecimal
	IncludeDeadheadMiles bool
	AsOfDate             string
	GLWindow             *GLWindowInfo
}

type CategoryCostLine struct {
	Category        costingcontrol.CategoryType
	Name            string
	CostBehavior    costingcontrol.CostBehavior
	RatePerMile     decimal.Decimal
	Amount          decimal.Decimal
	EffectiveSource costingcontrol.EffectiveRateSource
}

type ShipmentProfitabilityEstimate struct {
	ShipmentID           pulid.ID
	LoadedMiles          float64
	DeadheadMiles        float64
	TotalMiles           float64
	Revenue              decimal.Decimal
	EstimatedCost        decimal.Decimal
	Profit               decimal.Decimal
	MarginPercent        decimal.NullDecimal
	RevenuePerLoadedMile decimal.NullDecimal
	BreakEvenRPM         decimal.NullDecimal
	MissingDistance      bool
	Breakdown            []*CategoryCostLine
	Profile              *ResolvedCostProfile
}

type FleetCostSummary struct {
	AvgCPM             decimal.Decimal
	AvgMarginPercent   decimal.NullDecimal
	ShipmentCount      int
	UnprofitableCount  int
	TotalRevenue       decimal.Decimal
	TotalEstimatedCost decimal.Decimal
	TotalMiles         float64
	EmptyMiles         float64
}
