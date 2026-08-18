package rateimport_test

import (
	"bytes"
	"testing"

	"github.com/emoss08/trenova/pkg/rateimport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func csvBytes(lines ...string) []byte {
	var buf bytes.Buffer
	for _, line := range lines {
		buf.WriteString(line)
		buf.WriteByte('\n')
	}

	return buf.Bytes()
}

func TestReadCSV_ReadsHeadersAndRows(t *testing.T) {
	t.Parallel()

	sheet, err := rateimport.ReadCSV(csvBytes(
		"Origin State,Destination State,Rate",
		"IL,GA,2.35",
		"TX,CA,1.90",
	))

	require.NoError(t, err)
	assert.Equal(t, []string{"Origin State", "Destination State", "Rate"}, sheet.Headers)
	require.Len(t, sheet.Rows, 2)
	assert.Equal(t, []string{"IL", "GA", "2.35"}, sheet.Rows[0])
}

// Spreadsheets exported from Excel carry a byte order mark, and a header read
// as "\ufeffOrigin State" matches nothing.
func TestReadCSV_StripsTheByteOrderMarkExcelWrites(t *testing.T) {
	t.Parallel()

	sheet, err := rateimport.ReadCSV(append([]byte("\ufeff"), csvBytes(
		"Origin State,Rate",
		"IL,2.35",
	)...))

	require.NoError(t, err)
	assert.Equal(t, "Origin State", sheet.Headers[0])
}

// A rate sheet has ragged rows because somebody left the last columns of some
// lines blank, and refusing the file over it would be absurd.
func TestReadCSV_AcceptsRowsShorterThanTheHeader(t *testing.T) {
	t.Parallel()

	sheet, err := rateimport.ReadCSV(csvBytes(
		"Origin State,Destination State,Rate,Minimum",
		"IL,GA,2.35",
	))

	require.NoError(t, err)
	require.Len(t, sheet.Rows, 1)
	assert.Len(t, sheet.Rows[0], 3)
}

func TestReadCSV_AFileWithNoRowsIsRefused(t *testing.T) {
	t.Parallel()

	_, err := rateimport.ReadCSV(csvBytes("Origin State,Destination State,Rate"))

	require.Error(t, err)
}

func TestReadCSV_AnEmptyFileIsRefused(t *testing.T) {
	t.Parallel()

	_, err := rateimport.ReadCSV(nil)

	require.Error(t, err)
}

func xlsxBytes(t *testing.T, rows [][]string) []byte {
	t.Helper()

	file := excelize.NewFile()
	sheetName := file.GetSheetName(0)

	for index, row := range rows {
		cell, err := excelize.CoordinatesToCellName(1, index+1)
		require.NoError(t, err)
		require.NoError(t, file.SetSheetRow(sheetName, cell, &row))
	}

	var buf bytes.Buffer
	require.NoError(t, file.Write(&buf))

	return buf.Bytes()
}

func TestReadXLSX_ReadsTheFirstSheet(t *testing.T) {
	t.Parallel()

	sheet, err := rateimport.ReadXLSX(xlsxBytes(t, [][]string{
		{"Origin State", "Destination State", "Rate"},
		{"IL", "GA", "2.35"},
	}))

	require.NoError(t, err)
	assert.Equal(t, []string{"Origin State", "Destination State", "Rate"}, sheet.Headers)
	require.Len(t, sheet.Rows, 1)
	assert.Equal(t, []string{"IL", "GA", "2.35"}, sheet.Rows[0])
}

// A published tariff usually has a title, a blank line and then the table. A
// reader that took row one as the header would map nothing.
func TestReadXLSX_FindsTheHeaderRowUnderATitle(t *testing.T) {
	t.Parallel()

	sheet, err := rateimport.ReadXLSX(xlsxBytes(t, [][]string{
		{"ACME FREIGHT — 2026 TARIFF"},
		{},
		{"Origin State", "Destination State", "Rate"},
		{"IL", "GA", "2.35"},
	}))

	require.NoError(t, err)
	assert.Equal(t, []string{"Origin State", "Destination State", "Rate"}, sheet.Headers)
	require.Len(t, sheet.Rows, 1)
}

// The row number is what somebody uses to find the line in their own
// spreadsheet, so it has to count from the top of the file rather than from the
// header the reader happened to find.
func TestReadXLSX_RowNumbersCountFromTheTopOfTheFile(t *testing.T) {
	t.Parallel()

	sheet, err := rateimport.ReadXLSX(xlsxBytes(t, [][]string{
		{"ACME FREIGHT — 2026 TARIFF"},
		{},
		{"Origin State", "Destination State", "Rate"},
		{"IL", "GA", "2.35"},
	}))

	require.NoError(t, err)
	assert.Equal(t, 4, sheet.FirstDataRow)
}

func TestReadXLSX_AWorkbookWithNoTableIsRefused(t *testing.T) {
	t.Parallel()

	_, err := rateimport.ReadXLSX(xlsxBytes(t, [][]string{{"ACME FREIGHT"}, {}}))

	require.Error(t, err)
}

func TestReadXLSX_SomethingThatIsNotAWorkbookIsRefused(t *testing.T) {
	t.Parallel()

	_, err := rateimport.ReadXLSX([]byte("this is not a spreadsheet"))

	require.Error(t, err)
}
