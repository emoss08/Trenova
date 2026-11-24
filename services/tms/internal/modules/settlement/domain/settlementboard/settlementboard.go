package settlementboard

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
)

const (
	MeasureNetPay = "netPay"

	MetricSettlementBatch = "settlementBatch"
	MetricSettlementWeeks = "settlementWeeks"

	FigureBatchNet            = "batchNet"
	FigureBatchGross          = "batchGross"
	FigureBatchReimbursements = "batchReimbursements"
	FigureBatchDeducted       = "batchDeducted"
	FigureBatchSettlements    = "batchSettlements"
	FigureBatchPayDate        = "batchPayDate"
	FigureReadyNet            = "readyNet"
	FigureReadySettlements    = "readySettlements"
	FigureBlockedNet          = "blockedNet"
	FigureBlockedSettlements  = "blockedSettlements"
	FigureHeldSettlements     = "heldSettlements"
	FigureCriticalSettlements = "criticalExceptionSettlements"
	FigureDisputeSettlements  = "openDisputeSettlements"
	FigureLargestBatchEarned  = "largestBatchEarned"
	FigureLargestBatchPeriod  = "largestBatchPeriod"
	FigureLatestBatchEarned   = "latestBatchEarned"
	FigureLatestBatchNet      = "latestBatchNet"
	BlockerTool               = "settlement_list_batch_blockers"
	WeeksShown                = 5
	settlementPart            = permission.ResourceDriverSettlement
	disputePart               = permission.ResourceSettlementDispute
	settlementKind            = recordkind.Kind(permission.ResourceDriverSettlement)
	metricRowsSplit           = 6
	measureWords              = "net pay, or what keeps the settlement from release"
	batchWords                = "the newest driver pay batch: net, gross and deducted, " +
		"ready to release against what cannot be released as it stands"
	weeksWords = "the last five driver pay batches: what each paid out as net and what " +
		"came off as deductions"
)

var (
	netReads     = boardcatalog.Read(settlementPart, "netPayMinor")
	grossReads   = boardcatalog.Read(settlementPart, "grossEarningsMinor")
	reimbReads   = boardcatalog.Read(settlementPart, "reimbursementsMinor")
	disputeReads = boardcatalog.Read(disputePart, "status", "category")
	batchReads   = slices.Concat(
		boardcatalog.Read(settlementPart,
			"netPayMinor", "grossEarningsMinor", "reimbursementsMinor", "status", "exceptions",
			"payDate", "holdReason"),
		boardcatalog.Read(disputePart, "status"),
	)
	totalsReads = boardcatalog.Read(settlementPart,
		"totalGrossMinor", "totalNetMinor", "totalReimbursementsMinor", "periodStart")
	earnedReads = boardcatalog.Read(settlementPart, "totalGrossMinor", "totalReimbursementsMinor")
	holdReads   = boardcatalog.Read(settlementPart, "holdReason")
)

var Kinds = []boardcatalog.Kind{
	{Kind: settlementKind, Words: "driver settlements"},
}

var Measures = []boardcatalog.Measure{
	{
		Name:     MeasureNetPay,
		Kind:     settlementKind,
		Shape:    boardcatalog.ShapeAmount,
		Value:    boardcatalog.ValueMoney,
		Summable: true,
		Default:  true,
		Aliases: []string{
			"net", "netPayMinor", "netAmount", "takeHome", "pay", "amount", "total",
		},
		Reads: boardcatalog.Read(settlementPart,
			"netPayMinor", "carryForwardOutMinor", "exceptions", "status", "holdReason",
			"holdNote"),
		Place: &boardcatalog.PlacePart{
			Requires: []permission.Resource{disputePart},
			Reads:    disputeReads,
		},
		Words:   measureWords,
		Label:   "Net pay",
		Total:   "Total net pay",
		Largest: "Largest net pay",
		Heading: "Net",
	},
}

