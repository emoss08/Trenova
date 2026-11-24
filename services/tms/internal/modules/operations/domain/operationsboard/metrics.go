package operationsboard

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
)

const (
	MetricOnTimeByWeek   = "onTimeByWeek"
	MetricLateByCustomer = "lateByCustomer"
	MetricDriverCover    = "driverCover"
	MaxCoverAreas        = 6
)

var stopService = boardcatalog.Read(
	permission.ResourceShipmentStop,
	"actualArrival",
	"scheduledWindowEnd",
)

var Metrics = []boardcatalog.Metric{
	{
		Name: MetricOnTimeByWeek,
		Form: boardcatalog.FormTrend,
		Periods: []boardcatalog.Period{
			boardcatalog.PeriodThisQuarter,
			boardcatalog.PeriodLastQuarter,
		},
		Requires: []permission.Resource{permission.ResourceShipment},
		Aliases: []string{
			"onTime", "onTimeTrend", "onTimeDelivery", "onTimeDeliveryTrend", "onTimeWeekly",
			"onTimeRate", "onTimeShare", "onTimeThisQuarter", "deliveryOnTime", "serviceTrend",
		},
		Reads:    stopService,
		Unscoped: []permission.Resource{permission.ResourceShipment},
		Words: "how on-time delivery trended: share on time each week of a quarter, " +
			"the target and the lowest week marked as the slip",
		Leads: []string{"average", "lowest", "lowestWeek"},
		Figures: []boardcatalog.Figure{
			{Name: "average", Label: "On time, the quarter", Value: boardcatalog.ValuePercent},
			{Name: "lowest", Label: "On time, the lowest week", Value: boardcatalog.ValuePercent},
			{
				Name:  "lowestWeek",
				Label: "The day the lowest week starts",
				Value: boardcatalog.ValueText,
			},
			{
				Name:  "latest",
				Label: "On time, the last full week",
				Value: boardcatalog.ValuePercent,
			},
			{Name: "target", Label: "On-time target", Value: boardcatalog.ValuePercent},
			{
				Name:  "previous",
				Label: "On time, the quarter before",
				Value: boardcatalog.ValuePercent,
			},
		},
	},
	{
		Name: MetricLateByCustomer,
		Form: boardcatalog.FormBars,
		Periods: []boardcatalog.Period{
			boardcatalog.PeriodSlip,
			boardcatalog.PeriodLastWeek,
			boardcatalog.PeriodThisWeek,
		},
		Requires: []permission.Resource{
			permission.ResourceShipment, permission.ResourceCustomer,
		},
		Aliases: []string{
			"lateDeliveriesByCustomer", "lateByCustomers", "lateCustomers",
			"lateDeliveries", "lateLoadsByCustomer", "slipByCustomer",
		},
		Reads:    append(boardcatalog.Read(permission.ResourceCustomer, "name"), stopService...),
		Unscoped: []permission.Resource{permission.ResourceShipment},
		Words: "where on-time slipped: late deliveries in one week ranked by customer; " +
			"period slip is the week onTimeByWeek marks lowest",
		Sketch: boardcatalog.Sketch{Rows: 5},
		Leads:  []string{"late", "topCustomer"},
		Figures: []boardcatalog.Figure{
			{Name: "late", Label: "Late deliveries that week", Value: boardcatalog.ValueCount},
			{Name: "delivered", Label: "Deliveries that week", Value: boardcatalog.ValueCount},
			{Name: "top", Label: "Late at the first customer", Value: boardcatalog.ValueCount},
			{
				Name:  "topCustomer",
				Label: "The customer with the most late",
				Value: boardcatalog.ValueText,
			},
		},
	},
	{
		Name:     MetricDriverCover,
		Form:     boardcatalog.FormGrid,
		Periods:  []boardcatalog.Period{boardcatalog.PeriodNext48Hours},
		Requires: []permission.Resource{permission.ResourceShipmentMove},
		Feature:  platformcatalog.FeatureDispatchIntelligence,
		Aliases: []string{
			"driverCoverage", "driversShort", "driverShortage", "driversNeeded",
			"driverNeed", "driverSupply", "coverage", "shortOnDrivers", "forecastCover",
			"freightOutlook", "driverForecast",
		},
		Words: "where drivers fall short of forecast loads: forecast areas by 4-hour slot, " +
			"the next 48 hours only (no week)",
		Reads:  boardcatalog.Read(permission.ResourceShipmentMove, "status"),
		Sketch: boardcatalog.Sketch{Rows: MaxCoverAreas},
		Leads:  []string{"areasShort", "short", "worstArea", "worstSlot"},
		Figures: []boardcatalog.Figure{
			{Name: "areasShort", Label: "Forecast areas short", Value: boardcatalog.ValueCount},
			{Name: "short", Label: "Loads short of a driver", Value: boardcatalog.ValueCount},
			{Name: "mayBeShort", Label: "Slots that may be short", Value: boardcatalog.ValueCount},
			{Name: "worst", Label: "Short in the worst slot", Value: boardcatalog.ValueCount},
			{Name: "worstArea", Label: "The area of the worst slot", Value: boardcatalog.ValueText},
			{Name: "worstSlot", Label: "The worst slot", Value: boardcatalog.ValueText},
			{Name: "spare", Label: "Drivers spare after 48 hours", Value: boardcatalog.ValueCount},
		},
	},
	coverAgainstLoads,
}
