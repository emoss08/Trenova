package boardmarkdown

import (
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func written(read *boardcatalog.MetricRead) string {
	var sb strings.Builder
	writeMetric(&sb, &board.FilledMetric{Read: read}, time.UTC)

	return sb.String()
}

func dec(value float64) *decimal.Decimal {
	number := decimal.NewFromFloat(value)

	return &number
}

func day(month time.Month, date int) int64 {
	return time.Date(2026, month, date, 12, 0, 0, 0, time.UTC).Unix()
}

func TestMetricMarkdownLabelsEveryPointOfATrendByItsDay(t *testing.T) {
	t.Parallel()

	two := 1
	text := written(&boardcatalog.MetricRead{Chart: &boardcatalog.Chart{
		Unit: boardcatalog.ValuePercent,
		X:    []boardcatalog.Tick{{At: day(time.October, 26)}, {At: day(time.November, 2)}},
		Series: []boardcatalog.Series{
			{Name: "On time", Points: []*decimal.Decimal{dec(0.94), dec(0.81)}},
		},
		Marked: []int{two},
		Ghost:  true,
		Target: &boardcatalog.Line{Value: decimal.NewFromFloat(0.9), Label: "Target"},
	}})

	assert.Contains(t, text, "| Point | On time |")
	assert.Contains(t, text, "| Oct 26, 2026 | 94% |")
	assert.Contains(t, text, "| Nov 2, 2026 (marked) (still running) | 81% |")
	assert.Contains(t, text, "Target: 90%")
}

func TestMetricMarkdownLabelsColumnsByTheirTicks(t *testing.T) {
	t.Parallel()

	text := written(&boardcatalog.MetricRead{Chart: &boardcatalog.Chart{
		Unit: boardcatalog.ValueHours,
		X:    []boardcatalog.Tick{{Label: "Mon"}, {Label: "Tue", Marked: true}},
		Series: []boardcatalog.Series{
			{Name: "Worked", Points: []*decimal.Decimal{dec(30600), nil}},
		},
		Ref: &boardcatalog.Line{Value: decimal.NewFromInt(28800), Label: "Shift"},
	}})

	assert.Contains(t, text, "| Mon | 8h 30m |")
	assert.Contains(t, text, "| Tue (marked) |  |")
	assert.Contains(t, text, "Shift: 8h")
}

func TestMetricMarkdownLabelsRankedRowsDotsAndSpans(t *testing.T) {
	t.Parallel()

	cents := boardcatalog.Money(1250000, "USD")
	text := written(&boardcatalog.MetricRead{Chart: &boardcatalog.Chart{
		Unit: boardcatalog.ValueNumber,
		Rows: []boardcatalog.ChartRow{
			{Label: "Northline", Value: &cents, Marked: true},
			{Label: "3 other customers", Value: &cents, Rest: true},
			{Label: "T-131", Dots: []boardcatalog.Dot{{Value: dec(4.1), Ghost: true}}},
			{
				Label: "Ridgeline Freight",
				Spans: []boardcatalog.Span{{
					From: day(
						time.November,
						1,
					), To: day(time.November, 29), Kind: boardcatalog.SpanCover,
				}},
				Dots: []boardcatalog.Dot{
					{At: day(time.December, 1), Marked: true, Label: "S-1 delivers"},
				},
			},
		},
	}})

	assert.Contains(t, text, "- Northline: $12,500.00 (marked)")
	assert.Contains(t, text, "- 3 other customers: $12,500.00 (the rest together)")
	assert.Contains(t, text, "- T-131; 4.1, too little data to place")
	assert.Contains(t, text, "- Ridgeline Freight; cover Nov 1, 2026 to Nov 29, 2026; "+
		"S-1 delivers, Dec 1, 2026, marked")
}

func TestMetricMarkdownLabelsSplitGridLogChecksAndTable(t *testing.T) {
	t.Parallel()

	count := boardcatalog.Count(4)
	money := boardcatalog.Money(42500, "USD")
	text := written(&boardcatalog.MetricRead{
		Split: &boardcatalog.Split{Total: money, Parts: []boardcatalog.Part{
			{Label: "Billed", Value: money, Kind: boardcatalog.PartUsed},
		}},
		Grid: &boardcatalog.Grid{
			Columns: []boardcatalog.GridHead{{Label: "Mon", Sublabel: "Nov 23"}},
			Rows: []boardcatalog.GridRow{{Label: "Dallas", Cells: []boardcatalog.Cell{
				{Have: 2, Need: 4, State: boardcatalog.CellShort},
			}}},
		},
		Log: []boardcatalog.LogEntry{{At: day(time.November, 23), Text: "997 accepted"}},
		Checks: []boardcatalog.Check{
			{Title: "Open entries", Tone: boardcatalog.ToneWarn, Count: &count},
		},
		Table: &boardcatalog.Table{
			Columns: []boardcatalog.Column{{Label: "Customer"}, {Label: "Over 90"}},
			Rows: [][]boardcatalog.TableCell{{
				{Text: "Peak"}, {Value: &money, Mark: boardcatalog.MarkHot},
			}, {
				{Text: "Acme"}, {},
			}},
		},
	})

	assert.Contains(t, text, "- Billed (used): $425.00")
	assert.Contains(t, text, "| Row | Mon Nov 23 |")
	assert.Contains(t, text, "| Dallas | 2 of 4 needed (short) |")
	assert.Contains(t, text, "- Nov 23, 12:00 PM: 997 accepted")
	assert.Contains(t, text, "- [warn] Open entries: 4")
	assert.Contains(t, text, "| Peak | $425.00 (marked) |")
	assert.Contains(t, text, "| Acme | – |")
}

func TestMetricMarkdownWritesEachFigureByItsLabel(t *testing.T) {
	t.Parallel()

	var sb strings.Builder
	value := boardcatalog.Money(123400, "USD")
	writeMetricFigures(&sb, &board.FilledMetric{
		Name: "receivables",
		Read: &boardcatalog.MetricRead{Figures: map[string]*boardcatalog.Value{
			"arTotalOpen": &value, "arOverdue": nil,
		}},
	})

	assert.Contains(t, sb.String(), ": $1,234.00")
	assert.Contains(t, sb.String(), ": could not be read")
}
