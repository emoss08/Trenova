package deskboardresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardregistry"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/shopspring/decimal"
)

func metricToModel(metric *board.FilledMetric) *gqlmodel.DeskBoardMetric {
	model := &gqlmodel.DeskBoardMetric{
		Name:    metric.Name,
		Form:    string(metric.Form),
		Figures: []*gqlmodel.DeskBoardMetricFigure{},
		Log:     []*gqlmodel.DeskBoardLogEntry{},
		Checks:  []*gqlmodel.DeskBoardCheck{},
	}
	read := metric.Read
	if read == nil {
		return model
	}
	model.Title = read.Title
	model.Source = read.Source
	model.Figures = figuresToModel(metric.Name, read)
	if read.Chart != nil {
		model.Chart = chartToModel(metric, read.Chart)
	}
	if read.Split != nil {
		model.Split = splitToModel(read.Split)
	}
	if read.Grid != nil {
		model.Grid = gridToModel(read.Grid)
	}
	for idx := range read.Log {
		entry := &read.Log[idx]
		model.Log = append(model.Log, &gqlmodel.DeskBoardLogEntry{
			At:      int(entry.At),
			Tone:    string(entry.Tone),
			Text:    entry.Text,
			Detail:  entry.Detail,
			Records: refsToModel(metric, entry.Refs),
		})
	}
	for idx := range read.Checks {
		check := &read.Checks[idx]
		model.Checks = append(model.Checks, &gqlmodel.DeskBoardCheck{
			Title:   check.Title,
			Tone:    string(check.Tone),
			Detail:  check.Detail,
			Count:   valueToModel(check.Count),
			Records: refsToModel(metric, check.Refs),
			Hidden:  check.Hidden,
		})
	}
	if read.Table != nil {
		model.Table = tableToModel(read.Table)
	}

	return model
}

func figuresToModel(
	name string,
	read *boardcatalog.MetricRead,
) []*gqlmodel.DeskBoardMetricFigure {
	out := make([]*gqlmodel.DeskBoardMetricFigure, 0, len(read.Figures))
	spec, known := boardregistry.Catalog().MetricNamed(name)
	if !known {
		return out
	}
	for _, figure := range spec.Figures {
		value, found := read.Figures[figure.Name]
		if !found {
			continue
		}
		model := &gqlmodel.DeskBoardMetricFigure{
			Name:  figure.Name,
			Label: figure.Label,
			Value: valueToModel(value),
		}
		if shift, shifted := read.Changes[figure.Name]; shifted && value != nil {
			model.Was = valueToModel(shift.Was)
			model.WasAt = timestampOf(shift.WasAt)
			model.Tone = string(shift.Tone)
		}
		out = append(out, model)
	}

	return out
}

func refsToModel(metric *board.FilledMetric, refs []boardcatalog.Ref) []*gqlmodel.DeskBoardRow {
	out := make([]*gqlmodel.DeskBoardRow, 0, len(refs))
	for idx := range refs {
		if row := metric.Row(&refs[idx]); row != nil {
			out = append(out, rowToModel(row))
		}
	}

	return out
}

func decimalText(number *decimal.Decimal) *string {
	if number == nil {
		return nil
	}
	text := number.String()

	return &text
}

func lineToModel(line *boardcatalog.Line) *gqlmodel.DeskBoardLine {
	if line == nil {
		return nil
	}

	return &gqlmodel.DeskBoardLine{Value: line.Value.String(), Label: line.Label}
}

func chartToModel(metric *board.FilledMetric, chart *boardcatalog.Chart) *gqlmodel.DeskBoardChart {
	model := &gqlmodel.DeskBoardChart{
		Unit:     string(chart.Unit),
		Currency: chart.Currency,
		Ticks:    make([]*gqlmodel.DeskBoardTick, 0, len(chart.X)),
		Series:   make([]*gqlmodel.DeskBoardSeries, 0, len(chart.Series)),
		Marked:   chart.Marked,
		Target:   lineToModel(chart.Target),
		Ref:      lineToModel(chart.Ref),
		Ghost:    chart.Ghost,
		Rows:     make([]*gqlmodel.DeskBoardChartRow, 0, len(chart.Rows)),
	}
	if model.Marked == nil {
		model.Marked = []int{}
	}
	for _, tick := range chart.X {
		model.Ticks = append(model.Ticks, &gqlmodel.DeskBoardTick{
			At:     timestampOf(tick.At),
			Label:  tick.Label,
			Marked: tick.Marked,
		})
	}
	for idx := range chart.Series {
		series := &chart.Series[idx]
		points := make([]*string, 0, len(series.Points))
		for _, point := range series.Points {
			points = append(points, decimalText(point))
		}
		model.Series = append(model.Series, &gqlmodel.DeskBoardSeries{
			Name:   series.Name,
			Role:   string(series.Role),
			Points: points,
		})
	}
	if band := chart.Band; band != nil {
		model.Band = &gqlmodel.DeskBoardBand{
			From: band.From,
			To:   band.To,
			Low:  decimalText(band.Low),
			High: decimalText(band.High),
		}
	}
	for idx := range chart.Rows {
		model.Rows = append(model.Rows, chartRowToModel(metric, &chart.Rows[idx]))
	}

	return model
}

