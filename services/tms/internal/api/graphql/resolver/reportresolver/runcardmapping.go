package reportresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	reportingservice "github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
)

func reportRunDeskCardToModel(card *reportingservice.RunCard) *gqlmodel.ReportRunDeskCard {
	model := &gqlmodel.ReportRunDeskCard{
		Run: reportRunToModel(card.Run),
		Report: &gqlmodel.ReportRunDeskReport{
			Name:        card.Report.Name,
			Description: card.Report.Description,
			Category:    card.Report.Category,
			CannedKey:   nilIfEmpty(card.Report.CannedKey),
		},
		Parameters: make([]*gqlmodel.ReportRunDeskParameter, 0, len(card.Parameters)),
		Timezone:   card.Timezone,
		Withheld:   nilIfEmpty(string(card.Withheld)),
		Sort:       make([]*gqlmodel.ReportRunDeskSort, 0, len(card.Sort)),
	}
	if !card.Report.DefinitionID.IsNil() {
		definitionID := card.Report.DefinitionID.String()
		model.Report.DefinitionID = &definitionID
	}
	if card.Sample != nil {
		model.Sample = reportPreviewToModel(card.Sample)
	}

	for i := range card.Parameters {
		parameter := &card.Parameters[i]
		model.Parameters = append(model.Parameters, &gqlmodel.ReportRunDeskParameter{
			Name:        parameter.Name,
			Label:       parameter.Label,
			Type:        parameter.Type,
			Value:       parameter.Value,
			IsDefault:   parameter.IsDefault,
			Entity:      nilIfEmpty(parameter.Entity),
			WindowStart: nilIfZero(parameter.WindowStart),
			WindowEnd:   nilIfZero(parameter.WindowEnd),
		})
	}

	if card.Previous != nil {
		model.Previous = &gqlmodel.ReportRunDeskPrevious{
			RunID:       card.Previous.RunID.String(),
			CompletedAt: int(card.Previous.CompletedAt),
			RowCount:    int(card.Previous.RowCount),
			ColumnIds:   emptyIfNil(card.Previous.ColumnIDs),
			Totals:      card.Previous.Totals,
		}
	}

	if card.Schedule != nil {
		model.Schedule = reportScheduleToModel(card.Schedule)
	}

	if card.Chart != nil {
		model.Chart = &gqlmodel.ReportRunDeskChart{
			Type:   string(card.Chart.Type),
			X:      nilIfEmpty(card.Chart.X),
			Series: emptyIfNil(card.Chart.Series),
		}
	}

	for _, sort := range card.Sort {
		model.Sort = append(model.Sort, &gqlmodel.ReportRunDeskSort{
			Column: sort.Column,
			Desc:   sort.Desc,
		})
	}

	return model
}
