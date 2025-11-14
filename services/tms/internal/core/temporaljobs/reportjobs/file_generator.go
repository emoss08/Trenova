package reportjobs

import (
	"bytes"
	"encoding/csv"
	"fmt"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/xuri/excelize/v2"
)

// FileGenerator handles generating export files in different formats
type FileGenerator struct{}

// NewFileGenerator creates a new FileGenerator instance
func NewFileGenerator() *FileGenerator {
	return &FileGenerator{}
}

// GenerateCSV creates a CSV file from query results
func (fg *FileGenerator) GenerateCSV(result *temporaltype.QueryExecutionResult) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	if err := writer.Write(result.Columns); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}

	for _, row := range result.Rows {
		record := make([]string, len(result.Columns))
		for i, col := range result.Columns {
			if val, ok := row[col]; ok && val != nil {
				record[i] = fmt.Sprintf("%v", val)
			}
		}
		if err := writer.Write(record); err != nil {
			return nil, fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("CSV writer error: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateExcel creates an Excel file from query results
func (fg *FileGenerator) GenerateExcel(
	result *temporaltype.QueryExecutionResult,
	resourceType string,
) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := resourceType
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to create sheet: %w", err)
	}

	f.SetActiveSheet(index)

	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#E0E0E0"},
			Pattern: 1,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create header style: %w", err)
	}

	// Write headers
	for i, col := range result.Columns {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err = f.SetCellValue(sheetName, cell, col); err != nil {
			return nil, fmt.Errorf("failed to set header cell: %w", err)
		}
		if err = f.SetCellStyle(sheetName, cell, cell, headerStyle); err != nil {
			return nil, fmt.Errorf("failed to set header style: %w", err)
		}
	}

	// Write data rows
	for rowIdx, row := range result.Rows {
		for colIdx, col := range result.Columns {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
			if val, ok := row[col]; ok && val != nil {
				if err = f.SetCellValue(sheetName, cell, val); err != nil {
					return nil, fmt.Errorf("failed to set cell value: %w", err)
				}
			}
		}
	}

	// Set column widths
	for i := range result.Columns {
		col, _ := excelize.ColumnNumberToName(i + 1)
		if err = f.SetColWidth(sheetName, col, col, 15); err != nil {
			return nil, fmt.Errorf("failed to set column width: %w", err)
		}
	}

	// Delete default sheet
	if err = f.DeleteSheet("Sheet1"); err != nil {
		return nil, fmt.Errorf("failed to delete default sheet: %w", err)
	}

	var buf bytes.Buffer
	if err = f.Write(&buf); err != nil {
		return nil, fmt.Errorf("failed to write Excel file: %w", err)
	}

	return buf.Bytes(), nil
}
