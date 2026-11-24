package metricquerytools

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardhint"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/shopspring/decimal"
)

const maxEntries = 40

const unreadNote = "A figure whose value is null could not be read: it is withheld from " +
	"this person or Trenova holds none for the period. Never call it zero."

type figureView struct {
	Name    string  `json:"name"`
	Label   string  `json:"label"`
	Value   *string `json:"value"`
	Was     string  `json:"was,omitempty"`
	WriteAs string  `json:"writeAs,omitempty"`
}

type pointView struct {
	Point   string   `json:"point"`
	Values  []string `json:"values"`
	Marked  bool     `json:"marked,omitempty"`
	Running bool     `json:"stillRunning,omitempty"`
}

type rowView struct {
	Label  string `json:"label"`
	Value  string `json:"value,omitempty"`
	Marked bool   `json:"marked,omitempty"`
	Rest   bool   `json:"theRestTogether,omitempty"`
	Detail string `json:"detail,omitempty"`
}

type checkView struct {
	Title  string `json:"title"`
	Tone   string `json:"tone,omitempty"`
	Count  string `json:"count,omitempty"`
	Detail string `json:"detail,omitempty"`
}

type tableView struct {
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows"`
}

type metricResult struct {
	Metric    string       `json:"metric"`
	Title     string       `json:"title"`
	Period    string       `json:"period,omitempty"`
	Source    string       `json:"source,omitempty"`
	Empty     bool         `json:"nothingInPeriod,omitempty"`
	Figures   []figureView `json:"figures"`
	Series    []string     `json:"series,omitempty"`
	Points    []pointView  `json:"points,omitempty"`
	Target    string       `json:"target,omitempty"`
	Reference string       `json:"reference,omitempty"`
	Rows      []rowView    `json:"rows,omitempty"`
	Checks    []checkView  `json:"checks,omitempty"`
	Table     *tableView   `json:"table,omitempty"`
	Cut       []string     `json:"cut,omitempty"`
	Notes     []string     `json:"notes,omitempty"`
	HowToShow string       `json:"howToShow"`
}

type resultInput struct {
	metric *boardcatalog.Metric
	ask    *ask
	block  *board.FilledBlock
	notes  []string
	loc    *time.Location
}

func resultOf(in *resultInput) *metricResult {
	read := in.block.Metric.Read
	out := &metricResult{
		Metric:    in.metric.Name,
		Title:     read.Title,
		Period:    in.ask.period,
		Source:    read.Source,
		Empty:     in.block.State == board.StateEmpty,
		Figures:   figuresOf(in.metric, read, in.loc),
		Notes:     in.notes,
		HowToShow: howToShow(in.metric, in.ask),
	}
	if hasUnread(out.Figures) {
		out.Notes = append(out.Notes, unreadNote)
	}
	out.chart(read.Chart, in.loc)
	out.split(read.Split)
	out.grid(read.Grid)
	out.log(read.Log, in.loc)
	out.checks(read.Checks)
	out.table(read.Table)

	return out
}

func hasUnread(figures []figureView) bool {
	for idx := range figures {
		if figures[idx].Value == nil {
			return true
		}
	}

	return false
}

func figuresOf(
	metric *boardcatalog.Metric,
	read *boardcatalog.MetricRead,
	loc *time.Location,
) []figureView {
	out := make([]figureView, 0, len(metric.Figures))
	for _, figure := range metric.Figures {
		view := figureView{Name: figure.Name, Label: figure.Label}
		if value := read.Figures[figure.Name]; value != nil {
			text := board.FormatValue(value)
			view.Value = &text
			view.WriteAs = "{" + figure.Name + ":" + metric.Name + "}"
			if shift, ok := read.Changes[figure.Name]; ok && shift.Was != nil {
				view.Was = board.FormatValue(shift.Was)
				if day := board.FormatDay(shift.WasAt, loc); day != "" {
					view.Was += " on " + day
				}
			}
		}
		out = append(out, view)
	}

	return out
}

