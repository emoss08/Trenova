package reportfmt

import (
	"context"
	"strconv"
	"time"

	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/pkg/rules"
)

const BatchRows = 500

type Column struct {
	Display *Resolved
	Type    reportcatalog.FieldType
}

type Cell struct {
	Text string
	Tone Tone
}

type Row struct {
	Values []any
	Cells  []Cell
}

type Batch struct {
	columns []Column
	loc     *time.Location
	rows    [][]any
}

func NewBatch(columns []Column, loc *time.Location) *Batch {
	return &Batch{
		columns: columns,
		loc:     location(loc),
		rows:    make([][]any, 0, BatchRows),
	}
}

func (b *Batch) Add(row []any) {
	b.rows = append(b.rows, append([]any(nil), row...))
}

func (b *Batch) Len() int {
	return len(b.rows)
}

func (b *Batch) Full() bool {
	return len(b.rows) >= BatchRows
}

func (b *Batch) Format(ctx context.Context) ([]Row, error) {
	if len(b.rows) == 0 {
		return nil, nil
	}
	rows := b.rows
	b.rows = make([][]any, 0, BatchRows)

	out := make([]Row, len(rows))
	for i := range out {
		out[i] = Row{Values: rows[i], Cells: make([]Cell, len(b.columns))}
	}
	if len(b.columns) == 0 {
		return out, nil
	}

	request := make([]rules.ReportColumn, len(b.columns))
	for c := range b.columns {
		request[c] = rules.ReportColumn{
			Display:   b.columns[c].Display.wire(),
			ValueType: string(b.columns[c].Type),
			Values:    make(rules.ReportValues, len(rows)),
		}
		for r, row := range rows {
			value := cellValue(row, c)
			if _, isTime := value.(time.Time); isTime {
				continue
			}
			request[c].Values[r] = value
		}
	}

	formatted, err := rules.FormatReport(ctx, request)
	if err != nil {
		return nil, err
	}

	for c := range b.columns {
		display := b.columns[c].Display
		for r, row := range rows {
			if instant, isTime := cellValue(row, c).(time.Time); isTime {
				out[r].Cells[c] = Cell{Text: display.instant(instant, b.loc)}
				continue
			}
			cell := formatted[c][r]
			text := cell.Text
			if cell.Date.Valid {
				text = display.date(time.Unix(cell.Date.Decimal.IntPart(), 0).In(b.loc))
			}
			out[r].Cells[c] = Cell{Text: text, Tone: Tone(cell.Tone)}
		}
	}

	return out, nil
}

func FormatRow(
	ctx context.Context,
	columns []Column,
	row []any,
	loc *time.Location,
) ([]Cell, error) {
	batch := NewBatch(columns, loc)
	batch.Add(row)
	rows, err := batch.Format(ctx)
	if err != nil {
		return nil, err
	}

	return rows[0].Cells, nil
}

func cellValue(row []any, index int) any {
	if index < len(row) {
		return row[index]
	}

	return nil
}

func location(loc *time.Location) *time.Location {
	if loc == nil {
		return time.UTC
	}
	return loc
}

func (r *Resolved) instant(t time.Time, loc *time.Location) string {
	if r == nil || r.Style == StyleAuto {
		return t.Format(time.RFC3339)
	}

	return r.date(t.In(loc))
}

func (r *Resolved) date(t time.Time) string {
	switch r.DateStyle {
	case DateStyleDate:
		return t.Format("2006-01-02")
	case DateStyleDateLong:
		return t.Format("Jan 2, 2006")
	case DateStyleTime:
		return t.Format("15:04")
	case DateStyleMonthYear:
		return t.Format("Jan 2006")
	case DateStyleQuarter:
		return "Q" + strconv.Itoa((int(t.Month())-1)/3+1) + " " + t.Format("2006")
	case DateStyleYear:
		return t.Format("2006")
	case DateStyleWeekday:
		return t.Format("Mon 2006-01-02")
	case DateStyleISO:
		return t.Format(time.RFC3339)
	case DateStyleAuto, DateStyleDateTime:
		return t.Format("2006-01-02 15:04")
	default:
		return t.Format("2006-01-02 15:04")
	}
}
