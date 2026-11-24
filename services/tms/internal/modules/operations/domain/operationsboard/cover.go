package operationsboard

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
)

const (
	MetricCoverAgainstLoads = "coverAgainstLoads"
	MaxCoverCarriers        = 8
	MaxCoverLoads           = 25

	FigureLapsingCarriers = "lapsingCarriers"
	FigureLapsedCarriers  = "lapsedCarriers"
	FigurePastCover       = "pastCover"
	FigureLoadsWithThem   = "loadsWithThem"
)

var coverAgainstLoads = boardcatalog.Metric{
	Name:    MetricCoverAgainstLoads,
	Form:    boardcatalog.FormSpans,
	Periods: []boardcatalog.Period{boardcatalog.PeriodNext30Days},
	Requires: []permission.Resource{
		permission.ResourceCarrier, permission.ResourceShipment,
	},
	Aliases: []string{
		"loadsAgainstCover", "insuranceAgainstLoads", "coverVsLoads", "loadsPastCover",
		"insuranceLapsing", "coverLapsing", "lapsingCover", "carrierInsuranceLapsing",
		"insuranceExpiringLoads", "loadsAfterCoverEnds",
	},
	Reads: slices.Concat(
		boardcatalog.Read(permission.ResourceCarrier, "insurancePolicies"),
		boardcatalog.Read(permission.ResourceShipment, "proNumber", "status"),
		boardcatalog.Read(permission.ResourceShipmentStop, "scheduledWindowEnd"),
	),
	Words: "carriers whose required insurance lapses in the next 30 days, each load they " +
		"hold against the day cover ends",
	Sketch: boardcatalog.Sketch{Rows: MaxCoverCarriers},
	Figures: []boardcatalog.Figure{
		{
			Name:  FigureLapsingCarriers,
			Label: "Carriers whose cover ends in 30 days",
			Value: boardcatalog.ValueCount,
		},
		{
			Name:  FigureLapsedCarriers,
			Label: "Carriers whose cover lapsed in the last 90 days",
			Value: boardcatalog.ValueCount,
		},
		{
			Name:  FigurePastCover,
			Label: "Loads delivering after their carrier's cover ends",
			Value: boardcatalog.ValueCount,
		},
		{
			Name:  FigureLoadsWithThem,
			Label: "Our open loads with these carriers",
			Value: boardcatalog.ValueCount,
		},
	},
}