func (r *metricResult) cut(what string, shown, total int) {
	if total > shown {
		r.Cut = append(r.Cut, fmt.Sprintf("%s: the first %d of %d", what, shown, total))
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

func pointLabel(tick *boardcatalog.Tick, loc *time.Location) string {
	if label := strings.TrimSpace(tick.Label); label != "" {
		return label
	}

	return board.FormatDay(tick.At, loc)
}

func (r *metricResult) chart(chart *boardcatalog.Chart, loc *time.Location) {
	if chart == nil {
		return
	}
	if len(chart.Series) > 0 {
		for idx := range chart.Series {
			r.Series = append(r.Series, chart.Series[idx].Name)
		}
		shown := min(len(chart.X), maxEntries)
		for point := range shown {
			view := pointView{
				Point:   pointLabel(&chart.X[point], loc),
				Values:  make([]string, 0, len(chart.Series)),
				Marked:  chart.X[point].Marked || slices.Contains(chart.Marked, point),
				Running: chart.Ghost && point == len(chart.X)-1,
			}
			for idx := range chart.Series {
				value := ""
				if points := chart.Series[idx].Points; point < len(points) {
					value = chartValue(chart, points[point])
				}
				if len(chart.Series) > 1 && value != "" {
					value = chart.Series[idx].Name + ": " + value
				}
				view.Values = append(view.Values, value)
			}
			r.Points = append(r.Points, view)
		}
		r.cut("points", shown, len(chart.X))
	}
	if chart.Target != nil {
		r.Target = chartValue(chart, &chart.Target.Value)
	}
	if chart.Ref != nil {
		r.Reference = strings.TrimSpace(chart.Ref.Label + " " + chartValue(chart, &chart.Ref.Value))
	}
	shown := min(len(chart.Rows), maxEntries)
	for idx := range shown {
		row := &chart.Rows[idx]
		view := rowView{Label: row.Label, Marked: row.Marked, Rest: row.Rest}
		if row.Value != nil {
			view.Value = board.FormatValue(row.Value)
		}
		view.Detail = rowDetail(chart, row, loc)
		r.Rows = append(r.Rows, view)
	}
	r.cut("rows", shown, len(chart.Rows))
}

func rowDetail(chart *boardcatalog.Chart, row *boardcatalog.ChartRow, loc *time.Location) string {
	parts := make([]string, 0, len(row.Spans)+len(row.Dots))
	for _, span := range row.Spans {
		parts = append(parts, fmt.Sprintf("%s %s to %s", span.Kind,
			board.FormatDay(span.From, loc), board.FormatDay(span.To, loc)))
	}
	for idx := range row.Dots {
		dot := &row.Dots[idx]
		words := make([]string, 0, 3)
		if label := strings.TrimSpace(dot.Label); label != "" {
			words = append(words, label)
		}
		if day := board.FormatDay(dot.At, loc); day != "" {
			words = append(words, day)
		}
		if dot.Value != nil {
			words = append(words, chartValue(chart, dot.Value))
		}
		if dot.Ghost {
			words = append(words, "too little data to place")
		}
		parts = append(parts, strings.Join(words, " "))
	}

	return strings.Join(parts, "; ")
}

func (r *metricResult) split(split *boardcatalog.Split) {
	if split == nil {
		return
	}
	r.Rows = append(r.Rows, rowView{Label: "Total", Value: board.FormatValue(&split.Total)})
	for idx := range split.Parts {
		part := &split.Parts[idx]
		r.Rows = append(r.Rows, rowView{
			Label:  part.Label,
			Value:  board.FormatValue(&part.Value),
			Detail: string(part.Kind),
		})
	}
}

func (r *metricResult) grid(grid *boardcatalog.Grid) {
	if grid == nil {
		return
	}
	view := &tableView{Columns: []string{""}}
	for idx := range grid.Columns {
		view.Columns = append(view.Columns,
			strings.TrimSpace(grid.Columns[idx].Label+" "+grid.Columns[idx].Sublabel))
	}
	shown := min(len(grid.Rows), maxEntries)
	for idx := range shown {
		row := &grid.Rows[idx]
		cells := []string{row.Label}
		for _, cell := range row.Cells {
			text := strconv.Itoa(cell.Have) + " of " + strconv.Itoa(cell.Need) + " needed"
			if cell.State != "" && cell.State != boardcatalog.CellCovered {
				text += " (" + string(cell.State) + ")"
			}
			cells = append(cells, text)
		}
		view.Rows = append(view.Rows, cells)
	}
	r.Table = view
	r.cut("rows", shown, len(grid.Rows))
}

func (r *metricResult) log(entries []boardcatalog.LogEntry, loc *time.Location) {
	shown := min(len(entries), maxEntries)
	for idx := range shown {
		entry := &entries[idx]
		r.Rows = append(r.Rows, rowView{
			Label:  board.FormatMoment(entry.At, loc),
			Value:  entry.Text,
			Detail: entry.Detail,
		})
	}
	r.cut("events", shown, len(entries))
}

func (r *metricResult) checks(checks []boardcatalog.Check) {
	shown := min(len(checks), maxEntries)
	for idx := range shown {
		check := &checks[idx]
		view := checkView{Title: check.Title, Tone: string(check.Tone), Detail: check.Detail}
		if check.Count != nil {
			view.Count = board.FormatValue(check.Count)
		}
		r.Checks = append(r.Checks, view)
	}
	r.cut("checks", shown, len(checks))
}

func (r *metricResult) table(table *boardcatalog.Table) {
	if table == nil {
		return
	}
	view := &tableView{Columns: make([]string, 0, len(table.Columns))}
	for idx := range table.Columns {
		view.Columns = append(view.Columns, table.Columns[idx].Label)
	}
	shown := min(len(table.Rows), maxEntries)
	for idx := range shown {
		cells := make([]string, 0, len(table.Rows[idx]))
		for _, cell := range table.Rows[idx] {
			text := cell.Text
			if cell.Value != nil {
				text = board.FormatValue(cell.Value)
			}
			if cell.Mark == boardcatalog.MarkHot || cell.Mark == boardcatalog.MarkW {
				text += " (marked)"
			}
			cells = append(cells, text)
		}
		view.Rows = append(view.Rows, cells)
	}
	r.Table = view
	r.cut("rows", shown, len(table.Rows))
}

func howToShow(metric *boardcatalog.Metric, a *ask) string {
	line := boardhint.Line(&boardhint.Show{Metric: metric.Name, For: a.forID, Period: a.period})
	if metric.Form == boardcatalog.FormFigures && metric.Records == nil {
		return "Say the figure the person asked for in a sentence, as it reads here. " + line
	}

	return line + " When the question needs more than one metric, they are groups of one " +
		"show_answer. Only a question one figure answers is a sentence, with the figure as it " +
		"reads here."
}
