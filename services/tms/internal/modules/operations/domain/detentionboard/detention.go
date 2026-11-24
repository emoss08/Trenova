package detentionboard

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
)

const pastFreeLabel = "Past free time"

const (
	MeasureDetention = "detention"
	MeasurePastFree  = "pastFree"
	MeasureDriverPay = "driverPay"
	MeasureDwell     = "dwell"
)

var Kinds = []boardcatalog.Kind{
	{Kind: recordkind.DetentionOccurrence, Words: "detention stays"},
}

var Measures = []boardcatalog.Measure{
	{
		Name:     MeasureDetention,
		Kind:     recordkind.DetentionOccurrence,
		Shape:    boardcatalog.ShapeBounded,
		Value:    boardcatalog.ValueMoney,
		Summable: true,
		Default:  true,
		Aliases: []string{
			"billableAmount", "billable", "charge", "detentionAmount", "detentionCharge",
		},
		Reads: boardcatalog.Read(
			permission.ResourceDetention,
			"billableAmount",
			"billableMinutes",
		),
		Words:   "billable amount; time on site against the free time",
		Label:   "Billable",
		Total:   "Total billable",
		Largest: "Largest charge",
		Heading: "Billable",
	},
	{
		Name:     MeasurePastFree,
		Kind:     recordkind.DetentionOccurrence,
		Shape:    boardcatalog.ShapeAmount,
		Value:    boardcatalog.ValueDuration,
		Summable: true,
		Adverse:  true,
		Aliases: []string{
			"billableMinutes", "minutesPastFree", "pastFreeMinutes", "overFreeTime", "waited",
		},
		Reads:   boardcatalog.Read(permission.ResourceDetention, "billableMinutes"),
		Words:   "time waited past free time; adds up across stays",
		Label:   pastFreeLabel,
		Total:   pastFreeLabel,
		Largest: "Longest past free time",
		Heading: "Past free",
	},
	{
		Name:     MeasureDriverPay,
		Kind:     recordkind.DetentionOccurrence,
		Shape:    boardcatalog.ShapeAmount,
		Value:    boardcatalog.ValueMoney,
		Summable: true,
		Reads:    boardcatalog.Read(permission.ResourceDetention, "driverPayAmount"),
		Aliases:  []string{"driverPayAmount", "driverPayout", "paidToDriver"},
		Words:    "what the driver is paid for the wait",
		Label:    "Driver pay",
		Total:    "Driver pay on them",
		Largest:  "Largest driver pay",
		Heading:  "Driver pay",
	},
	{
		Name:     MeasureDwell,
		Kind:     recordkind.Kind(permission.ResourceLocation),
		Shape:    boardcatalog.ShapeSpread,
		Value:    boardcatalog.ValueDuration,
		Summable: true,
		Adverse:  true,
		Default:  true,
		Requires: []permission.Resource{permission.ResourceDetention},
		Aliases: []string{
			"pastFree", "pastFreeMinutes", "waiting", "waitingTime", "detention",
			"medianDwell", "dwellTime", "medianDwellMinutes", "p90DwellMinutes",
		},
		Reads:   boardcatalog.Read(permission.ResourceDetention, "billableMinutes"),
		Words:   "time past free this quarter; free time, median, slowest",
		Label:   "Past free this quarter",
		Total:   "Past free time this quarter",
		Largest: "Most past free time",
		Heading: "Past free",
	},
}

const (
	MetricDetentionTotals        = "detentionTotals"
	MetricDetentionByArrivalHour = "detentionByArrivalHour"
)

var detentionPeriods = []boardcatalog.Period{
	boardcatalog.PeriodThisQuarter, boardcatalog.PeriodLastQuarter, boardcatalog.PeriodThisMonth,
	boardcatalog.PeriodLastMonth, boardcatalog.PeriodThisWeek, boardcatalog.PeriodLastWeek,
}

var detentionMoney = boardcatalog.Read(permission.ResourceDetention, "billableAmount")

var Metrics = []boardcatalog.Metric{
	{
		Name:     MetricDetentionTotals,
		Form:     boardcatalog.FormFigures,
		Periods:  detentionPeriods,
		Requires: []permission.Resource{permission.ResourceDetention},
		Records: &boardcatalog.MetricRecords{
			Kind:    recordkind.DetentionOccurrence,
			Measure: MeasureDetention,
			Words:   "the stays that need a decision (held back), at most 25",
			From:    "detention_list_stays",
		},
		Aliases: []string{
			"detentionSummary", "detentionFigures", "detentionThisQuarter", "waitingTotals",
		},
		Words: "detention at every facility: what is ready to bill, time past free, billed, " +
			"waived, notice on time",
		Reads:  boardcatalog.Read(permission.ResourceDetention, "status", "billableMinutes"),
		Sketch: boardcatalog.Sketch{Rows: 2},
		Figures: []boardcatalog.Figure{
			boardcatalog.MoneyFigure("readyToBill", "Ready to bill", detentionMoney),
			{
				Name:  "readyToBillStays",
				Label: "Stays ready to bill",
				Value: boardcatalog.ValueCount,
			},
			{Name: "pastFreeTime", Label: pastFreeLabel, Value: boardcatalog.ValueDuration},
			boardcatalog.MoneyFigure("billed", "Billed", detentionMoney),
			boardcatalog.MoneyFigure("waived", "Waived", detentionMoney),
			{Name: "noticeOnTime", Label: "Notice on time", Value: boardcatalog.ValuePercent},
		},
	},
	{
		Name:     MetricDetentionByArrivalHour,
		Form:     boardcatalog.FormColumns,
		Periods:  detentionPeriods,
		Requires: []permission.Resource{permission.ResourceDetention},
		Aliases: []string{
			"arrivalHour", "byArrivalHour", "detentionByArrival", "arrivalWindows",
			"detainedByArrivalHour",
		},
		Reads: boardcatalog.Read(permission.ResourceDetention, "status", "billableMinutes"),
		Words: "share of arrivals detained by two-hour arrival window, every facility " +
			"together, never one facility's",
		Figures: []boardcatalog.Figure{
			{Name: "worstWindow", Label: "Worst arrival window", Value: boardcatalog.ValueText},
			{
				Name:  "worstWindowShare",
				Label: "Detained in that window",
				Value: boardcatalog.ValuePercent,
			},
			{
				Name:  "detainedShare",
				Label: "Detained, every arrival",
				Value: boardcatalog.ValuePercent,
			},
		},
	},
}
