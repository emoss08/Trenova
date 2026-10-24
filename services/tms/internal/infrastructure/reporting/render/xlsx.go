package render

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/reportfmt"
	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"
)

var _ reportports.ReportRenderer = (*XLSXRenderer)(nil)

// excelSheetRowLimit is the XLSX per-sheet row capacity (1,048,576) minus one
// header row per data sheet.
const excelSheetRowLimit = 1_048_575

type XLSXRenderer struct{}

func NewXLSX() *XLSXRenderer { return &XLSXRenderer{} }

func (r *XLSXRenderer) Format() report.Format { return report.FormatXLSX }

func (r *XLSXRenderer) Render(
	ctx context.Context,
	req *reportports.ReportRenderRequest,
) (*reportports.ReportRenderStats, error) {
	file := excelize.NewFile()
	defer file.Close()

	schema := req.Dataset.Schema()

	if err := r.writeInfoSheet(file, req); err != nil {
		return nil, err
	}

	headerStyle, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return nil, err
	}

	columnStyles, err := newColumnStyles(file, schema, false)
	if err != nil {
		return nil, err
	}
	totalsStyles, err := newColumnStyles(file, schema, true)
	if err != nil {
		return nil, err
	}
	toneStyles, err := newToneStyles(file, schema)
	if err != nil {
		return nil, err
	}

	if err = r.writeDataSheets(ctx, file, req, &sheetStyles{
		header:  headerStyle,
		columns: columnStyles,
		totals:  totalsStyles,
		tones:   toneStyles,
	}); err != nil {
		return nil, err
	}

	if index, idxErr := file.GetSheetIndex("Data"); idxErr == nil && index >= 0 {
		file.SetActiveSheet(index)
	}

	truncated := req.Dataset.Truncated()

	if err = file.Write(req.Sink); err != nil {
		return nil, err
	}

	return &reportports.ReportRenderStats{
		Rows:      req.Dataset.RowCount(),
		Truncated: truncated,
	}, nil
}

type sheetStyles struct {
	header  int
	columns []int
	totals  []int
	// tones indexes a column's conditional-formatting styles by tone, so a
	// banded cell keeps its number format and gains the emphasis colour.
	tones []map[reportfmt.Tone]int
}

// toneColours mirror the on-screen palette closely enough to be recognisable
// while staying legible against Excel's default white sheet.
//
//nolint:exhaustive // ToneNone means "unbanded", which has no colour
var toneColours = map[reportfmt.Tone]string{
	reportfmt.TonePositive: "#15803D",
	reportfmt.ToneWarning:  "#B45309",
	reportfmt.ToneNegative: "#B91C1C",
	reportfmt.ToneNeutral:  "#3F3F46",
}

// newToneStyles registers one style per (column, tone) pair that a rule can
// actually produce, so no styles are created for columns without rules.
func newToneStyles(
	file *excelize.File,
	schema []reportports.ReportResultColumn,
) ([]map[reportfmt.Tone]int, error) {
	styles := make([]map[reportfmt.Tone]int, len(schema))

	for i := range schema {
		display := &schema[i].Display
		if len(display.Rules) == 0 {
			continue
		}
		code, native := display.ExcelFormat()
		byTone := make(map[reportfmt.Tone]int, len(display.Rules))

		for _, rule := range display.Rules {
			if _, done := byTone[rule.Tone]; done {
				continue
			}
			colour, known := toneColours[rule.Tone]
			if !known {
				continue
			}
			style := &excelize.Style{Font: &excelize.Font{Color: colour, Bold: true}}
			if native && code != "" {
				style.CustomNumFmt = &code
			}
			styleID, err := file.NewStyle(style)
			if err != nil {
				return nil, fmt.Errorf("register tone style %q: %w", rule.Tone, err)
			}
			byTone[rule.Tone] = styleID
		}
		styles[i] = byTone
	}

	return styles, nil
}

func (r *XLSXRenderer) writeDataSheets(
	ctx context.Context,
	file *excelize.File,
	req *reportports.ReportRenderRequest,
	styles *sheetStyles,
) error {
	loc := metaLocation(&req.Meta)
	schema := req.Dataset.Schema()

	sheetIndex := 1
	writer, err := r.newDataSheet(file, "Data", schema, styles.header)
	if err != nil {
		return err
	}
	rowsOnSheet := 0

	cells := make([]any, len(schema))
	if err = streamFormatted(ctx, req.Dataset, loc, func(row *reportfmt.Row) error {
		if rowsOnSheet >= excelSheetRowLimit {
			if flushErr := writer.Flush(); flushErr != nil {
				return flushErr
			}
			sheetIndex++
			var sheetErr error
			writer, sheetErr = r.newDataSheet(
				file, fmt.Sprintf("Data (%d)", sheetIndex), schema, styles.header,
			)
			if sheetErr != nil {
				return sheetErr
			}
			rowsOnSheet = 0
		}

		for i := range schema {
			cells[i] = xlsxCell(
				&schema[i],
				cellAt(row.Values, i),
				row.Cells[i],
				loc,
				styles.styleFor(i, row.Cells[i].Tone),
			)
		}
		if rowErr := r.writeRow(writer, rowsOnSheet+2, cells); rowErr != nil {
			return rowErr
		}
		rowsOnSheet++
		return nil
	}); err != nil {
		return err
	}

	written, err := r.writeTotalsRow(ctx, writer, req, styles, loc, rowsOnSheet+2)
	if err != nil {
		return err
	}
	rowsOnSheet += written

	if req.Dataset.Truncated() {
		if err = r.writeRow(writer, rowsOnSheet+2, []any{truncationNotice}); err != nil {
			return err
		}
	}

	return writer.Flush()
}

