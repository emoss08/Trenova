package reportresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	reportingservice "github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/pkg/reportrows"
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
		Sample:     reportRunSampleToModel(card.Sample),
		Withheld:   nilIfEmpty(string(card.Withheld)),
		Sort:       make([]*gqlmodel.ReportRunDeskSort, 0, len(card.Sort)),
	}
	if !card.Report.DefinitionID.IsNil() {
		definitionID := card.Report.DefinitionID.String()
		model.Report.DefinitionID = &definitionID
	}

	for i := range card.Parameters {
		parameter := &card.Parameters[i]
		model.Parameters = append(model.Parameters, &gqlmodel.ReportRunDeskParameter{
			Name:        parameter.Name,
			Label:       parameter.Label,
			Type:        string(parameter.Type),
			Value:       parameter.Value,
			IsDefault:   parameter.IsDefault,
			RefEntity:   nilIfEmpty(parameter.RefEntity),
			WindowStart: nilIfZero(parameter.WindowStart),
			WindowEnd:   nilIfZero(parameter.WindowEnd),
		})
	}

	if card.Previous != nil {
		columnIDs := make([]string, 0, len(card.Previous.Schema))
		for i := range card.Previous.Schema {
			columnIDs = append(columnIDs, card.Previous.Schema[i].ID)
		}
		model.Previous = &gqlmodel.ReportRunDeskPrevious{
			RunID:       card.Previous.RunID.String(),
			CompletedAt: int(card.Previous.CompletedAt),
			ColumnIds:   columnIDs,
			Totals:      card.Previous.Totals,
		}
	}

	if card.Schedule != nil {
		model.Schedule = reportScheduleToModel(card.Schedule)
	}

	if card.Chart != nil {
		model.Chart = &gqlmodel.ReportRunDeskChart{
			Type:      string(card.Chart.Type),
			XColumnID: nilIfEmpty(card.Chart.XColumnID),
			SeriesIds: emptyIfNil(card.Chart.SeriesIDs),
		}
	}

	for _, sort := range card.Sort {
		model.Sort = append(model.Sort, &gqlmodel.ReportRunDeskSort{
			ColumnID:  sort.ColumnID,
			Direction: string(sort.Direction),
		})
	}

	return model
}

func reportRunSampleToModel(sample *reportrows.Sample) *gqlmodel.ReportRunSample {
	if sample == nil {
		return nil
	}

	columns := make([]*gqlmodel.ReportPreviewColumn, 0, len(sample.Schema))
	for i := range sample.Schema {
		column := &sample.Schema[i]
		columns = append(columns, &gqlmodel.ReportPreviewColumn{
			ID:      column.ID,
			Label:   column.Label,
			Type:    string(column.Type),
			Format:  nilIfEmpty(string(column.Format)),
			Display: reportDisplayToModel(&column.Display),
		})
	}

	rows := sample.Rows
	if rows == nil {
		rows = [][]any{}
	}
	model := &gqlmodel.ReportRunSample{
		Columns: columns,
		Rows:    rows,
		More:    sample.More,
	}
	if len(sample.Totals) > 0 {
		model.Totals = sample.Totals
	}

	return model
}
