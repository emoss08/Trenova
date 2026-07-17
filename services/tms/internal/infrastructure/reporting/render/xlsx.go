package render

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"
)

var _ services.ReportRenderer = (*XLSXRenderer)(nil)

// excelSheetRowLimit is the XLSX per-sheet row capacity (1,048,576) minus one
// header row per data sheet.
const excelSheetRowLimit = 1_048_575

type XLSXRenderer struct{}

func NewXLSX() *XLSXRenderer { return &XLSXRenderer{} }

func (r *XLSXRenderer) Format() report.Format { return report.FormatXLSX }

func (r *XLSXRenderer) Render(
	ctx context.Context,
	req *services.ReportRenderRequest,
) (*services.ReportRenderStats, error) {
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

	if err = r.writeDataSheets(ctx, file, req, schema, headerStyle); err != nil {
		return nil, err
	}

	if index, idxErr := file.GetSheetIndex("Data"); idxErr == nil && index >= 0 {
		file.SetActiveSheet(index)
	}

	truncated := req.Dataset.Truncated()

	if err = file.Write(req.Sink); err != nil {
		return nil, err
	}

	return &services.ReportRenderStats{
		Rows:      req.Dataset.RowCount(),
		Truncated: truncated,
	}, nil
}

func (r *XLSXRenderer) writeDataSheets(
	ctx context.Context,
	file *excelize.File,
	req *services.ReportRenderRequest,
	schema []services.ReportResultColumn,
	headerStyle int,
) error {
	loc := metaLocation(&req.Meta)

	sheetIndex := 1
	writer, err := r.newDataSheet(file, "Data", schema, headerStyle)
	if err != nil {
		return err
	}
	rowsOnSheet := 0

	cells := make([]any, len(schema))
	for {
		row, nextErr := req.Dataset.Next(ctx)
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			return nextErr
		}

		if rowsOnSheet >= excelSheetRowLimit {
			if err = writer.Flush(); err != nil {
				return err
			}
			sheetIndex++
			writer, err = r.newDataSheet(
				file, fmt.Sprintf("Data (%d)", sheetIndex), schema, headerStyle,
			)
			if err != nil {
				return err
			}
			rowsOnSheet = 0
		}

		for i := range schema {
			cells[i] = xlsxCell(&schema[i], row[i], loc)
		}
		if err = r.writeRow(writer, rowsOnSheet+2, cells); err != nil {
			return err
		}
		rowsOnSheet++
	}

	if req.Dataset.Truncated() {
		if err = r.writeRow(writer, rowsOnSheet+2, []any{truncationNotice}); err != nil {
			return err
		}
	}

	return writer.Flush()
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
	req *services.ReportRenderRequest,
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
	for name, value := range req.Meta.Params {
		rows = append(rows, []any{"Parameter: " + name, fmt.Sprint(value)})
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
	schema []services.ReportResultColumn,
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

func xlsxCell(column *services.ReportResultColumn, value any, loc *time.Location) any {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case decimal.Decimal:
		f, _ := v.Float64()
		return f
	case int64:
		if column.Type == reportcatalog.FieldEpoch {
			return time.Unix(v, 0).In(loc).Format("2006-01-02 15:04:05")
		}
		return v
	default:
		return v
	}
}
