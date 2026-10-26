package analysis

import (
	"math"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	MaxTables           = 6
	MaxTableColumns     = 20
	MaxTableRows        = 200
	MaxCellRunes        = 300
	MaxFigures          = 8
	MaxFigureLabelRunes = 80
	MaxFigureNoteRunes  = 200
	MaxColumnNameRunes  = 120
	MaxCharts           = 4
	MaxChartTitleRunes  = 200
	MaxChartSeries      = 8
	MaxChartPoints      = 500
	MaxSeriesNameRunes  = 80
	MaxAxisLabelRunes   = 80
)

// ValueKind is how a value is shown: the person reads money as money and a
// share as a percentage, whatever the script computed it as.
type ValueKind string

const (
	ValueText     = ValueKind("text")
	ValueNumber   = ValueKind("number")
	ValueMoney    = ValueKind("money")
	ValueCount    = ValueKind("count")
	ValuePercent  = ValueKind("percent")
	ValueDate     = ValueKind("date")
	ValueDateTime = ValueKind("datetime")
	ValueBool     = ValueKind("bool")
)

func AllValueKinds() []ValueKind {
	return []ValueKind{
		ValueText, ValueNumber, ValueMoney, ValueCount, ValuePercent, ValueDate, ValueDateTime,
		ValueBool,
	}
}

func (k ValueKind) IsValid() bool {
	return slices.Contains(AllValueKinds(), k)
}

// Or is the kind, or the fallback when it is not one.
func (k ValueKind) Or(fallback ValueKind) ValueKind {
	if k.IsValid() {
		return k
	}

	return fallback
}

// TableColumn is one column of a table a script showed.
type TableColumn struct {
	Name     string    `json:"name"`
	Kind     ValueKind `json:"kind"`
	Currency string    `json:"currency,omitempty"`
}

// Table is a table a script showed with show_table, kept as values so it is
// drawn as a table rather than as printed text. Percentages are fractions;
// dates are ISO 8601 text.
type Table struct {
	Title     string        `json:"title,omitempty"`
	Columns   []TableColumn `json:"columns"`
	Rows      [][]any       `json:"rows"`
	TotalRows int           `json:"totalRows"`
}

// Figure is one headline number a script showed with show_figure.
type Figure struct {
	Label    string    `json:"label"`
	Value    any       `json:"value"`
	Kind     ValueKind `json:"kind"`
	Currency string    `json:"currency,omitempty"`
	Note     string    `json:"note,omitempty"`
}

type ChartKind string

const (
	ChartBar     = ChartKind("bar")
	ChartLine    = ChartKind("line")
	ChartArea    = ChartKind("area")
	ChartScatter = ChartKind("scatter")
	ChartPie     = ChartKind("pie")
)

func AllChartKinds() []ChartKind {
	return []ChartKind{ChartBar, ChartLine, ChartArea, ChartScatter, ChartPie}
}

func (k ChartKind) IsValid() bool {
	return slices.Contains(AllChartKinds(), k)
}

// ChartAxis is the categories or values a chart is drawn along.
type ChartAxis struct {
	Label  string    `json:"label,omitempty"`
	Kind   ValueKind `json:"kind"`
	Values []any     `json:"values"`
}

// ChartSeries is one line, set of bars or slice set of a chart: a number, or
// nothing, for each point along its axis.
type ChartSeries struct {
	Name   string `json:"name"`
	Values []any  `json:"values"`
}

// Chart is a chart a script showed with show_chart, kept as what it plots so
// the client draws it in the person's theme rather than as an image.
type Chart struct {
	Kind       ChartKind     `json:"kind"`
	Title      string        `json:"title,omitempty"`
	X          ChartAxis     `json:"x"`
	Series     []ChartSeries `json:"series"`
	ValueKind  ValueKind     `json:"valueKind"`
	Currency   string        `json:"currency,omitempty"`
	Stacked    bool          `json:"stacked,omitempty"`
	Horizontal bool          `json:"horizontal,omitempty"`
}

type BlockKind string

const (
	BlockChart = BlockKind("chart")
	BlockTable = BlockKind("table")
)

// Block places one chart or table in the order the script showed it, by its
// index among the charts or the tables.
type Block struct {
	Kind  BlockKind `json:"kind"`
	Index int       `json:"index"`
}

// Display is everything a script showed the person, in the order it showed it.
type Display struct {
	Figures []Figure `json:"figures"`
	Tables  []Table  `json:"tables"`
	Charts  []Chart  `json:"charts"`
	Layout  []Block  `json:"layout"`
}