func chartRowToModel(
	metric *board.FilledMetric,
	row *boardcatalog.ChartRow,
) *gqlmodel.DeskBoardChartRow {
	model := &gqlmodel.DeskBoardChartRow{
		Label:  row.Label,
		Value:  valueToModel(row.Value),
		Marked: row.Marked,
		Rest:   row.Rest,
		Spans:  make([]*gqlmodel.DeskBoardSpan, 0, len(row.Spans)),
		Dots:   make([]*gqlmodel.DeskBoardChartMark, 0, len(row.Dots)),
	}
	if record := metric.Row(row.Ref); record != nil {
		model.Record = rowToModel(record)
	}
	for _, span := range row.Spans {
		model.Spans = append(model.Spans, &gqlmodel.DeskBoardSpan{
			From: int(span.From),
			To:   int(span.To),
			Kind: string(span.Kind),
		})
	}
	for idx := range row.Dots {
		dot := &row.Dots[idx]
		model.Dots = append(model.Dots, &gqlmodel.DeskBoardChartMark{
			At:     timestampOf(dot.At),
			Value:  decimalText(dot.Value),
			Marked: dot.Marked,
			Ghost:  dot.Ghost,
			Label:  dot.Label,
		})
	}

	return model
}

func splitToModel(split *boardcatalog.Split) *gqlmodel.DeskBoardSplit {
	model := &gqlmodel.DeskBoardSplit{
		Total: valueToModel(&split.Total),
		Parts: make([]*gqlmodel.DeskBoardSplitPart, 0, len(split.Parts)),
	}
	for idx := range split.Parts {
		part := &split.Parts[idx]
		model.Parts = append(model.Parts, &gqlmodel.DeskBoardSplitPart{
			Label: part.Label,
			Value: valueToModel(&part.Value),
			Kind:  string(part.Kind),
		})
	}

	return model
}

func gridToModel(grid *boardcatalog.Grid) *gqlmodel.DeskBoardGrid {
	model := &gqlmodel.DeskBoardGrid{
		Columns: make([]*gqlmodel.DeskBoardGridColumn, 0, len(grid.Columns)),
		Rows:    make([]*gqlmodel.DeskBoardGridRow, 0, len(grid.Rows)),
	}
	for _, column := range grid.Columns {
		model.Columns = append(model.Columns, &gqlmodel.DeskBoardGridColumn{
			Label:    column.Label,
			Sublabel: column.Sublabel,
		})
	}
	for idx := range grid.Rows {
		row := &grid.Rows[idx]
		cells := make([]*gqlmodel.DeskBoardGridCell, 0, len(row.Cells))
		for _, cell := range row.Cells {
			cells = append(cells, &gqlmodel.DeskBoardGridCell{
				Have:     cell.Have,
				Need:     cell.Need,
				NeedHigh: cell.NeedHigh,
				State:    string(cell.State),
			})
		}
		model.Rows = append(model.Rows, &gqlmodel.DeskBoardGridRow{Label: row.Label, Cells: cells})
	}

	return model
}

func tableToModel(table *boardcatalog.Table) *gqlmodel.DeskBoardMetricTable {
	model := &gqlmodel.DeskBoardMetricTable{
		Columns: make([]*gqlmodel.DeskBoardMetricColumn, 0, len(table.Columns)),
		Rows:    make([][]*gqlmodel.DeskBoardCell, 0, len(table.Rows)),
	}
	for _, column := range table.Columns {
		model.Columns = append(model.Columns, &gqlmodel.DeskBoardMetricColumn{
			Label: column.Label,
			Kind:  string(column.Kind),
		})
	}
	for _, row := range table.Rows {
		cells := make([]*gqlmodel.DeskBoardCell, 0, len(row))
		for idx := range row {
			cells = append(cells, &gqlmodel.DeskBoardCell{
				Value: valueToModel(row[idx].Value),
				Text:  row[idx].Text,
				Mark:  string(row[idx].Mark),
			})
		}
		model.Rows = append(model.Rows, cells)
	}

	return model
}
