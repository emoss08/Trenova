package fuelboard

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
)

const (
	MetricFuelCostPerMile   = "fuelCostPerMile"
	MetricFuelCostByTractor = "fuelCostByTractor"

	FigureCostPerMile         = "currentCostPerMile"
	FigurePreviousCostPerMile = "previousCostPerMile"
	FigurePricePerGallon      = "pricePerGallon"
	FigureFuelSpend           = "fuelSpend"
	FigureMilesRun            = "milesRun"

	FigureFleetCostPerMile   = "fleetCostPerMile"
	FigureTractorsAbove      = "tractorsAboveRange"
	FigureTractorsBelow      = "tractorsBelowRange"
	FigureHighestTractor     = "highestTractor"
	FigureHighestCostPerMile = "highestCostPerMile"
	FigureLowestTractor      = "lowestTractor"
	FigureLowestCostPerMile  = "lowestCostPerMile"
	FigureUsualLow           = "usualRangeLow"
	FigureUsualHigh          = "usualRangeHigh"

	MaxTractorDots = 60
	RunDays        = 7
)

var fuelPeriods = []boardcatalog.Period{boardcatalog.PeriodThisMonth, boardcatalog.PeriodLastMonth}

var fuelRequires = []permission.Resource{
	permission.ResourceFuelPurchase,
	permission.ResourceIFTAJurisdictionMileage,
}

var fuelReads = slices.Concat(
	boardcatalog.Read(permission.ResourceFuelPurchase, "totalAmount", "gallons", "purchasedAt"),
	boardcatalog.Read(permission.ResourceIFTAJurisdictionMileage, "miles", "traveledAt"),
)

var spendReads = boardcatalog.Read(permission.ResourceFuelPurchase, "totalAmount")

var fuelMetrics = []boardcatalog.Metric{
	{
		Name:     MetricFuelCostPerMile,
		Form:     boardcatalog.FormTrend,
		Periods:  fuelPeriods,
		Requires: fuelRequires,
		Reads:    fuelReads,
		Unscoped: []permission.Resource{permission.ResourceShipment},
		Aliases: []string{
			"fuelCost", "fuelCostTrend", "fuelPerMile", "fuelSpendPerMile",
			"fuelCpm", "cpm",
		},
		Leads: []string{FigureCostPerMile, FigurePreviousCostPerMile, FigurePricePerGallon},
		Words: "what fuel costs a mile across the fleet, day by day over a month as a seven-day " +
			"run, against the month before",
		Figures: []boardcatalog.Figure{
			boardcatalog.MoneyFigure(FigureCostPerMile, "Fuel cost per mile", spendReads),
			boardcatalog.MoneyFigure(
				FigurePreviousCostPerMile, "Fuel cost per mile the month before", spendReads,
			),
			boardcatalog.MoneyFigure(
				FigurePricePerGallon, "Average price paid a gallon", spendReads,
			),
			boardcatalog.MoneyFigure(FigureFuelSpend, "Fuel bought", spendReads),
			{Name: FigureMilesRun, Label: "Miles run", Value: boardcatalog.ValueNumber},
		},
	},
	{
		Name:     MetricFuelCostByTractor,
		Form:     boardcatalog.FormDots,
		Periods:  fuelPeriods,
		Requires: slices.Concat(fuelRequires, []permission.Resource{permission.ResourceTractor}),
		Reads:    slices.Concat(boardcatalog.Read(permission.ResourceTractor, "code"), fuelReads),
		Unscoped: []permission.Resource{permission.ResourceShipment},
		Aliases: []string{
			"fuelCostPerTractor", "tractorFuelCost", "costPerMileByTractor", "fuelOutliers",
			"tractorsOffTheLine", "fuelByTractor", "usualFuelRange",
		},
		Leads: []string{
			FigureTractorsAbove, FigureHighestTractor, FigureHighestCostPerMile,
			FigureTractorsBelow,
		},
		Words: "each tractor's cost per mile over a month against the range most of the fleet " +
			"runs in; the furthest outside named",
		Figures: []boardcatalog.Figure{
			{
				Name:  FigureTractorsAbove,
				Label: "Tractors running above the fleet's usual range",
				Value: boardcatalog.ValueCount,
			},
			{
				Name:  FigureHighestTractor,
				Label: "The tractor furthest above the usual range",
				Value: boardcatalog.ValueText,
			},
			boardcatalog.MoneyFigure(
				FigureHighestCostPerMile, "Cost per mile of the furthest above", spendReads,
			),
			{
				Name:  FigureTractorsBelow,
				Label: "Tractors running below the fleet's usual range",
				Value: boardcatalog.ValueCount,
			},
			{
				Name:  FigureLowestTractor,
				Label: "The tractor furthest below the usual range",
				Value: boardcatalog.ValueText,
			},
			boardcatalog.MoneyFigure(
				FigureLowestCostPerMile, "Cost per mile of the furthest below", spendReads,
			),
			boardcatalog.MoneyFigure(
				FigureUsualLow, "Low end of the fleet's usual range", spendReads,
			),
			boardcatalog.MoneyFigure(
				FigureUsualHigh, "High end of the fleet's usual range", spendReads,
			),
			boardcatalog.MoneyFigure(
				FigureFleetCostPerMile, "The fleet's cost per mile", spendReads,
			),
		},
	},
}
