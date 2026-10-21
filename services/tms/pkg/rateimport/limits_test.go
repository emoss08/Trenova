package rateimport_test

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/rateimport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func csvRows(rows int) []byte {
	var buf bytes.Buffer
	buf.WriteString("Origin State,Destination State,Rate\n")
	for i := range rows {
		buf.WriteString("TX,OK," + strconv.Itoa(i) + "\n")
	}
	return buf.Bytes()
}

func limitOf(t *testing.T, err error) *rateimport.LimitError {
	t.Helper()
	require.ErrorIs(t, err, rateimport.ErrLimitExceeded)
	var limit *rateimport.LimitError
	require.ErrorAs(t, err, &limit)
	return limit
}

func TestReadCSV_RefusesAFilePastEachLimit(t *testing.T) {
	t.Parallel()

	_, err := rateimport.ReadCSV(csvRows(20), rateimport.Limits{MaxRows: 10})
	assert.Equal(t, rateimport.LimitRows, limitOf(t, err).Kind)

	wide := "A,B," + strings.Repeat("x,", 20) + "\nTX,OK,1\n"
	_, err = rateimport.ReadCSV([]byte(wide), rateimport.Limits{MaxColumns: 8})
	limit := limitOf(t, err)
	assert.Equal(t, rateimport.LimitColumns, limit.Kind)
	assert.Equal(t, 1, limit.Line)

	long := "Origin State,Rate\nTX," + strings.Repeat("9", 200) + "\n"
	_, err = rateimport.ReadCSV([]byte(long), rateimport.Limits{MaxCellBytes: 100})
	limit = limitOf(t, err)
	assert.Equal(t, rateimport.LimitCellBytes, limit.Kind)
	assert.Equal(t, 2, limit.Line)

	_, err = rateimport.ReadCSV(csvRows(200), rateimport.Limits{MaxSheetBytes: 512})
	assert.Equal(t, rateimport.LimitSheet, limitOf(t, err).Kind)
}

func TestReadCSV_ReadsAFileWithinItsLimits(t *testing.T) {
	t.Parallel()

	sheet, err := rateimport.ReadCSV(csvRows(10), rateimport.Limits{MaxRows: 11})
	require.NoError(t, err)
	assert.Len(t, sheet.Rows, 10)
}

func workbook(t *testing.T, rows int, cell string) []byte {
	t.Helper()
	file := excelize.NewFile()
	t.Cleanup(func() { _ = file.Close() })
	writer, err := file.NewStreamWriter("Sheet1")
	require.NoError(t, err)
	require.NoError(t, writer.SetRow("A1", []any{"Origin State", "Destination State", "Rate"}))
	for i := range rows {
		cellName, cellErr := excelize.CoordinatesToCellName(1, i+2)
		require.NoError(t, cellErr)
		require.NoError(t, writer.SetRow(cellName, []any{"TX", cell, i}))
	}
	require.NoError(t, writer.Flush())
	var buf bytes.Buffer
	require.NoError(t, file.Write(&buf))
	return buf.Bytes()
}

func TestReadXLSX_RefusesAWorkbookPastItsRowLimit(t *testing.T) {
	t.Parallel()

	_, err := rateimport.ReadXLSX(workbook(t, 50, "OK"), rateimport.Limits{MaxRows: 20})
	assert.Equal(t, rateimport.LimitRows, limitOf(t, err).Kind)

	sheet, err := rateimport.ReadXLSX(workbook(t, 5, "OK"), rateimport.Limits{})
	require.NoError(t, err)
	assert.Len(t, sheet.Rows, 5)
}

func TestReadXLSX_RefusesAWorkbookThatUnzipsPastItsLimit(t *testing.T) {
	t.Parallel()

	data := workbook(t, 2_000, strings.Repeat("A", 500))
	require.Less(t, len(data), 200_000, "the archive compresses the repeated text")

	_, err := rateimport.ReadXLSX(data, rateimport.Limits{
		MaxUnzippedBytes: 256 << 10,
		MaxPartBytes:     128 << 10,
	})
	require.ErrorContains(t, err, "unzip size exceeds")
}

func TestValidationError_NamesTheLimitAFileWentPast(t *testing.T) {
	t.Parallel()

	_, err := rateimport.ReadCSV(csvRows(20), rateimport.Limits{MaxRows: 10})
	mapped := rateimport.ValidationError("file", err)

	var validation *errortypes.Error
	require.ErrorAs(t, mapped, &validation)
	assert.Equal(t, "file", validation.Field)
	assert.Equal(t, []any{10}, validation.Args)
	assert.Contains(t, validation.Message, "more than {0} rows")

	plain := rateimport.ErrNoRows
	assert.Equal(t, plain, rateimport.ValidationError("file", plain))
}
