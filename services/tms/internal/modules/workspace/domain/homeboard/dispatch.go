package homeboard

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
)

const MetricDispatchToday = "dispatchToday"

var shipmentRevenue = boardcatalog.Read(permission.ResourceShipment, "totalChargeAmount")

var Metrics = []boardcatalog.Metric{
	{
		Name:     MetricDispatchToday,
		Form:     boardcatalog.FormFigures,
		Periods:  []boardcatalog.Period{boardcatalog.PeriodToday},
		Requires: []permission.Resource{permission.ResourceShipment},
		Aliases:  []string{"dispatch", "today", "operations", "homeFigures"},
		Words:    "the dispatch figures the home page shows for today",
		Reads:    boardcatalog.Read(permission.ResourceShipment, "status"),
		Sketch:   boardcatalog.Sketch{Rows: 4},
		Figures: []boardcatalog.Figure{
			{Name: "activeShipments", Label: "Active shipments", Value: boardcatalog.ValueCount},
			boardcatalog.MoneyFigure("revenueToday", "Revenue today", shipmentRevenue),
			{Name: "onTimePercent", Label: "On-time %", Value: boardcatalog.ValuePercent},
			{Name: "emptyMilePercent", Label: "Empty mile %", Value: boardcatalog.ValuePercent},
			{Name: "atRisk", Label: "At risk", Value: boardcatalog.ValueCount},
			{Name: "unassigned", Label: "Unassigned", Value: boardcatalog.ValueCount},
			{Name: "readyToDispatch", Label: "Ready to dispatch", Value: boardcatalog.ValueCount},
			{
				Name:    "marginPercent",
				Label:   "Margin %",
				Value:   boardcatalog.ValuePercent,
				Feature: platformcatalog.FeatureAccounting,
				Reads:   boardcatalog.Read(permission.ResourceShipment, "marginPercent"),
			},
			boardcatalog.MoneyFigure("revenuePerMile", "Revenue per mile", shipmentRevenue),
			{
				Name:    "costPerMile",
				Label:   "Cost per mile",
				Value:   boardcatalog.ValueMoney,
				Feature: platformcatalog.FeatureAccounting,
				Reads:   boardcatalog.Read(permission.ResourceShipment, "costPerMile"),
			},
			{Name: "movesLate", Label: "Late moves", Value: boardcatalog.ValueCount},
			{Name: "movesDueNow", Label: "Moves due now", Value: boardcatalog.ValueCount},
			{Name: "movesDueToday", Label: "Moves due today", Value: boardcatalog.ValueCount},
			{Name: "movesUncovered", Label: "Uncovered moves", Value: boardcatalog.ValueCount},
		},
	},
}
