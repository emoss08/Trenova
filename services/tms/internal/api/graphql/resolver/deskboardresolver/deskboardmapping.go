package deskboardresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/shared/jsonutils"
)

func boardCardToModel(card *agentports.BoardCard) *gqlmodel.DeskBoardCard {
	model := &gqlmodel.DeskBoardCard{
		ArtifactID: card.ArtifactID.String(),
		ThreadID:   card.ThreadID.String(),
		Title:      card.Title,
		Author:     card.Author,
		Question:   card.Question,
		AnsweredAt: int(card.AnsweredAt),
		ReadAt:     int(card.Filled.ReadAt),
		Verdict:    segmentsToModel(card.Filled.Verdict),
		Blocks:     make([]*gqlmodel.DeskBoardBlock, 0, len(card.Filled.Blocks)),
		Markdown:   card.Markdown,
	}
	for idx := range card.Filled.Blocks {
		model.Blocks = append(model.Blocks, blockToModel(&card.Filled.Blocks[idx]))
	}

	return model
}

func blockToModel(block *board.FilledBlock) *gqlmodel.DeskBoardBlock {
	model := &gqlmodel.DeskBoardBlock{
		Type:    string(block.Type),
		Title:   block.Title,
		Half:    block.Half,
		Numeral: block.Numeral,
		State:   string(block.State),
		Tone:    string(block.Tone),
		Source: &gqlmodel.DeskBoardSource{
			Kind:     string(block.Source.Kind),
			Resource: string(block.Source.Resource),
			Count:    block.Source.Count,
			Readable: block.Source.Readable,
			Title:    block.Source.Title,
		},
		Figures:  make([]*gqlmodel.DeskBoardFigure, 0, len(block.Figures)),
		Measure:  string(block.Measure),
		Rows:     rowsToModel(block.Rows),
		Events:   make([]*gqlmodel.DeskBoardEvent, 0, len(block.Events)),
		Checks:   make([]*gqlmodel.DeskBoardCheck, 0, len(block.Checks)),
		Text:     segmentsToModel(block.Text),
		NoteTone: string(block.NoteTone),
		Chips:    rowsToModel(block.Chips),
	}
	for idx := range block.Figures {
		figure := &block.Figures[idx]
		model.Figures = append(model.Figures, &gqlmodel.DeskBoardFigure{
			Label: figure.Label,
			Value: valueToModel(figure.Value),
			Was:   valueToModel(figure.Was),
			Tone:  string(figure.Tone),
		})
	}
	if block.Compare != nil {
		model.Compare = compareToModel(block.Compare)
	}
	if block.Table != nil {
		model.Table = &gqlmodel.DeskBoardTable{
			Table: asJSON(block.Table.Table),
			Total: block.Table.Total,
		}
		if block.Table.Artifact.IsNotNil() {
			id := block.Table.Artifact.String()
			model.Table.ArtifactID = &id
		}
	}
	if block.Chart != nil {
		model.Chart = asJSON(block.Chart)
	}
	for idx := range block.Events {
		event := &block.Events[idx]
		model.Events = append(model.Events, &gqlmodel.DeskBoardEvent{
			At:            int(event.At),
			Operation:     event.Operation,
			Actor:         event.Actor,
			PrincipalType: event.Principal,
			Comment:       event.Comment,
			Changes:       event.Changes,
		})
	}
	for idx := range block.Checks {
		check := &block.Checks[idx]
		model.Checks = append(model.Checks, &gqlmodel.DeskBoardCheck{
			Title:   check.Title,
			Tone:    string(check.Tone),
			Detail:  check.Detail,
			Count:   valueToModel(check.Count),
			Was:     valueToModel(check.Was),
			Records: rowsToModel(check.Records),
			Hidden:  check.Hidden,
		})
	}
	if block.Meter != nil {
		tones := make([]string, 0, len(block.Meter.Tones))
		for _, tone := range block.Meter.Tones {
			tones = append(tones, string(tone))
		}
		model.Meter = &gqlmodel.DeskBoardMeter{
			Clear:   block.Meter.Clear,
			Total:   block.Meter.Total,
			Tones:   tones,
			Percent: valueToModel(block.Meter.Percent),
		}
	}

	return model
}

func compareToModel(compare *board.FilledCompare) *gqlmodel.DeskBoardCompare {
	model := &gqlmodel.DeskBoardCompare{
		Columns: rowsToModel(compare.Columns),
		Lines:   make([]*gqlmodel.DeskBoardCompareLine, 0, len(compare.Lines)),
		Wins:    compare.Wins,
	}
	for idx := range compare.Lines {
		line := &compare.Lines[idx]
		cells := make([]*gqlmodel.DeskBoardValue, 0, len(line.Cells))
		for _, cell := range line.Cells {
			cells = append(cells, valueToModel(cell))
		}
		model.Lines = append(model.Lines, &gqlmodel.DeskBoardCompareLine{
			Label:  line.Label,
			Better: string(line.Better),
			Cells:  cells,
			Wins:   line.Wins,
		})
	}

	return model
}

func rowsToModel(rows []board.Row) []*gqlmodel.DeskBoardRow {
	out := make([]*gqlmodel.DeskBoardRow, 0, len(rows))
	for idx := range rows {
		row := &rows[idx]
		model := &gqlmodel.DeskBoardRow{
			Resource: string(row.Resource),
			ID:       row.ID.String(),
			Readable: row.Readable,
			Label:    row.Label,
			Subtitle: row.Subtitle,
			Status:   row.Status,
		}
		if measure := row.Measure; measure != nil {
			model.Measure = &gqlmodel.DeskBoardRowMeasure{
				Measure: string(measure.Measure),
				Value:   valueToModel(&measure.Value),
				State:   string(measure.State),
				Place:   measure.Place,
				Tone:    string(measure.Tone),
			}
			if measure.At > 0 {
				at := int(measure.At)
				model.Measure.At = &at
			}
		}
		out = append(out, model)
	}

	return out
}

func segmentsToModel(segments []board.FilledSegment) []*gqlmodel.DeskBoardSegment {
	out := make([]*gqlmodel.DeskBoardSegment, 0, len(segments))
	for _, segment := range segments {
		out = append(out, &gqlmodel.DeskBoardSegment{
			Text:    segment.Text,
			Numeral: segment.Numeral,
			Tone:    string(segment.Tone),
			Struck:  segment.Struck,
		})
	}

	return out
}

func valueToModel(value *board.Value) *gqlmodel.DeskBoardValue {
	if value == nil {
		return nil
	}

	return &gqlmodel.DeskBoardValue{
		Kind:     string(value.Kind),
		Number:   value.Number.String(),
		Currency: value.Currency,
		Text:     value.Text,
	}
}

func asJSON(value any) map[string]any {
	out := map[string]any{}
	if err := jsonutils.Convert(value, &out); err != nil {
		return nil
	}

	return out
}
