package agentextension_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBoundedTablesKeepWhatATableCanShow(t *testing.T) {
	t.Parallel()

	tables := agentextension.BoundedTables([]agentextension.AnalysisTable{
		{
			Title: "  Open balance  ",
			Columns: []agentextension.AnalysisTableColumn{
				{Name: "Customer", Kind: "text", Currency: "USD"},
				{Name: "Open Balance", Kind: "money", Currency: "usd"},
				{Name: "Share", Kind: "share"},
			},
			Rows: [][]any{
				{"Acme", 36912.16, 0.006},
				{strings.Repeat("x", agentextension.MaxAnalysisCellRunes+10), nil},
				{"Peak", map[string]any{"nested": true}, true},
			},
			TotalRows: 2,
		},
		{Columns: nil, Rows: [][]any{{"lost"}}},
	})

	require.Len(t, tables, 1, "a table without columns is left out")
	table := tables[0]
	assert.Equal(t, "Open balance", table.Title)
	assert.Equal(t, []agentextension.AnalysisTableColumn{
		{Name: "Customer", Kind: agentextension.AnalysisValueText},
		{Name: "Open Balance", Kind: agentextension.AnalysisValueMoney, Currency: "USD"},
		{Name: "Share", Kind: agentextension.AnalysisValueText},
	}, table.Columns, "an unknown kind reads as text, a currency is its code, and only "+
		"money keeps one")
	assert.Equal(t, []any{"Acme", 36912.16, 0.006}, table.Rows[0])
	assert.Len(t, []rune(table.Rows[1][0].(string)), agentextension.MaxAnalysisCellRunes)
	assert.Nil(t, table.Rows[1][2], "a short row is padded to the table's width")
	assert.Nil(t, table.Rows[2][1], "a value no table can show is left empty")
	assert.Equal(t, 3, table.TotalRows, "never fewer rows than it holds")
}

func TestBoundedTablesCapHowMuchARunKeeps(t *testing.T) {
	t.Parallel()

	columns := make([]agentextension.AnalysisTableColumn, agentextension.MaxAnalysisTableColumns+5)
	for idx := range columns {
		columns[idx] = agentextension.AnalysisTableColumn{Name: "c", Kind: "number"}
	}
	rows := make([][]any, agentextension.MaxAnalysisTableRows+5)
	raw := make([]agentextension.AnalysisTable, agentextension.MaxAnalysisTables+2)
	for idx := range raw {
		raw[idx] = agentextension.AnalysisTable{Columns: columns, Rows: rows}
	}

	tables := agentextension.BoundedTables(raw)
	require.Len(t, tables, agentextension.MaxAnalysisTables)
	assert.Len(t, tables[0].Columns, agentextension.MaxAnalysisTableColumns)
	assert.Len(t, tables[0].Rows, agentextension.MaxAnalysisTableRows)
	assert.Equal(t, agentextension.MaxAnalysisTableRows+5, tables[0].TotalRows)
}

func TestBoundedFiguresNeedALabel(t *testing.T) {
	t.Parallel()

	figures := agentextension.BoundedFigures([]agentextension.AnalysisFigure{
		{Label: " Total open ", Value: 6045662.16, Kind: "money", Currency: "dollars"},
		{Label: "  ", Value: 1.0},
		{Label: "Customers", Value: 3.0, Kind: "made-up", Note: " with open invoices "},
	})

	assert.Equal(t, []agentextension.AnalysisFigure{
		{Label: "Total open", Value: 6045662.16, Kind: agentextension.AnalysisValueMoney},
		{
			Label: "Customers",
			Value: 3.0,
			Kind:  agentextension.AnalysisValueNumber,
			Note:  "with open invoices",
		},
	}, figures)
}

func TestBoundedLayoutPointsOnlyAtWhatWasKept(t *testing.T) {
	t.Parallel()

	layout := agentextension.BoundedLayout([]agentextension.AnalysisBlock{
		{Kind: agentextension.AnalysisBlockTable, Index: 0},
		{Kind: agentextension.AnalysisBlockChart, Index: 1},
		{Kind: agentextension.AnalysisBlockChart, Index: 2},
		{Kind: agentextension.AnalysisBlockTable, Index: 1},
		{Kind: "map", Index: 0},
	}, 1, 1)

	assert.Equal(t, []agentextension.AnalysisBlock{
		{Kind: agentextension.AnalysisBlockTable, Index: 0},
		{Kind: agentextension.AnalysisBlockChart, Index: 1},
	}, layout)
}

// A report's display carries the organization's currency on every column; a
// run of AR aging handed it on and tagged customer names and invoice counts USD.
func TestAnalysisCurrencyIsOnlyMoneys(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		"USD",
		agentextension.AnalysisCurrency(agentextension.AnalysisValueMoney, " usd "),
	)
	assert.Empty(t, agentextension.AnalysisCurrency(agentextension.AnalysisValueMoney, "dollars"))
	for _, kind := range []agentextension.AnalysisValueKind{
		agentextension.AnalysisValueText,
		agentextension.AnalysisValueCount,
		agentextension.AnalysisValuePercent,
		agentextension.AnalysisValueNumber,
		agentextension.AnalysisValueDateTime,
	} {
		assert.Empty(t, agentextension.AnalysisCurrency(kind, "USD"), string(kind))
	}

	figures := agentextension.BoundedFigures([]agentextension.AnalysisFigure{
		{
			Label:    "Open invoices",
			Value:    36,
			Kind:     agentextension.AnalysisValueCount,
			Currency: "USD",
		},
		{
			Label:    "Open balance",
			Value:    45662.16,
			Kind:     agentextension.AnalysisValueMoney,
			Currency: "usd",
		},
	})
	require.Len(t, figures, 2)
	assert.Empty(t, figures[0].Currency)
	assert.Equal(t, "USD", figures[1].Currency)
}

func TestAnalysisKindOfReadsTheReportsFormatFirst(t *testing.T) {
	t.Parallel()

	assert.Equal(t, agentextension.AnalysisValueMoney,
		agentextension.AnalysisKindOf("decimal", "money", ""))
	assert.Equal(t, agentextension.AnalysisValuePercent,
		agentextension.AnalysisKindOf("decimal", "percent", ""))
	assert.Equal(t, agentextension.AnalysisValueCount,
		agentextension.AnalysisKindOf("int", "count", ""))
	assert.Equal(t, agentextension.AnalysisValueDate,
		agentextension.AnalysisKindOf("epoch", "", "date"))
	assert.Equal(t, agentextension.AnalysisValueDateTime,
		agentextension.AnalysisKindOf("epoch", "", "dateTime"))
	assert.Equal(t, agentextension.AnalysisValueText,
		agentextension.AnalysisKindOf("enum", "", ""))
}

func TestNormalizeAnalysisTitleFallsBackToThePurpose(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Open AR by customer",
		agentextension.NormalizeAnalysisTitle("  Open   AR by customer ", "ignored"))
	assert.Equal(t, "Margin by lane for Q3",
		agentextension.NormalizeAnalysisTitle("", " Margin by lane  for Q3 "))
	assert.Len(t, []rune(agentextension.NormalizeAnalysisTitle(strings.Repeat("a", 200), "")),
		agentextension.MaxAnalysisTitleRunes)
}
