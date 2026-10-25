package render

import (
	"github.com/emoss08/keel/analytics/export"
	"github.com/emoss08/keel/analytics/format"
	"github.com/emoss08/keel/analytics/table"
)

func Formatters(columns []table.Column, opts format.Options) []*format.Formatter {
	out := make([]*format.Formatter, len(columns))
	for i := range columns {
		out[i] = format.New(columns[i].Type, columns[i].Display, opts)
	}
	return out
}

type Cell struct {
	Text string
	Tone string
}

func FormatRow(formatters []*format.Formatter, b *table.Batch, row int, label string) []Cell {
	cells := make([]Cell, len(formatters))
	for c, f := range formatters {
		text, tone := f.Append(nil, &b.Columns[c], row)
		cells[c] = Cell{Text: string(text), Tone: string(tone)}
	}
	if label != "" && len(cells) > 0 && b.Columns[0].Null(row) {
		cells[0].Text = label
	}
	return cells
}

func levelLabel(text *export.Text, level table.Level) string {
	switch {
	case level == table.Total:
		return text.Total
	case level.IsSubtotal():
		return text.Subtotal
	default:
		return ""
	}
}
