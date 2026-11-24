package billingboard

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
)

const (
	MetricReceivables       = "receivables"
	MetricAgingCompare      = "agingCompare"
	MetricCollecting        = "collecting"
	MetricUnappliedReceipts = "unappliedReceipts"

	FigureReceiptsToApply          = "receiptsToApply"
	FigureAmountToApply            = "amountToApply"
	FigureOldestToApply            = "oldestToApply"
	FigureReceiptsExact            = "receiptsExact"
	FigureReceiptsSameAmount       = "receiptsSameAmount"
	FigureReceiptsShort            = "receiptsShort"
	FigureReceiptsOver             = "receiptsOver"
	FigureReceiptsWithoutCandidate = "receiptsWithoutCandidate"
)

var (
	receivableAmounts = boardcatalog.Read(permission.ResourceAccountsReceivable, "amountMinor")
	receiptAmounts    = boardcatalog.Read(permission.ResourceBankReceipt, "amountMinor")
	candidateAmounts  = slices.Concat(
		receiptAmounts,
		boardcatalog.Read(permission.ResourceInvoice, "balanceDue"),
		boardcatalog.Read(permission.ResourceCustomerPayment, "amountMinor"),
	)
	comparePeriods = []boardcatalog.Period{
		boardcatalog.PeriodLastMonth,
		boardcatalog.PeriodLastWeek,
		boardcatalog.PeriodLastQuarter,
	}
)

var Metrics = []boardcatalog.Metric{
	{
		Name:     MetricReceivables,
		Form:     boardcatalog.FormFigures,
		Requires: []permission.Resource{permission.ResourceAccountsReceivable},
		Aliases:  []string{"ar", "accountsReceivable", "receivable"},
		Words:    "open and overdue receivables and days sales outstanding",
		Reads:    receivableAmounts,
		Sketch:   boardcatalog.Sketch{Rows: 1},
		Figures: []boardcatalog.Figure{
			boardcatalog.MoneyFigure("arTotalOpen", "Open receivables", receivableAmounts),
			boardcatalog.MoneyFigure("arOverdue", "Overdue receivables", receivableAmounts),
			{
				Name:  "arDaysSalesOutstanding",
				Label: "Days sales outstanding",
				Value: boardcatalog.ValueNumber,
			},
		},
	},
	{
		Name:     MetricAgingCompare,
		Form:     boardcatalog.FormBars,
		Periods:  comparePeriods,
		Requires: []permission.Resource{permission.ResourceAccountsReceivable},
		Aliases: []string{
			"agingAgainstLastMonth", "agingTrend", "agingChange", "arAging", "agingBuckets",
			"aging", "receivableAging",
		},
		Words: "receivable aging now against a month (or week, quarter) ago, bucket by bucket, " +
			"the past-due bucket that grew most marked",
		Reads:  receivableAmounts,
		Sketch: boardcatalog.Sketch{Rows: 5},
		Records: &boardcatalog.MetricRecords{
			Kind:    recordkind.Kind(permission.ResourceInvoice),
			Measure: MeasureAmountDue,
			Words: "the open invoices of the bucket that grew (minDaysPastDue and maxDaysPastDue " +
				"of that bucket, largestFirst)",
			From: "receivable_list_open_items",
		},
		Figures: []boardcatalog.Figure{
			boardcatalog.MoneyFigure("openNow", "Open receivables now", receivableAmounts),
			boardcatalog.MoneyFigure("openThen", "Open receivables then", receivableAmounts),
			{
				Name:  "grewBucket",
				Label: "The past-due bucket that grew most",
				Value: boardcatalog.ValueText,
			},
			boardcatalog.MoneyFigure("grewBucketNow", "That bucket now", receivableAmounts),
			boardcatalog.MoneyFigure("grewBucketThen", "That bucket then", receivableAmounts),
			{Name: "openInvoicesNow", Label: "Open invoices now", Value: boardcatalog.ValueCount},
		},
	},
	{
		Name:     MetricCollecting,
		Form:     boardcatalog.FormFigures,
		Periods:  comparePeriods,
		Requires: []permission.Resource{permission.ResourceAccountsReceivable},
		Aliases: []string{
			"collections", "collectionPerformance", "daysToCollect", "collectionEffectiveness",
		},
		Words: "how fast customers pay and how much of what was due came in, over 91 days, " +
			"against a month (or week, quarter) ago",
		Reads:  receivableAmounts,
		Sketch: boardcatalog.Sketch{Rows: 1},
		Figures: []boardcatalog.Figure{
			{Name: "daysToPay", Label: "Days to pay", Value: boardcatalog.ValueNumber},
			{
				Name:  "collectedShare",
				Label: "Collected of what was due",
				Value: boardcatalog.ValuePercent,
			},
			boardcatalog.MoneyFigure("collectedAmount", "Collected in 91 days", receivableAmounts),
		},
	},
	{
		Name:     MetricUnappliedReceipts,
		Form:     boardcatalog.FormFigures,
		Requires: []permission.Resource{permission.ResourceBankReceipt},
		Aliases: []string{
			"unappliedCash", "bankExceptions", "receiptExceptions",
			"receiptsNotApplied", "receiptCandidates", "receiptMatches",
		},
		Words:  "receipts not applied (in Exception): how many, the total, the oldest; how many match an open item exactly, short, over",
		Reads:  boardcatalog.Read(permission.ResourceBankReceipt, "status", "receiptDate"),
		Sketch: boardcatalog.Sketch{Rows: 1},
		Records: &boardcatalog.MetricRecords{
			Kind:    recordkind.Kind(permission.ResourceBankReceipt),
			Measure: MeasureAgainstCandidate,
			Words: "the bank receipts not applied that need a decision, those whose reference " +
				"or memo names an invoice number first, each drawn against the open item it likely " +
				"pays",
			From: "receivable_list_bank_receipt_exceptions",
		},
		Leads: []string{
			FigureReceiptsToApply, FigureAmountToApply, FigureReceiptsExact, FigureReceiptsShort,
			FigureReceiptsOver, FigureReceiptsWithoutCandidate,
		},
		Figures: []boardcatalog.Figure{
			{
				Name:  FigureReceiptsToApply,
				Label: "Receipts not applied",
				Value: boardcatalog.ValueCount,
			},
			boardcatalog.MoneyFigure(FigureAmountToApply, "Not applied", receiptAmounts),
			{Name: FigureOldestToApply, Label: "Oldest waiting", Value: boardcatalog.ValueDays},
			candidateCount(FigureReceiptsExact, "Match an open item to the cent"),
			candidateCount(FigureReceiptsSameAmount, "Same amount, not named by their reference"),
			candidateCount(FigureReceiptsShort, "Short of the item they likely pay"),
			candidateCount(FigureReceiptsOver, "Over the item they likely pay"),
			candidateCount(FigureReceiptsWithoutCandidate, "No open item to match"),
		},
	},
}

func candidateCount(name, label string) boardcatalog.Figure {
	return boardcatalog.Figure{
		Name:  name,
		Label: label,
		Value: boardcatalog.ValueCount,
		Reads: candidateAmounts,
	}
}
