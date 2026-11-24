package ediboard

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
)

const MeasureWaiting = "waiting"

var Kinds = []boardcatalog.Kind{
	{Kind: recordkind.EDIInboundFile, Words: "inbound EDI files"},
	{Kind: recordkind.EDIMessage, Words: "EDI messages"},
}

var Measures = []boardcatalog.Measure{
	{
		Name:    MeasureWaiting,
		Kind:    recordkind.EDIInboundFile,
		Shape:   boardcatalog.ShapeAmount,
		Value:   boardcatalog.ValueDuration,
		Adverse: true,
		Default: true,
		Aliases: []string{
			"waitingFor", "waitingTime", "age", "since", "receivedAt", "quarantinedFor",
			"failureReason", "status",
		},
		Reads:   boardcatalog.Read(permission.ResourceEDI, "receivedAt", "failureReason"),
		Words:   "time a file not processed has waited, and why it failed",
		Label:   "Waiting",
		Total:   "Waiting",
		Largest: "Longest waiting",
		Heading: "Waiting",
	},
	{
		Name:    MeasureWaiting,
		Kind:    recordkind.EDIMessage,
		Shape:   boardcatalog.ShapeAmount,
		Value:   boardcatalog.ValueDuration,
		Adverse: true,
		Default: true,
		Aliases: []string{
			"waitingFor", "waitingTime", "age", "since", "generatedAt", "deliveryStatus",
			"deliveryLastError", "ackStatus", "failedFor",
		},
		Reads:   boardcatalog.Read(permission.ResourceEDI, "generatedAt", "deliveryLastError"),
		Words:   "time a message not yet through has waited, and its error",
		Label:   "Waiting",
		Total:   "Waiting",
		Largest: "Longest waiting",
		Heading: "Waiting",
	},
}

const (
	MetricPartnerDay = "partnerDay"

	PartnerDayRows = 20
)

var Metrics = []boardcatalog.Metric{
	{
		Name:      MetricPartnerDay,
		Form:      boardcatalog.FormLog,
		For:       recordkind.EDIPartner,
		ForNeeded: true,
		Periods:   []boardcatalog.Period{boardcatalog.PeriodToday, boardcatalog.PeriodYesterday},
		Aliases: []string{
			"ediPartnerDay", "partnerToday", "tendersToday", "todaysTenders", "ediDay",
			"tenderLog", "partnerTenders", "ediTenders", "tenders", "ediLog",
		},
		Words: "one EDI partner's tenders in a day: each 204, the shipment it made, our 990 " +
			"and its 997; for: the partner's id",
		Reads:  boardcatalog.Read(permission.ResourceEDI, "status", "receivedAt", "generatedAt"),
		Sketch: boardcatalog.Sketch{Rows: 6},
		Figures: []boardcatalog.Figure{
			{Name: "tendersIn", Label: "Tenders in", Value: boardcatalog.ValueCount},
			{Name: "shipmentsMade", Label: "Shipments made", Value: boardcatalog.ValueCount},
			{Name: "answersDelivered", Label: "Answers delivered", Value: boardcatalog.ValueCount},
			{Name: "needHand", Label: "Need a hand", Value: boardcatalog.ValueCount},
			{
				Name:  "medianAnswer",
				Label: "Median time to answer",
				Value: boardcatalog.ValueDuration,
			},
		},
	},
}
