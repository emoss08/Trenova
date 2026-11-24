package fuelquerytools

import (
	"cmp"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardhint"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelboard"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/typeutils"
	"github.com/shopspring/decimal"
)

const (
	paramIFTAReturnID  = "iftaReturnId"
	fieldLines         = "lines"
	fieldLineTotal     = "lineTotal"
	fieldRateMissing   = "rateMissing"
	fieldJurisdiction  = "jurisdiction"
	fieldName          = "name"
	fieldLabel         = "label"
	rowLineCount       = "lineCount"
	rowLinesNote       = "linesNote"
	rowHowToShow       = "howToShow"
	iftaLinesShown     = 12
	iftaReturnEntity   = "IFTA return"
	iftaReturnGetIDSrc = "from fuel_list_ifta_returns"

	iftaReturnGetSummary = "Retrieve one quarter's IFTA return by id: its status, the filing " +
		"date, the net tax due, the problems it carries and its jurisdiction lines, each " +
		"with the net tax it owes or the credit it earns (the largest first, every line " +
		"with no rate kept). Use it to say whether a return is ready to file; " +
		"fuel_list_ifta_returns gives the id."
	iftaLinesNote = "The lines owing or crediting the most, and every line with no rate; " +
		"lineCount is the whole return."
)

// IFTAReturnGetSpec reads one return through the Desk's IFTA return card.
func IFTAReturnGetSpec() readspec.Get {
	return readspec.Get{
		Name:        fuelboard.GetIFTAReturnTool,
		Entity:      iftaReturnEntity,
		SearchTerms: []string{"IFTA return lines", "ready to file IFTA", "fuel tax return"},
		Resource:    permission.ResourceIFTAReturn,
		Summary:     iftaReturnGetSummary,
		ParamName:   paramIFTAReturnID,
		IDSource:    iftaReturnGetIDSrc,
		Shape:       shapeIFTAReturnCard,
	}
}

type iftaLine struct {
	record  map[string]any
	total   decimal.Decimal
	missing bool
}

// shapeIFTAReturnCard names each line by its jurisdiction, keeps the lines
// that weigh most on the return and says how to show whether it can be filed.
func shapeIFTAReturnCard(
	record map[string]any,
	withheld map[string]struct{},
	_ filtercatalog.Clock,
) {
	if id, ok := record["id"].(string); ok && id != "" {
		record[rowHowToShow] = boardhint.Line(&boardhint.Show{
			Metric: fuelboard.MetricIFTAReturnChecks,
			For:    pulid.ID(id),
		})
	}
	if _, hidden := withheld[fieldLines]; hidden {
		return
	}
	raw, _ := record[fieldLines].([]any)
	lines := make([]iftaLine, 0, len(raw))
	for _, item := range raw {
		line, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if jurisdiction, isRecord := line[fieldJurisdiction].(map[string]any); isRecord {
			if name, named := jurisdiction[fieldName].(string); named {
				line[fieldLabel] = name
			}
		}
		total, _ := decimal.NewFromString(typeutils.StringOf(line[fieldLineTotal]))
		missing, _ := line[fieldRateMissing].(bool)
		lines = append(lines, iftaLine{record: line, total: total, missing: missing})
	}
	record[rowLineCount] = len(lines)
	if len(lines) <= iftaLinesShown {
		return
	}
	slices.SortStableFunc(lines, func(a, b iftaLine) int {
		if a.missing != b.missing {
			if a.missing {
				return -1
			}
			return 1
		}
		return cmp.Compare(b.total.Abs().InexactFloat64(), a.total.Abs().InexactFloat64())
	})
	kept := make([]any, 0, iftaLinesShown)
	for idx := range lines {
		if idx >= iftaLinesShown && !lines[idx].missing {
			break
		}
		kept = append(kept, lines[idx].record)
	}
	record[fieldLines] = kept
	record[rowLinesNote] = iftaLinesNote
}