// writeTotalsRow keeps the grand total a native, bold, still-formatted row so
// the spreadsheet can be re-sorted or charted without losing it to a string.
func (r *XLSXRenderer) writeTotalsRow(
	ctx context.Context,
	writer *excelize.StreamWriter,
	req *reportports.ReportRenderRequest,
	styles *sheetStyles,
	loc *time.Location,
	rowNum int,
) (int, error) {
	row, err := req.Dataset.Totals(ctx)
	if err != nil || row == nil {
		return 0, err
	}

	schema := req.Dataset.Schema()
	formatted, err := reportfmt.FormatRow(ctx, formatColumns(schema), row, loc)
	if err != nil {
		return 0, err
	}

	cells := make([]any, len(schema))
	for i := range schema {
		if i >= len(row) || row[i] == nil {
			continue
		}
		cells[i] = xlsxCell(&schema[i], row[i], formatted[i], loc, styles.totals[i])
	}
	if len(cells) > 0 && row[0] == nil {
		cells[0] = excelize.Cell{StyleID: styles.header, Value: totalsLabel}
	}

	if err = r.writeRow(writer, rowNum, cells); err != nil {
		return 0, err
	}
	return 1, nil
}

// styleFor picks the banded style when a rule matched this cell, falling back
// to the column's own number format.
func (s *sheetStyles) styleFor(index int, tone reportfmt.Tone) int {
	if tone != reportfmt.ToneNone && index < len(s.tones) && s.tones[index] != nil {
		if styleID, ok := s.tones[index][tone]; ok {
			return styleID
		}
	}
	return s.columns[index]
}

func cellAt(row []any, index int) any {
	if index < len(row) {
		return row[index]
	}
	return nil
}

func (r *XLSXRenderer) writeRow(writer *excelize.StreamWriter, rowNum int, cells []any) error {
	cell, err := excelize.CoordinatesToCellName(1, rowNum)
	if err != nil {
		return err
	}
	return writer.SetRow(cell, cells)
}

func (r *XLSXRenderer) writeInfoSheet(
	file *excelize.File,
	req *reportports.ReportRenderRequest,
) error {
	const sheet = "Report"
	if err := file.SetSheetName("Sheet1", sheet); err != nil {
		return err
	}

	loc := metaLocation(&req.Meta)
	generatedAt := time.Unix(req.Meta.GeneratedAtUnix, 0).In(loc).Format("2006-01-02 15:04:05 MST")

	rows := [][]any{
		{"Report", req.Meta.Title},
		{"Description", req.Meta.Description},
		{"Generated At", generatedAt},
		{"Requested By", req.Meta.RequestedBy},
	}
	for _, name := range orderedParamNames(&req.Meta) {
		rows = append(rows, []any{"Parameter: " + name, fmt.Sprint(req.Meta.Params[name])})
	}

	for i, row := range rows {
		cell, err := excelize.CoordinatesToCellName(1, i+1)
		if err != nil {
			return err
		}
		if err = file.SetSheetRow(sheet, cell, &row); err != nil {
			return err
		}
	}

	return nil
}

func (r *XLSXRenderer) newDataSheet(
	file *excelize.File,
	sheetName string,
	schema []reportports.ReportResultColumn,
	headerStyle int,
) (*excelize.StreamWriter, error) {
	if _, err := file.NewSheet(sheetName); err != nil {
		return nil, err
	}

	writer, err := file.NewStreamWriter(sheetName)
	if err != nil {
		return nil, err
	}

	header := make([]any, len(schema))
	for i := range schema {
		header[i] = excelize.Cell{StyleID: headerStyle, Value: schema[i].Label}
	}
	if err = writer.SetRow("A1", header); err != nil {
		return nil, err
	}

	return writer, nil
}

// newColumnStyles registers one cell style per column so numbers, money, and
// dates land in the sheet as native values carrying the column's display
// format, instead of pre-rendered strings a spreadsheet cannot aggregate.
func newColumnStyles(
	file *excelize.File,
	schema []reportports.ReportResultColumn,
	bold bool,
) ([]int, error) {
	styles := make([]int, len(schema))
	byCode := make(map[string]int, len(schema))

	var font *excelize.Font
	if bold {
		font = &excelize.Font{Bold: true}
	}

	for i := range schema {
		code, native := schema[i].Display.ExcelFormat()
		if !native || code == "" {
			continue
		}
		if styleID, ok := byCode[code]; ok {
			styles[i] = styleID
			continue
		}
		styleID, err := file.NewStyle(&excelize.Style{CustomNumFmt: &code, Font: font})
		if err != nil {
			return nil, fmt.Errorf("register number format %q: %w", code, err)
		}
		byCode[code] = styleID
		styles[i] = styleID
	}

	return styles, nil
}

func xlsxCell(
	column *reportports.ReportResultColumn,
	value any,
	formatted reportfmt.Cell,
	loc *time.Location,
	styleID int,
) any {
	if value == nil {
		return nil
	}

	_, native := column.Display.ExcelFormat()
	if !native {
		return formatted.Text
	}

	return excelize.Cell{StyleID: styleID, Value: xlsxNativeValue(column, value, loc)}
}

func xlsxNativeValue(column *reportports.ReportResultColumn, value any, loc *time.Location) any {
	asDate := column.Display.Style == reportfmt.StyleDate

	switch v := value.(type) {
	case decimal.Decimal:
		if asDate {
			return time.Unix(v.IntPart(), 0).In(loc)
		}
		f, _ := v.Float64()
		return f
	case int64:
		if asDate {
			return time.Unix(v, 0).In(loc)
		}
		return v
	default:
		return v
	}
}