// Empty reports whether the script showed nothing.
func (d *Display) Empty() bool {
	return len(d.Figures)+len(d.Tables)+len(d.Charts) == 0
}

// AddFigure keeps a figure, unless the run has shown as many as it may.
func (d *Display) AddFigure(figure *Figure) bool {
	if len(d.Figures) >= MaxFigures {
		return false
	}
	bounded, ok := boundedFigure(figure)
	if !ok {
		return false
	}
	d.Figures = append(d.Figures, bounded)

	return true
}

// AddTable keeps a table, unless the run has shown as many as it may.
func (d *Display) AddTable(table *Table) bool {
	if len(d.Tables) >= MaxTables {
		return false
	}
	bounded := boundedTable(table)
	if len(bounded.Columns) == 0 {
		return false
	}
	d.Layout = append(d.Layout, Block{Kind: BlockTable, Index: len(d.Tables)})
	d.Tables = append(d.Tables, bounded)

	return true
}

// AddChart keeps a chart, unless the run has shown as many as it may.
func (d *Display) AddChart(chart *Chart) bool {
	if len(d.Charts) >= MaxCharts {
		return false
	}
	bounded, ok := boundedChart(chart)
	if !ok {
		return false
	}
	d.Layout = append(d.Layout, Block{Kind: BlockChart, Index: len(d.Charts)})
	d.Charts = append(d.Charts, bounded)

	return true
}

// Bound keeps what a stored display may hold, and a layout that points only
// at what is kept.
func (d *Display) Bound() {
	var bounded Display
	for idx := range d.Figures {
		bounded.AddFigure(&d.Figures[idx])
	}
	for _, block := range d.Layout {
		switch block.Kind {
		case BlockTable:
			if block.Index >= 0 && block.Index < len(d.Tables) {
				bounded.AddTable(&d.Tables[block.Index])
			}
		case BlockChart:
			if block.Index >= 0 && block.Index < len(d.Charts) {
				bounded.AddChart(&d.Charts[block.Index])
			}
		}
	}
	*d = bounded
	if d.Figures == nil {
		d.Figures = []Figure{}
	}
	if d.Tables == nil {
		d.Tables = []Table{}
	}
	if d.Charts == nil {
		d.Charts = []Chart{}
	}
	if d.Layout == nil {
		d.Layout = []Block{}
	}
}

func boundedFigure(raw *Figure) (Figure, bool) {
	label := stringutils.TruncateRunes(strings.TrimSpace(raw.Label), MaxFigureLabelRunes)
	if label == "" {
		return Figure{}, false
	}
	value := boundedCell(raw.Value)
	kind := raw.Kind.Or(kindOfValue(value))

	return Figure{
		Label:    label,
		Value:    value,
		Kind:     kind,
		Currency: Currency(kind, raw.Currency),
		Note: stringutils.TruncateRunes(strings.TrimSpace(raw.Note),
			MaxFigureNoteRunes),
	}, true
}

func boundedTable(raw *Table) Table {
	width := min(len(raw.Columns), MaxTableColumns)
	columns := make([]TableColumn, 0, width)
	height := min(len(raw.Rows), MaxTableRows)
	rows := make([][]any, 0, height)
	for _, row := range raw.Rows[:height] {
		cells := make([]any, width)
		for col := 0; col < width && col < len(row); col++ {
			cells[col] = boundedCell(row[col])
		}
		rows = append(rows, cells)
	}
	for col, column := range raw.Columns[:width] {
		kind := column.Kind.Or(kindOfColumn(rows, col))
		columns = append(columns, TableColumn{
			Name: stringutils.TruncateRunes(strings.TrimSpace(column.Name),
				MaxColumnNameRunes),
			Kind:     kind,
			Currency: Currency(kind, column.Currency),
		})
	}

	return Table{
		Title:     stringutils.TruncateRunes(strings.TrimSpace(raw.Title), MaxChartTitleRunes),
		Columns:   columns,
		Rows:      rows,
		TotalRows: max(raw.TotalRows, len(raw.Rows)),
	}
}

