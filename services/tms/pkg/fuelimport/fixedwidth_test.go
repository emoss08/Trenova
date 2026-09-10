package fuelimport_test

import (
	"testing"

	"github.com/emoss08/trenova/pkg/fuelimport"
	"github.com/stretchr/testify/require"
)

func TestParseFixedWidthLayout(t *testing.T) {
	t.Parallel()

	layout, err := fuelimport.ParseFixedWidthLayout(
		"Transaction Date:1-8, Card Number:9-24,Unit Number:25-34",
	)
	require.NoError(t, err)
	require.Equal(
		t,
		[]string{"Transaction Date", "Card Number", "Unit Number"},
		layout.Headers(),
	)
}

func TestParseFixedWidthLayoutAcceptsNewlinesAndComments(t *testing.T) {
	t.Parallel()

	layout, err := fuelimport.ParseFixedWidthLayout(`
		# Comdata AC00029, per the record layout Corpay publishes
		Transaction Date: 1-8
		Card Number: 9-24
	`)
	require.NoError(t, err)
	require.Equal(t, []string{"Transaction Date", "Card Number"}, layout.Headers())
}

func TestParseFixedWidthLayoutRejectsBadSpecs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		spec    string
		wantErr string
	}{
		{name: "empty", spec: "   ", wantErr: "layout is empty"},
		{name: "no range", spec: "Card Number", wantErr: "must be written as"},
		{name: "not numeric", spec: "Card Number:a-b", wantErr: "must be written as"},
		{name: "reversed", spec: "Card Number:24-9", wantErr: "ends before it starts"},
		{name: "zero start", spec: "Card Number:0-8", wantErr: "starts at column 1"},
		{name: "no name", spec: ":1-8", wantErr: "needs a name"},
		{
			name:    "duplicate name",
			spec:    "Card Number:1-8,Card Number:9-16",
			wantErr: "appears twice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := fuelimport.ParseFixedWidthLayout(tt.spec)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestReadFixedWidthSlicesEachRecord(t *testing.T) {
	t.Parallel()

	layout, err := fuelimport.ParseFixedWidthLayout("Date:1-8,Card:9-12,Gallons:13-18")
	require.NoError(t, err)

	sheet, err := fuelimport.ReadFixedWidth([]byte(
		"202601054411 125.4\n2026010622320098.2\n",
	), layout)
	require.NoError(t, err)

	require.Equal(t, []string{"Date", "Card", "Gallons"}, sheet.Headers)
	require.Equal(t, 1, sheet.FirstDataRow)
	require.Equal(t, [][]string{
		{"20260105", "4411", "125.4"},
		{"20260106", "2232", "0098.2"},
	}, sheet.Rows)
}

// Vendors pad the last field of a record only when they feel like it, and some
// trailers are shorter than the layout. A short record must yield blank cells for
// the fields it does not reach rather than failing the whole file.
func TestReadFixedWidthToleratesShortRecords(t *testing.T) {
	t.Parallel()

	layout, err := fuelimport.ParseFixedWidthLayout("Date:1-8,Card:9-12,Gallons:13-18")
	require.NoError(t, err)

	sheet, err := fuelimport.ReadFixedWidth([]byte("202601054411\n"), layout)
	require.NoError(t, err)
	require.Equal(t, [][]string{{"20260105", "4411", ""}}, sheet.Rows)
}

func TestReadFixedWidthSkipsBlankLinesAndCarriageReturns(t *testing.T) {
	t.Parallel()

	layout, err := fuelimport.ParseFixedWidthLayout("Date:1-8,Card:9-12")
	require.NoError(t, err)

	sheet, err := fuelimport.ReadFixedWidth(
		[]byte("202601054411\r\n\r\n   \n202601064412\r\n"),
		layout,
	)
	require.NoError(t, err)
	require.Equal(t, [][]string{
		{"20260105", "4411"},
		{"20260106", "4412"},
	}, sheet.Rows)
}

func TestReadFixedWidthRejectsAFileWithNoRecords(t *testing.T) {
	t.Parallel()

	layout, err := fuelimport.ParseFixedWidthLayout("Date:1-8")
	require.NoError(t, err)

	_, err = fuelimport.ReadFixedWidth([]byte("\n  \n"), layout)
	require.ErrorIs(t, err, fuelimport.ErrNoFixedWidthRecords)
}
