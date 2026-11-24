package fuelboard

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
)

const (
	MeasureNetTax = "netTax"

	MetricIFTAReturnChecks = "iftaReturnChecks"

	FigureFilingBlockers   = "filingBlockers"
	FigureNetTaxDue        = "netTaxDue"
	FigureRatesMissing     = "ratesMissing"
	FigureLargestCredit    = "largestCredit"
	FigureLargestCreditFor = "largestCreditJurisdiction"
	FigureFilingDue        = "filingDueDay"

	GetIFTAReturnTool = "fuel_get_ifta_return"

	IFTAReturnCheckRows = 9
)

var Kinds = []boardcatalog.Kind{
	{Kind: recordkind.IFTAReturnLine, Words: "IFTA return lines"},
}

var lineReads = boardcatalog.Read(
	permission.ResourceIFTAReturn,
	"lineTotal", "rateMissing", "totalMiles", "taxPaidGallons", "jurisdiction",
)

var Measures = []boardcatalog.Measure{
	{
		Name:     MeasureNetTax,
		Kind:     recordkind.IFTAReturnLine,
		Shape:    boardcatalog.ShapeCentred,
		Value:    boardcatalog.ValueMoney,
		Summable: true,
		Default:  true,
		Aliases: []string{
			"lineTotal", "lineTotalMinor", "taxDue", "netTaxDue", "tax", "owed", "credit",
			"netDue",
		},
		Reads:   lineReads,
		Words:   "tax owed on a jurisdiction's line, or the credit it earns",
		Label:   "Net tax",
		Total:   "Net tax on these lines",
		Largest: "Largest tax owed",
		Heading: "Net tax",
	},
}

var returnReads = boardcatalog.Read(
	permission.ResourceIFTAReturn,
	"status", "problems", "period", "unattributedMiles", "noTractorMiles", "lines",
)

var iftaMetrics = []boardcatalog.Metric{
	{
		Name:      MetricIFTAReturnChecks,
		Form:      boardcatalog.FormChecks,
		For:       recordkind.IFTAReturn,
		ForNeeded: true,
		Aliases: []string{
			"iftaChecks", "iftaReadiness", "readyToFile", "iftaReturnProblems",
			"iftaFilingChecks", "fuelTaxReturnChecks", "iftaProblems", "iftaReady",
		},
		Words: "whether a quarter's IFTA return can be filed: each check on it, what " +
			"blocks filing and what needs a look",
		Reads:  returnReads,
		Sketch: boardcatalog.Sketch{Rows: IFTAReturnCheckRows},
		Records: &boardcatalog.MetricRecords{
			Kind:    recordkind.IFTAReturnLine,
			Measure: MeasureNetTax,
			Words:   "the return's jurisdiction lines, owed or credit",
			From:    GetIFTAReturnTool,
		},
		Figures: []boardcatalog.Figure{
			{
				Name:  FigureFilingBlockers,
				Label: "Checks that block filing",
				Value: boardcatalog.ValueCount,
			},
			boardcatalog.MoneyFigure(
				FigureNetTaxDue,
				"Net tax on the return",
				boardcatalog.Read(permission.ResourceIFTAReturn, "netDue"),
			),
			{
				Name:  FigureRatesMissing,
				Label: "Lines with no tax rate",
				Value: boardcatalog.ValueCount,
			},
			boardcatalog.MoneyFigure(
				FigureLargestCredit,
				"Largest credit on a line",
				boardcatalog.Read(permission.ResourceIFTAReturn, "lineTotal"),
			),
			{
				Name:  FigureLargestCreditFor,
				Label: "Jurisdiction with the largest credit",
				Value: boardcatalog.ValueText,
			},
			{
				Name:  FigureFilingDue,
				Label: "Day the return is due",
				Value: boardcatalog.ValueText,
			},
		},
	},
}

var Metrics = slices.Concat(iftaMetrics, fuelMetrics)
