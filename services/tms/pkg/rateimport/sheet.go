package rateimport

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/xuri/excelize/v2"
)

// Sheet is an uploaded file, read.
type Sheet struct {
	Headers []string
	Rows    [][]string

	// FirstDataRow is the line in the original file the first data row came
	// from, counting from one. It is what somebody uses to find a row in their
	// own spreadsheet, so it counts from the top of the file rather than from
	// wherever the header turned out to be.
	FirstDataRow int
}

var (
	// ErrNoRows means the file has a header and nothing under it.
	ErrNoRows = errors.New("this file has no rate rows in it")
	// ErrNoTable means nothing in the workbook looks like a table.
	ErrNoTable = errors.New("nothing in this file looks like a table of rates")
)

// headerScanLimit is how far into a workbook to look for the header row.
//
// A published tariff has a title, a blank line, sometimes a note about who to
// call. Past a handful of rows, a file without a recognisable header does not
// have one.
const headerScanLimit = 10

// ReadCSV reads a comma separated rate sheet, one record at a time, and
// refuses a file past any of limits.
func ReadCSV(data []byte, limits Limits) (*Sheet, error) {
	limits = limits.withDefaults()
	// Excel writes a byte order mark, and a header read as "\ufeffOrigin State"
	// matches nothing.
	data = bytes.TrimPrefix(data, []byte("\ufeff"))

	reader := csv.NewReader(bytes.NewReader(data))
	// A rate sheet is ragged because somebody left the last columns of some
	// lines blank. Refusing the file over that would be absurd.
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	collector := newRecordCollector(limits)
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("this file could not be read as a CSV: %w", err)
		}
		if err = collector.add(record); err != nil {
			return nil, err
		}
	}

	return sheetFrom(collector.records)
}

// ReadXLSX reads the first sheet of a workbook row by row. The archive is
// opened with the same unzip limits the office document reader uses, so a
// small upload cannot expand into gigabytes of XML.
func ReadXLSX(data []byte, limits Limits) (sheet *Sheet, err error) {
	limits = limits.withDefaults()
	file, err := excelize.OpenReader(bytes.NewReader(data), excelize.Options{
		UnzipSizeLimit:    limits.MaxUnzippedBytes,
		UnzipXMLSizeLimit: limits.MaxPartBytes,
	})
	if err != nil {
		return nil, fmt.Errorf("this file could not be read as a spreadsheet: %w", err)
	}
	defer func() { err = errors.Join(err, file.Close()) }()

	names := file.GetSheetList()
	if len(names) == 0 {
		return nil, ErrNoTable
	}

	rows, err := file.Rows(names[0])
	if err != nil {
		return nil, fmt.Errorf("the first sheet could not be read: %w", err)
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	collector := newRecordCollector(limits)
	for rows.Next() {
		record, columnsErr := rows.Columns()
		if columnsErr != nil {
			return nil, fmt.Errorf("the first sheet could not be read: %w", columnsErr)
		}
		if err = collector.add(record); err != nil {
			return nil, err
		}
	}
	if err = rows.Error(); err != nil {
		return nil, fmt.Errorf("the first sheet could not be read: %w", err)
	}

	return sheetFrom(collector.records)
}

// sheetFrom finds the header row and everything under it.
//
// The header is the first row carrying more than one non-empty cell. A
// published tariff usually opens with a title and a blank line, and a reader
// that took row one as the header would map nothing.
func sheetFrom(records [][]string) (*Sheet, error) {
	if len(records) == 0 {
		return nil, ErrNoTable
	}

	headerIndex := -1

	for index, record := range records {
		if index >= headerScanLimit {
			break
		}

		if populated(record) > 1 {
			headerIndex = index
			break
		}
	}

	if headerIndex < 0 {
		return nil, ErrNoTable
	}

	rows := make([][]string, 0, len(records)-headerIndex-1)
	for _, record := range records[headerIndex+1:] {
		if populated(record) == 0 {
			continue
		}

		rows = append(rows, record)
	}

	if len(rows) == 0 {
		return nil, ErrNoRows
	}

	return &Sheet{
		Headers: records[headerIndex],
		Rows:    rows,
		// Counting from one, the row after the header.
		FirstDataRow: headerIndex + 2,
	}, nil
}

func populated(record []string) int {
	count := 0

	for _, cell := range record {
		if strings.TrimSpace(cell) != "" {
			count++
		}
	}

	return count
}
