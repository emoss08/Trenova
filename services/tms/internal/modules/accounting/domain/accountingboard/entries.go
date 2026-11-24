package accountingboard

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
)

const (
	MeasureAmount     = "amount"
	MeasurePendingFor = "pendingFor"
)

var Kinds = []boardcatalog.Kind{
	{Kind: recordkind.Kind(permission.ResourceJournalEntry)},
}

var Measures = []boardcatalog.Measure{
	{
		Name:     MeasureAmount,
		Kind:     recordkind.Kind(permission.ResourceJournalEntry),
		Shape:    boardcatalog.ShapeAmount,
		Value:    boardcatalog.ValueMoney,
		Summable: true,
		Aliases:  []string{"totalDebit", "debit", "total"},
		Reads:    boardcatalog.Read(permission.ResourceJournalEntry, "totalDebit"),
		Label:    "Amount",
		Total:    "Total amount",
		Largest:  "Largest amount",
		Heading:  "Amount",
	},
	{
		Name:    MeasurePendingFor,
		Kind:    recordkind.Kind(permission.ResourceJournalEntry),
		Shape:   boardcatalog.ShapeTrack,
		Value:   boardcatalog.ValueDuration,
		Adverse: true,
		Aliases: []string{"pending", "waiting", "waitingFor"},
		Reads:   boardcatalog.Read(permission.ResourceJournalEntry, "createdAt", "status"),
		Label:   "Time pending",
		Total:   "Total time pending",
		Largest: "Longest pending",
		Heading: "Waiting",
	},
}

const MetricPeriodClose = "periodClose"

var Metrics = []boardcatalog.Metric{
	{
		Name:      MetricPeriodClose,
		Form:      boardcatalog.FormChecks,
		For:       recordkind.Kind(permission.ResourceFiscalPeriod),
		ForNeeded: true,
		Requires:  []permission.Resource{permission.ResourceJournalEntry},
		Aliases: []string{
			"closeBlockers", "fiscalCloseBlockers", "periodCloseBlockers", "closeChecks",
			"monthEndClose", "whatHoldsTheClose", "fiscalClose",
		},
		Words: "what stops a fiscal period closing: every close check, blocking or clear, " +
			"with its count; the entries waiting to post",
		Reads: append(
			boardcatalog.Read(permission.ResourceFiscalPeriod, "name", "status"),
			boardcatalog.Read(permission.ResourceJournalEntry, "status", "createdAt")...,
		),
		Sketch: boardcatalog.Sketch{Rows: 7},
		Records: &boardcatalog.MetricRecords{
			Kind:    recordkind.Kind(permission.ResourceJournalEntry),
			Measure: MeasureAmount,
			Words:   "the entries still waiting to post",
			From:    "ledger_list_journal_entries for this period with status Pending",
		},
		Figures: []boardcatalog.Figure{
			{Name: "blocking", Label: "Checks blocking the close", Value: boardcatalog.ValueCount},
			{
				Name:  "pendingEntries",
				Label: "Entries waiting to post",
				Value: boardcatalog.ValueCount,
			},
			boardcatalog.MoneyFigure(
				"pendingAmount",
				"Waiting to post",
				boardcatalog.Read(permission.ResourceJournalEntry, "totalDebit"),
			),
			{
				Name:  "oldestPending",
				Label: "Oldest entry waiting",
				Value: boardcatalog.ValueDuration,
			},
		},
	},
}
