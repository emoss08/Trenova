package reportvalue

import (
	"time"

	"github.com/emoss08/keel/analytics/table"
	"github.com/emoss08/keel/analytics/value"
)

func Cell(t value.Type, v *value.Vector, i int) any {
	if v.Null(i) {
		return nil
	}
	switch t {
	case value.String, value.Enum, value.Ref:
		return v.String[i]
	case value.Int, value.Epoch:
		return v.Int64[i]
	case value.Decimal:
		return v.Numeric[i].String()
	case value.Float:
		return v.Float64[i]
	case value.Bool:
		return v.Bool[i]
	case value.Time:
		return v.Time[i].UTC().Format(time.RFC3339)
	case value.JSON:
		return string(v.JSON[i])
	default:
		return nil
	}
}

func Native(t value.Type, v *value.Vector, i int) any {
	if v.Null(i) {
		return nil
	}
	switch t {
	case value.Decimal:
		return v.Numeric[i]
	case value.Time:
		return v.Time[i]
	case value.JSON:
		return v.JSON[i]
	case value.String, value.Enum, value.Ref, value.Int, value.Epoch, value.Float, value.Bool:
		return Cell(t, v, i)
	default:
		return Cell(t, v, i)
	}
}

func NativeRow(columns []table.Column, b *table.Batch, i int) []any {
	row := make([]any, len(columns))
	for c := range columns {
		row[c] = Native(columns[c].Type, &b.Columns[c], i)
	}
	return row
}

func Row(columns []table.Column, b *table.Batch, i int) []any {
	row := make([]any, len(columns))
	for c := range columns {
		row[c] = Cell(columns[c].Type, &b.Columns[c], i)
	}
	return row
}

func Float(t value.Type, v *value.Vector, i int) (float64, bool) {
	if v.Null(i) {
		return 0, false
	}
	switch t {
	case value.Int, value.Epoch:
		return float64(v.Int64[i]), true
	case value.Decimal:
		f, err := v.Numeric[i].Float64()
		return f, err == nil
	case value.Float:
		return v.Float64[i], true
	case value.String, value.Enum, value.Ref, value.Bool, value.Time, value.JSON:
		return 0, false
	default:
		return 0, false
	}
}

type Rows struct {
	Detail [][]any
	Levels []table.Level
	Total  []any
}

func TableRows(t *table.Table) Rows {
	var out Rows
	if t == nil {
		return out
	}
	out.Detail = make([][]any, 0, t.Len())
	for _, batch := range t.Batches {
		for i := range batch.Len {
			row := Row(t.Columns, batch, i)
			level := batch.Level(i)
			if level == table.Total {
				out.Total = row
				continue
			}
			out.Detail = append(out.Detail, row)
			out.Levels = append(out.Levels, level)
		}
	}
	return out
}
