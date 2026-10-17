package agentextension

import (
	"strings"

	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	MaxAnalysisTitleRunes        = 80
	MaxAnalysisTables            = 6
	MaxAnalysisTableColumns      = 20
	MaxAnalysisTableRows         = 200
	MaxAnalysisCellRunes         = 300
	MaxAnalysisFigures           = 8
	MaxAnalysisFigureLabelRunes  = 80
	MaxAnalysisFigureNoteRunes   = 200
	MaxAnalysisColumnNameRunes   = 120
	MaxAnalysisDatasetColumns    = 60
	MaxAnalysisDatasetErrorRunes = 300

	// AnalysisConfigPath is where the sandbox finds what Trenova loaded for
	// it: the organization's timezone and each dataset's columns.
	AnalysisConfigPath = "/home/user/.trenova/analysis.json"
)

// AnalysisValueKind is how a value is shown: the person reads money as money
// and a share as a percentage, whatever the code computed it as.
type AnalysisValueKind string

const (
	AnalysisValueText     = AnalysisValueKind("text")
	AnalysisValueNumber   = AnalysisValueKind("number")
	AnalysisValueMoney    = AnalysisValueKind("money")
	AnalysisValueCount    = AnalysisValueKind("count")
	AnalysisValuePercent  = AnalysisValueKind("percent")
	AnalysisValueDate     = AnalysisValueKind("date")
	AnalysisValueDateTime = AnalysisValueKind("datetime")
	AnalysisValueBool     = AnalysisValueKind("bool")
)

func AllAnalysisValueKinds() []AnalysisValueKind {
	return []AnalysisValueKind{
		AnalysisValueText, AnalysisValueNumber, AnalysisValueMoney, AnalysisValueCount,
		AnalysisValuePercent, AnalysisValueDate, AnalysisValueDateTime, AnalysisValueBool,
	}
}

func (k AnalysisValueKind) IsValid() bool {
	switch k {
	case AnalysisValueText, AnalysisValueNumber, AnalysisValueMoney, AnalysisValueCount,
		AnalysisValuePercent, AnalysisValueDate, AnalysisValueDateTime, AnalysisValueBool:
		return true
	default:
		return false
	}
}

// Or is the kind, or the fallback when it is not one.
func (k AnalysisValueKind) Or(fallback AnalysisValueKind) AnalysisValueKind {
	if k.IsValid() {
		return k
	}

	return fallback
}

// AnalysisTableColumn is one column of a table the code showed.
type AnalysisTableColumn struct {
	Name     string            `json:"name"`
	Kind     AnalysisValueKind `json:"kind"`
	Currency string            `json:"currency,omitempty"`
}

// AnalysisTable is a table the code showed with show_table, kept as values
// so it is drawn as a table rather than as printed text. Percentages are
// fractions; dates are ISO 8601 text.
type AnalysisTable struct {
	Title     string                `json:"title,omitempty"`
	Columns   []AnalysisTableColumn `json:"columns"`
	Rows      [][]any               `json:"rows"`
	TotalRows int                   `json:"totalRows"`
}

// AnalysisFigure is one headline number the code showed with show_figure.
type AnalysisFigure struct {
	Label    string            `json:"label"`
	Value    any               `json:"value"`
	Kind     AnalysisValueKind `json:"kind"`
	Currency string            `json:"currency,omitempty"`
	Note     string            `json:"note,omitempty"`
}

// AnalysisBlockKind is what one thing the code showed was.
type AnalysisBlockKind string

const (
	AnalysisBlockChart = AnalysisBlockKind("chart")
	AnalysisBlockTable = AnalysisBlockKind("table")
)

// AnalysisBlock places one chart or table in the order the code showed it,
// by its position among the charts or its index among the tables.
type AnalysisBlock struct {
	Kind  AnalysisBlockKind `json:"kind"`
	Index int               `json:"index"`
}

// AnalysisDatasetColumn is one column of a dataset as the sandbox loaded it.
type AnalysisDatasetColumn struct {
	Name  string `json:"name"`
	DType string `json:"dtype"`
}

// AnalysisDatasetLoad is how the sandbox loaded one dataset: the variable
// that holds it, its rows and columns, or why it could not be loaded.
type AnalysisDatasetLoad struct {
	Name    string                  `json:"name"`
	Rows    int                     `json:"rows"`
	Columns []AnalysisDatasetColumn `json:"columns"`
	Error   string                  `json:"error,omitempty"`
}

// AnalysisColumnSpec is a report column as the sandbox types and labels it.
type AnalysisColumnSpec struct {
	ID       string            `json:"id"`
	Label    string            `json:"label"`
	Type     string            `json:"type"`
	Kind     AnalysisValueKind `json:"kind"`
	Currency string            `json:"currency,omitempty"`
}

// NormalizeAnalysisTitle is the short name an analysis is shown under.
func NormalizeAnalysisTitle(raw, purpose string) string {
	title := strings.Join(strings.Fields(raw), " ")
	if title == "" {
		title = strings.Join(strings.Fields(purpose), " ")
	}

	return stringutils.TruncateRunes(title, MaxAnalysisTitleRunes)
}

