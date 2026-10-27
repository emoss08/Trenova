package analysis_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/analysis"
	"github.com/stretchr/testify/assert"
)

func TestMarkdownWritesWhatTheAnalysisShowedInItsOrder(t *testing.T) {
	t.Parallel()

	var shown analysis.Display
	shown.AddFigure(&analysis.Figure{
		Label: "Revenue", Value: 12517479.24, Kind: analysis.ValueMoney, Currency: "USD",
	})
	shown.AddFigure(&analysis.Figure{
		Label: "Top 10 share", Value: 0.1712, Kind: analysis.ValuePercent, Note: "of revenue",
	})
	shown.AddChart(&analysis.Chart{
		Kind:  analysis.ChartBar,
		Title: "Revenue by origin",
		X: analysis.ChartAxis{
			Label:  "Origin",
			Kind:   analysis.ValueText,
			Values: []any{"Chicago", "Norfolk"},
		},
		Series:    []analysis.ChartSeries{{Name: "Revenue", Values: []any{971712.0, 795297.5}}},
		ValueKind: analysis.ValueMoney,
	})
	shown.AddTable(&analysis.Table{
		Title: "Weakest lanes",
		Columns: []analysis.TableColumn{
			{Name: "Lane", Kind: analysis.ValueText},
			{Name: "Loads", Kind: analysis.ValueCount},
		},
		Rows: [][]any{{"Chicago | Jacksonville", 5.0}},
	})

	written := shown.Markdown("Lane performance, 90 days")

	assert.Equal(t, `## Lane performance, 90 days

- **Revenue:** $12,517,479.24
- **Top 10 share:** 17.1% (of revenue)

### Revenue by origin

| Origin | Revenue |
| --- | --- |
| Chicago | $971,712.00 |
| Norfolk | $795,297.50 |

### Weakest lanes

| Lane | Loads |
| --- | --- |
| Chicago \| Jacksonville | 5 |
`, written)
}

// A document holds the first rows of a long table and says in figures how
// many of its rows that is, so it reads the same in any language.
func TestMarkdownCutsALongTableAndSaysHowManyRowsItHolds(t *testing.T) {
	t.Parallel()

	rows := make([][]any, 0, 30)
	for idx := range 30 {
		rows = append(rows, []any{float64(idx)})
	}
	shown := analysis.Display{
		Tables: []analysis.Table{{
			Columns:   []analysis.TableColumn{{Name: "n", Kind: analysis.ValueNumber}},
			Rows:      rows,
			TotalRows: 1412,
		}},
	}

	written := shown.Markdown("Lanes")

	assert.Contains(t, written, "### (25 / 1,412)")
	assert.Equal(t, analysis.MaxMarkdownRows+2, strings.Count(written, "\n|"))
}

func TestFormatValueWritesEachKindAsTheCardDoes(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "-$1,250.50", analysis.FormatValue(-1250.5, analysis.ValueMoney, ""))
	assert.Equal(t, "1,250.50 EUR", analysis.FormatValue(1250.5, analysis.ValueMoney, "eur"))
	assert.Equal(t, "3,220", analysis.FormatValue(3220.0, analysis.ValueCount, ""))
	assert.Equal(t, "2.89", analysis.FormatValue(2.8912, analysis.ValueNumber, ""))
	assert.Equal(t, "12517479.24", analysis.FormatValue(12517479.24, analysis.ValueText, ""))
	assert.Equal(t, "2026-10-01", analysis.FormatValue("2026-10-01", analysis.ValueDate, ""))
	assert.Empty(t, analysis.FormatValue(nil, analysis.ValueMoney, ""))
}
