package partiesboard

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
)

const MeasureCoverLeft = "coverLeft"

var LoadsReads = boardcatalog.Read(permission.ResourceShipment, "status")

var Kinds = []boardcatalog.Kind{
	{Kind: recordkind.Kind(permission.ResourceCarrier)},
	{Kind: recordkind.Kind(permission.ResourceCustomer)},
}

var Measures = []boardcatalog.Measure{
	{
		Name:      MeasureCoverLeft,
		Kind:      recordkind.Kind(permission.ResourceCarrier),
		Shape:     boardcatalog.ShapeCountdown,
		Value:     boardcatalog.ValueDays,
		Ascending: true,
		Default:   true,
		Aliases: []string{
			"expirationDate", "insuranceExpiry", "policyEnds", "coverEnds", "insuranceEnds",
			"daysOfCover", "coverDaysLeft", "insuranceLapse", "lapsesIn", "daysLeft",
			"insuranceExpiration", "policyExpiry",
		},
		Reads: boardcatalog.Read(
			permission.ResourceCarrier, "insurancePolicies", "complianceStatus",
		),
		Place: &boardcatalog.PlacePart{
			Requires: []permission.Resource{permission.ResourceShipment},
			Reads:    LoadsReads,
		},
		Words:   "time left on its required insurance, soonest first",
		Label:   "Cover left",
		Total:   "Total cover left",
		Largest: "Soonest to lapse",
		Heading: "Left",
		FollowUp: &boardcatalog.FollowUp{
			Label: "Ask for renewed certificates",
			Ask: "Draft a short request to each carrier on this board whose required cover has " +
				"lapsed or ends soon, asking for its renewed certificate, as a document I can read " +
				"and send myself.",
		},
	},
}