func boundedChart(raw *Chart) (Chart, bool) {
	if !raw.Kind.IsValid() || len(raw.Series) == 0 {
		return Chart{}, false
	}
	points := min(len(raw.X.Values), MaxChartPoints)
	if points == 0 {
		return Chart{}, false
	}
	x := make([]any, 0, points)
	for _, value := range raw.X.Values[:points] {
		x = append(x, boundedCell(value))
	}
	series := make([]ChartSeries, 0, min(len(raw.Series), MaxChartSeries))
	for _, each := range raw.Series[:min(len(raw.Series), MaxChartSeries)] {
		values := make([]any, points)
		for idx := 0; idx < points && idx < len(each.Values); idx++ {
			values[idx] = numberOrNil(each.Values[idx])
		}
		series = append(series, ChartSeries{
			Name:   stringutils.TruncateRunes(strings.TrimSpace(each.Name), MaxSeriesNameRunes),
			Values: values,
		})
	}
	valueKind := raw.ValueKind.Or(ValueNumber)

	return Chart{
		Kind:  raw.Kind,
		Title: stringutils.TruncateRunes(strings.TrimSpace(raw.Title), MaxChartTitleRunes),
		X: ChartAxis{
			Label:  stringutils.TruncateRunes(strings.TrimSpace(raw.X.Label), MaxAxisLabelRunes),
			Kind:   raw.X.Kind.Or(kindOfAxis(x)),
			Values: x,
		},
		Series:     series,
		ValueKind:  valueKind,
		Currency:   Currency(valueKind, raw.Currency),
		Stacked:    raw.Stacked,
		Horizontal: raw.Horizontal,
	}, true
}

// boundedCell keeps a value as one a table can show: a finite number, text, a
// flag or nothing. Anything else is shown as nothing.
func boundedCell(value any) any {
	switch typed := value.(type) {
	case nil, bool:
		return typed
	case float64:
		if math.IsNaN(typed) || math.IsInf(typed, 0) {
			return nil
		}
		return typed
	case int:
		return typed
	case int64:
		return typed
	case string:
		return stringutils.TruncateRunes(typed, MaxCellRunes)
	default:
		return nil
	}
}

func numberOrNil(value any) any {
	switch typed := boundedCell(value).(type) {
	case float64, int, int64:
		return typed
	default:
		return nil
	}
}

// kindOfColumn is a column's kind read off its values when the script named
// none: numbers are numbers, ISO dates are dates, flags are flags.
func kindOfColumn(rows [][]any, col int) ValueKind {
	kind := ValueKind("")
	for _, row := range rows {
		if col >= len(row) || row[col] == nil {
			continue
		}
		cell := kindOfValue(row[col])
		if kind == "" {
			kind = cell
			continue
		}
		if kind != cell {
			return ValueText
		}
	}

	return kind.Or(ValueText)
}

func kindOfAxis(values []any) ValueKind {
	return kindOfColumn(rowsOf(values), 0)
}

func rowsOf(values []any) [][]any {
	rows := make([][]any, 0, len(values))
	for _, value := range values {
		rows = append(rows, []any{value})
	}

	return rows
}

func kindOfValue(value any) ValueKind {
	switch typed := value.(type) {
	case bool:
		return ValueBool
	case float64, int, int64:
		return ValueNumber
	case string:
		if _, err := time.Parse(time.DateOnly, typed); err == nil {
			return ValueDate
		}
		if isDateTime(typed) {
			return ValueDateTime
		}
		return ValueText
	default:
		return ValueText
	}
}

// Currency is the currency a value is shown in: only money has one. A
// report's display carries the organization's currency on every column, and
// handed on as it was it tagged customer names and invoice counts as USD.
func Currency(kind ValueKind, raw string) string {
	if kind != ValueMoney {
		return ""
	}
	code := strings.ToUpper(strings.TrimSpace(raw))
	if len(code) != 3 {
		return ""
	}
	for _, char := range code {
		if char < 'A' || char > 'Z' {
			return ""
		}
	}

	return code
}

// KindOf is how a report column's values are shown, from the column's type
// and the format the report gives it.
func KindOf(fieldType, format, dateStyle string) ValueKind {
	switch format {
	case "money":
		return ValueMoney
	case "percent":
		return ValuePercent
	case "count":
		return ValueCount
	}

	switch fieldType {
	case "int", "decimal", "float":
		return ValueNumber
	case "epoch":
		if dateStyle == "date" {
			return ValueDate
		}
		return ValueDateTime
	case "bool":
		return ValueBool
	default:
		return ValueText
	}
}

// isDateTime reports ISO 8601 date and time text, with or without an offset,
// as Python's isoformat writes it.
func isDateTime(text string) bool {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.999999999"} {
		if _, err := time.Parse(layout, text); err == nil {
			return true
		}
	}

	return false
}
