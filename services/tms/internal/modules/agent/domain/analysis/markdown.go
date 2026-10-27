package analysis

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/shared/floatutils"
	"github.com/emoss08/trenova/shared/intutils"
)

// MaxMarkdownRows is the most rows of one table or chart an analysis puts in
// a document; the analysis itself keeps every row it showed.
const MaxMarkdownRows = 25

const (
	moneyDecimals  = 2
	numberDecimals = 2
	percentDigits  = 1
	usd            = "USD"
)

// Markdown is what an analysis showed, written into a document under a
// heading of its title: the figures as a list, then each table and chart in
// the order the script showed them, a chart as the table of what it plots.
// The values are the ones the analysis kept, so nothing is copied by hand.
func (d *Display) Markdown(title string) string {
	var sb strings.Builder
	sb.WriteString("## ")
	sb.WriteString(cell(title))
	sb.WriteString("\n")
	if len(d.Figures) > 0 {
		sb.WriteString("\n")
		for idx := range d.Figures {
			writeFigure(&sb, &d.Figures[idx])
		}
	}
	for _, block := range d.layout() {
		switch block.Kind {
		case BlockTable:
			if block.Index < len(d.Tables) {
				writeTable(&sb, &d.Tables[block.Index])
			}
		case BlockChart:
			if block.Index < len(d.Charts) {
				writeChart(&sb, &d.Charts[block.Index])
			}
		}
	}

	return sb.String()
}

func (t *Table) Markdown() string {
	var sb strings.Builder
	writeTable(&sb, t)

	return sb.String()
}

func (c *Chart) Markdown() string {
	var sb strings.Builder
	writeChart(&sb, c)

	return sb.String()
}

// layout is the order to write the tables and charts in: the one the script
// showed them in, or tables then charts for a display kept without it.
func (d *Display) layout() []Block {
	if len(d.Layout) > 0 {
		return d.Layout
	}
	blocks := make([]Block, 0, len(d.Tables)+len(d.Charts))
	for idx := range d.Tables {
		blocks = append(blocks, Block{Kind: BlockTable, Index: idx})
	}
	for idx := range d.Charts {
		blocks = append(blocks, Block{Kind: BlockChart, Index: idx})
	}

	return blocks
}

func writeFigure(sb *strings.Builder, figure *Figure) {
	fmt.Fprintf(sb, "- **%s:** %s", cell(figure.Label),
		FormatValue(figure.Value, figure.Kind, figure.Currency))
	if note := strings.TrimSpace(figure.Note); note != "" {
		fmt.Fprintf(sb, " (%s)", cell(note))
	}
	sb.WriteString("\n")
}

func writeTable(sb *strings.Builder, table *Table) {
	header := make([]string, 0, len(table.Columns))
	for _, column := range table.Columns {
		header = append(header, column.Name)
	}
	rows := make([][]string, 0, min(len(table.Rows), MaxMarkdownRows))
	for _, row := range table.Rows[:min(len(table.Rows), MaxMarkdownRows)] {
		cells := make([]string, 0, len(table.Columns))
		for idx, column := range table.Columns {
			var value any
			if idx < len(row) {
				value = row[idx]
			}
			cells = append(cells, FormatValue(value, column.Kind, column.Currency))
		}
		rows = append(rows, cells)
	}
	writeGrid(sb, table.Title, header, rows, max(table.TotalRows, len(table.Rows)))
}

func writeChart(sb *strings.Builder, chart *Chart) {
	header := make([]string, 0, len(chart.Series)+1)
	header = append(header, chart.X.Label)
	for _, series := range chart.Series {
		header = append(header, series.Name)
	}
	points := len(chart.X.Values)
	rows := make([][]string, 0, min(points, MaxMarkdownRows))
	for point := range min(points, MaxMarkdownRows) {
		cells := make([]string, 0, len(header))
		cells = append(cells, FormatValue(chart.X.Values[point], chart.X.Kind, ""))
		for _, series := range chart.Series {
			var value any
			if point < len(series.Values) {
				value = series.Values[point]
			}
			cells = append(cells, FormatValue(value, chart.ValueKind, chart.Currency))
		}
		rows = append(rows, cells)
	}
	writeGrid(sb, chart.Title, header, rows, points)
}

// writeGrid writes one markdown table under its title. A table cut short says
// how many of its rows it holds, in figures alone, so the document reads the
// same in any language.
func writeGrid(sb *strings.Builder, title string, header []string, rows [][]string, total int) {
	sb.WriteString("\n")
	heading := cell(title)
	if len(rows) < total {
		heading = strings.TrimSpace(fmt.Sprintf("%s (%s / %s)", heading,
			intutils.FormatWithCommas(int64(len(rows))),
			intutils.FormatWithCommas(int64(total))))
	}
	if heading != "" {
		sb.WriteString("### ")
		sb.WriteString(heading)
		sb.WriteString("\n\n")
	}
	writeRow(sb, header)
	sb.WriteString("|")
	for range header {
		sb.WriteString(" --- |")
	}
	sb.WriteString("\n")
	for _, row := range rows {
		writeRow(sb, row)
	}
}

func writeRow(sb *strings.Builder, cells []string) {
	sb.WriteString("|")
	for _, value := range cells {
		sb.WriteString(" ")
		sb.WriteString(cell(value))
		sb.WriteString(" |")
	}
	sb.WriteString("\n")
}

// cell keeps a value on one line of a markdown table.
func cell(text string) string {
	text = strings.Join(strings.Fields(text), " ")

	return strings.ReplaceAll(text, "|", `\|`)
}

// FormatValue writes a value the way the analysis card shows it: money with
// its currency and cents, a count whole, a percent from its fraction, and a
// number grouped. A value of another type is written as it is.
func FormatValue(value any, kind ValueKind, currency string) string {
	number, isNumber := numberOf(value)
	if !isNumber {
		if value == nil {
			return ""
		}
		if flag, isFlag := value.(bool); isFlag {
			return strconv.FormatBool(flag)
		}

		return fmt.Sprint(value)
	}

	switch kind {
	case ValueMoney:
		return money(number, currency)
	case ValueCount:
		return floatutils.FormatGrouped(number, 0)
	case ValuePercent:
		return floatutils.FormatFractionAsPercent(number, percentDigits)
	case ValueText, ValueDate, ValueDateTime, ValueBool:
		return strconv.FormatFloat(number, 'f', -1, 64)
	default:
		return floatutils.FormatGrouped(number, numberDecimals)
	}
}

func money(amount float64, currency string) string {
	text := floatutils.FormatGrouped(amount, moneyDecimals)
	whole, fraction, _ := strings.Cut(text, ".")
	text = whole + "." + fraction + strings.Repeat("0", moneyDecimals-len(fraction))
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" || currency == usd {
		if negative, ok := strings.CutPrefix(text, "-"); ok {
			return "-$" + negative
		}

		return "$" + text
	}

	return text + " " + currency
}

func numberOf(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	default:
		return 0, false
	}
}
