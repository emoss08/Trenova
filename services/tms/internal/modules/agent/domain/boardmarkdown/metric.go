package boardmarkdown

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardregistry"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/shopspring/decimal"
)

func writeMetric(sb *strings.Builder, metric *board.FilledMetric, location *time.Location) {
	if metric == nil || metric.Read == nil {
		return
	}
	read := metric.Read
	if source := oneLine(read.Source); source != "" {
		sb.WriteString("Read from ")
		sb.WriteString(source)
		sb.WriteString(".\n\n")
	}
	writeMetricFigures(sb, metric)
	writeChart(sb, read.Chart, location)
	writeSplit(sb, read.Split)
	writeGrid(sb, read.Grid)
	for idx := range read.Log {
		entry := &read.Log[idx]
		fmt.Fprintf(sb, "- %s: %s", board.FormatMoment(entry.At, location), oneLine(entry.Text))
		if detail := oneLine(entry.Detail); detail != "" {
			sb.WriteString(" (" + detail + ")")
		}
		sb.WriteString("\n")
	}
	for idx := range read.Checks {
		check := &read.Checks[idx]
		fmt.Fprintf(sb, "- [%s] %s", check.Tone, oneLine(check.Title))
		if check.Count != nil {
			sb.WriteString(": " + board.FormatValue(check.Count))
		}
		if detail := oneLine(check.Detail); detail != "" {
			sb.WriteString(" (" + detail + ")")
		}
		sb.WriteString("\n")
	}
	writeMetricTable(sb, read.Table)
}

func writeMetricFigures(sb *strings.Builder, metric *board.FilledMetric) {
	spec, known := boardregistry.Catalog().MetricNamed(metric.Name)
	if !known {
		return
	}
	for _, figure := range spec.Figures {
		value, read := metric.Read.Figures[figure.Name]
		if !read {
			continue
		}
		fmt.Fprintf(sb, "- %s: ", figure.Label)
		if value == nil {
			sb.WriteString("could not be read\n")
			continue
		}
		sb.WriteString(board.FormatValue(value))
		sb.WriteString("\n")
	}
}

func chartValue(chart *boardcatalog.Chart, number *decimal.Decimal) string {
	if number == nil {
		return ""
	}

	return board.FormatValue(&boardcatalog.Value{
		Kind:     chart.Unit,
		Number:   *number,
		Currency: chart.Currency,
	})
}

func tickLabel(tick *boardcatalog.Tick, location *time.Location) string {
	if label := oneLine(tick.Label); label != "" {
		return label
	}

	return board.FormatDay(tick.At, location)
}

func writeChart(sb *strings.Builder, chart *boardcatalog.Chart, location *time.Location) {
	if chart == nil {
		return
	}
	if len(chart.X) > 0 && len(chart.Series) > 0 {
		header := []string{"Point"}
		for idx := range chart.Series {
			header = append(header, chart.Series[idx].Name)
		}
		writeRow(sb, header)
		writeRule(sb, len(header))
		for point := range chart.X {
			label := tickLabel(&chart.X[point], location)
			if chart.X[point].Marked || slices.Contains(chart.Marked, point) {
				label += " (marked)"
			}
			if chart.Ghost && point == len(chart.X)-1 {
				label += " (still running)"
			}
			cells := []string{label}
			for idx := range chart.Series {
				points := chart.Series[idx].Points
				value := ""
				if point < len(points) {
					value = chartValue(chart, points[point])
				}
				cells = append(cells, value)
			}
			writeRow(sb, cells)
		}
	}
	for idx := range chart.Rows {
		writeChartRow(sb, chart, &chart.Rows[idx], location)
	}
	if chart.Band != nil && chart.Band.From != nil && chart.Band.To != nil &&
		*chart.Band.From < len(chart.X) && *chart.Band.To < len(chart.X) {
		fmt.Fprintf(
			sb,
			"\nMarked span: %s to %s\n",
			tickLabel(&chart.X[*chart.Band.From], location),
			tickLabel(&chart.X[*chart.Band.To], location),
		)
	}
	if chart.Target != nil {
		fmt.Fprintf(sb, "\n%s: %s\n", lineLabel(chart.Target, "Target"),
			chartValue(chart, &chart.Target.Value))
	}
	if chart.Ref != nil {
		fmt.Fprintf(sb, "\n%s: %s\n", lineLabel(chart.Ref, "Reference"),
			chartValue(chart, &chart.Ref.Value))
	}
}

