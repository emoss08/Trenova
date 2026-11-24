package operationsboard

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
)

const (
	MeasureLateness = "lateness"
	MeasureRevenue  = "revenue"
)

var Kinds = []boardcatalog.Kind{
	{Kind: recordkind.Kind(permission.ResourceShipment)},
}

var Measures = []boardcatalog.Measure{
	{
		Name:    MeasureLateness,
		Kind:    recordkind.Kind(permission.ResourceShipment),
		Shape:   boardcatalog.ShapeTrack,
		Value:   boardcatalog.ValueDuration,
		Adverse: true,
		Aliases: []string{"lateBy", "lateByMinutes", "lateMinutes", "late", "behind"},
		Reads: boardcatalog.Read(
			permission.ResourceShipmentStop, "actualArrival", "scheduledWindowEnd",
		),
		Words:   "past its next window, or estimated to miss one still open",
		Label:   "Time behind",
		Total:   "Total time behind",
		Largest: "Furthest behind",
		Heading: "Behind",
	},
	{
		Name:     MeasureRevenue,
		Kind:     recordkind.Kind(permission.ResourceShipment),
		Shape:    boardcatalog.ShapeAmount,
		Value:    boardcatalog.ValueMoney,
		Summable: true,
		Aliases: []string{
			"totalCharge", "totalChargeAmount", "totalCharges", "charge", "charges", "total",
		},
		Reads:   boardcatalog.Read(permission.ResourceShipment, "totalChargeAmount"),
		Label:   "Revenue",
		Total:   "Total revenue",
		Largest: "Largest revenue",
		Heading: "Revenue",
	},
}