// BoundedTables keeps what a run stores of the tables it showed.
func BoundedTables(raw []AnalysisTable) []AnalysisTable {
	tables := make([]AnalysisTable, 0, min(len(raw), MaxAnalysisTables))
	for idx := range raw {
		if len(tables) == MaxAnalysisTables {
			break
		}
		table := boundedTable(&raw[idx])
		if len(table.Columns) == 0 {
			continue
		}
		tables = append(tables, table)
	}

	return tables
}

func boundedTable(raw *AnalysisTable) AnalysisTable {
	width := min(len(raw.Columns), MaxAnalysisTableColumns)
	columns := make([]AnalysisTableColumn, 0, width)
	for _, column := range raw.Columns[:width] {
		columns = append(columns, AnalysisTableColumn{
			Name: stringutils.TruncateRunes(strings.TrimSpace(column.Name),
				MaxAnalysisColumnNameRunes),
			Kind:     column.Kind.Or(AnalysisValueText),
			Currency: boundedCurrency(column.Currency),
		})
	}

	height := min(len(raw.Rows), MaxAnalysisTableRows)
	rows := make([][]any, 0, height)
	for _, row := range raw.Rows[:height] {
		cells := make([]any, width)
		for col := 0; col < width && col < len(row); col++ {
			cells[col] = boundedCell(row[col])
		}
		rows = append(rows, cells)
	}

	return AnalysisTable{
		Title:     stringutils.TruncateRunes(strings.TrimSpace(raw.Title), MaxAnalysisChartTitle),
		Columns:   columns,
		Rows:      rows,
		TotalRows: max(raw.TotalRows, len(raw.Rows)),
	}
}

// boundedCell keeps a value as one a table can show: a number, text, a flag
// or nothing. Anything else is shown as its text.
func boundedCell(value any) any {
	switch typed := value.(type) {
	case nil, bool, float64, int, int64:
		return typed
	case string:
		return stringutils.TruncateRunes(typed, MaxAnalysisCellRunes)
	default:
		return nil
	}
}

func boundedCurrency(raw string) string {
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

// BoundedFigures keeps what a run stores of the headline numbers it showed.
func BoundedFigures(raw []AnalysisFigure) []AnalysisFigure {
	figures := make([]AnalysisFigure, 0, min(len(raw), MaxAnalysisFigures))
	for idx := range raw {
		if len(figures) == MaxAnalysisFigures {
			break
		}
		label := stringutils.TruncateRunes(strings.TrimSpace(raw[idx].Label),
			MaxAnalysisFigureLabelRunes)
		if label == "" {
			continue
		}
		figures = append(figures, AnalysisFigure{
			Label:    label,
			Value:    boundedCell(raw[idx].Value),
			Kind:     raw[idx].Kind.Or(AnalysisValueNumber),
			Currency: boundedCurrency(raw[idx].Currency),
			Note: stringutils.TruncateRunes(strings.TrimSpace(raw[idx].Note),
				MaxAnalysisFigureNoteRunes),
		})
	}

	return figures
}

// BoundedLayout keeps the order of the charts and tables a run kept, and
// nothing that points past them.
func BoundedLayout(raw []AnalysisBlock, charts, tables int) []AnalysisBlock {
	layout := make([]AnalysisBlock, 0, min(len(raw), charts+tables))
	for _, block := range raw {
		switch {
		case block.Kind == AnalysisBlockChart && block.Index >= 1 && block.Index <= charts,
			block.Kind == AnalysisBlockTable && block.Index >= 0 && block.Index < tables:
			layout = append(layout, block)
		}
	}

	return layout
}

// BoundedLoads keeps what the answer tells the agent about the datasets.
func BoundedLoads(raw []AnalysisDatasetLoad) []AnalysisDatasetLoad {
	loads := make([]AnalysisDatasetLoad, 0, min(len(raw), MaxAnalysisDatasets))
	for idx := range raw {
		if len(loads) == MaxAnalysisDatasets {
			break
		}
		width := min(len(raw[idx].Columns), MaxAnalysisDatasetColumns)
		columns := make([]AnalysisDatasetColumn, 0, width)
		for _, column := range raw[idx].Columns[:width] {
			columns = append(columns, AnalysisDatasetColumn{
				Name:  stringutils.TruncateRunes(column.Name, MaxAnalysisColumnNameRunes),
				DType: stringutils.TruncateRunes(column.DType, 40),
			})
		}
		loads = append(loads, AnalysisDatasetLoad{
			Name:    raw[idx].Name,
			Rows:    max(raw[idx].Rows, 0),
			Columns: columns,
			Error: stringutils.TruncateRunes(strings.TrimSpace(raw[idx].Error),
				MaxAnalysisDatasetErrorRunes),
		})
	}

	return loads
}

// AnalysisKindOf is how a report column's values are shown, from the
// column's type and the format the report gives it.
func AnalysisKindOf(fieldType, format, dateStyle string) AnalysisValueKind {
	switch format {
	case "money":
		return AnalysisValueMoney
	case "percent":
		return AnalysisValuePercent
	case "count":
		return AnalysisValueCount
	}

	switch fieldType {
	case "int", "decimal":
		return AnalysisValueNumber
	case "epoch":
		if dateStyle == "date" {
			return AnalysisValueDate
		}
		return AnalysisValueDateTime
	case "bool":
		return AnalysisValueBool
	default:
		return AnalysisValueText
	}
}