func lineLabel(line *boardcatalog.Line, fallback string) string {
	if label := oneLine(line.Label); label != "" {
		return label
	}

	return fallback
}

func writeChartRow(
	sb *strings.Builder,
	chart *boardcatalog.Chart,
	row *boardcatalog.ChartRow,
	location *time.Location,
) {
	sb.WriteString("- " + oneLine(row.Label))
	if row.Value != nil {
		sb.WriteString(": " + board.FormatValue(row.Value))
	}
	if row.Marked {
		sb.WriteString(" (marked)")
	}
	if row.Rest {
		sb.WriteString(" (the rest together)")
	}
	for _, span := range row.Spans {
		fmt.Fprintf(sb, "; %s %s to %s", span.Kind, board.FormatDay(span.From, location),
			board.FormatDay(span.To, location))
	}
	for idx := range row.Dots {
		dot := &row.Dots[idx]
		parts := make([]string, 0, 3)
		if label := oneLine(dot.Label); label != "" {
			parts = append(parts, label)
		}
		if dot.At > 0 {
			parts = append(parts, board.FormatDay(dot.At, location))
		}
		if dot.Value != nil {
			parts = append(parts, chartValue(chart, dot.Value))
		}
		if dot.Ghost {
			parts = append(parts, "too little data to place")
		} else if dot.Marked {
			parts = append(parts, "marked")
		}
		sb.WriteString("; " + strings.Join(parts, ", "))
	}
	sb.WriteString("\n")
}

func writeSplit(sb *strings.Builder, split *boardcatalog.Split) {
	if split == nil {
		return
	}
	fmt.Fprintf(sb, "Total: %s\n", board.FormatValue(&split.Total))
	for idx := range split.Parts {
		part := &split.Parts[idx]
		fmt.Fprintf(sb, "- %s (%s): %s\n", oneLine(part.Label), part.Kind,
			board.FormatValue(&part.Value))
	}
}

func writeGrid(sb *strings.Builder, grid *boardcatalog.Grid) {
	if grid == nil {
		return
	}
	header := []string{"Row"}
	for idx := range grid.Columns {
		label := oneLine(grid.Columns[idx].Label)
		if sub := oneLine(grid.Columns[idx].Sublabel); sub != "" {
			label += " " + sub
		}
		header = append(header, label)
	}
	writeRow(sb, header)
	writeRule(sb, len(header))
	for idx := range grid.Rows {
		row := &grid.Rows[idx]
		cells := []string{row.Label}
		for _, cell := range row.Cells {
			text := strconv.Itoa(cell.Have) + " of " + strconv.Itoa(cell.Need) + " needed"
			if cell.State != "" && cell.State != boardcatalog.CellCovered {
				text += " (" + string(cell.State) + ")"
			}
			cells = append(cells, text)
		}
		writeRow(sb, cells)
	}
}

func writeMetricTable(sb *strings.Builder, table *boardcatalog.Table) {
	if table == nil {
		return
	}
	header := make([]string, 0, len(table.Columns))
	for idx := range table.Columns {
		header = append(header, table.Columns[idx].Label)
	}
	writeRow(sb, header)
	writeRule(sb, len(header))
	for _, row := range table.Rows {
		cells := make([]string, 0, len(row))
		for idx := range row {
			text := row[idx].Text
			if row[idx].Value != nil {
				text = board.FormatValue(row[idx].Value)
			}
			if text == "" {
				text = "–"
			}
			if row[idx].Mark == boardcatalog.MarkHot || row[idx].Mark == boardcatalog.MarkW {
				text += " (marked)"
			}
			cells = append(cells, text)
		}
		writeRow(sb, cells)
	}
}