var Metrics = []boardcatalog.Metric{
	{
		Name:     MetricSettlementBatch,
		Form:     boardcatalog.FormSplit,
		Requires: []permission.Resource{settlementPart, disputePart},
		Unscoped: []permission.Resource{permission.ResourceWorker},
		Reads:    batchReads,
		Aliases: []string{
			"settlementBatchTotals", "payBatch", "batch", "payRun", "thisWeeksBatch",
			"driverPayBatch", "settlementsBatch", "batchSplit", "settlementRun",
		},
		Words:  batchWords,
		Sketch: boardcatalog.Sketch{Rows: metricRowsSplit},
		Records: &boardcatalog.MetricRecords{
			Kind:    settlementKind,
			Measure: MeasureNetPay,
			Words:   "the batch's settlements that cannot be released as they stand",
			From:    BlockerTool,
		},
		Leads: []string{
			FigureBatchNet, FigureBlockedSettlements, FigureCriticalSettlements,
			FigureDisputeSettlements, FigureHeldSettlements,
		},
		Figures: []boardcatalog.Figure{
			boardcatalog.MoneyFigure(FigureBatchNet, "Net pay", netReads),
			boardcatalog.MoneyFigure(FigureBatchGross, "Gross earnings", grossReads),
			boardcatalog.MoneyFigure(FigureBatchReimbursements, "Reimbursements", reimbReads),
			boardcatalog.MoneyFigure(FigureBatchDeducted, "Deducted",
				slices.Concat(netReads, grossReads, reimbReads)),
			{Name: FigureBatchSettlements, Label: "Settlements", Value: boardcatalog.ValueCount},
			{Name: FigureBatchPayDate, Label: "Pay date", Value: boardcatalog.ValueText},
			boardcatalog.MoneyFigure(FigureReadyNet, "Ready to release", netReads),
			{
				Name:  FigureReadySettlements,
				Label: "Settlements ready to release",
				Value: boardcatalog.ValueCount,
			},
			boardcatalog.MoneyFigure(FigureBlockedNet, "Cannot be released as they stand",
				slices.Concat(netReads, disputeReads)),
			{
				Name:  FigureBlockedSettlements,
				Label: "Settlements that cannot be released as they stand",
				Value: boardcatalog.ValueCount,
				Reads: disputeReads,
			},
			{
				Name:  FigureHeldSettlements,
				Label: "Settlements on hold",
				Value: boardcatalog.ValueCount,
				Reads: holdReads,
			},
			{
				Name:  FigureCriticalSettlements,
				Label: "Settlements with a critical exception",
				Value: boardcatalog.ValueCount,
			},
			{
				Name:  FigureDisputeSettlements,
				Label: "Settlements with a dispute open",
				Value: boardcatalog.ValueCount,
				Reads: disputeReads,
			},
		},
	},
	{
		Name:     MetricSettlementWeeks,
		Form:     boardcatalog.FormBars,
		Requires: []permission.Resource{settlementPart},
		Unscoped: []permission.Resource{permission.ResourceWorker},
		Reads:    totalsReads,
		Aliases: []string{
			"batchHistory", "settlementHistory", "lastFiveBatches", "payRunHistory",
			"weeklyPay", "settlementTrend", "batchTrend", "batchWeeks",
		},
		Words: weeksWords,
		Figures: []boardcatalog.Figure{
			boardcatalog.MoneyFigure(FigureLargestBatchEarned, "Largest of the five", earnedReads),
			{
				Name:  FigureLargestBatchPeriod,
				Label: "The week of the largest",
				Value: boardcatalog.ValueText,
			},
			boardcatalog.MoneyFigure(FigureLatestBatchEarned, "Newest batch earned", earnedReads),
			boardcatalog.MoneyFigure(FigureLatestBatchNet, "Newest batch net",
				boardcatalog.Read(settlementPart, "totalNetMinor")),
		},
	},
}
