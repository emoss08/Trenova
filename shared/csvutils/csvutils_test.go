package csvutils

import (
	"bytes"
	"encoding/csv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSafeCell_NeutralisesEveryFormulaPrefix(t *testing.T) {
	t.Parallel()

	for _, value := range []string{
		"=HYPERLINK(\"http://x\")",
		"+1+1",
		"-2+3",
		"@SUM(A1)",
		"\t=1",
		"\r=1",
	} {
		assert.Equal(t, "'"+value, SafeCell(value), "%q", value)
	}
}

func TestSafeCell_LeavesOrdinaryTextAlone(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"", "shipment", "12.50", " =not first", "a=b", "'quoted"} {
		assert.Equal(t, value, SafeCell(value), "%q", value)
	}
}

func TestSafeRow_SurvivesTheCSVWriter(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	require.NoError(t, w.Write(SafeRow([]string{"=cmd|' /C calc'!A0", "plain", "-5, \"x\""})))
	w.Flush()
	require.NoError(t, w.Error())

	records, err := csv.NewReader(&buf).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, []string{"'=cmd|' /C calc'!A0", "plain", "'-5, \"x\""}, records[0])
}

func TestSafeNumberCell_KeepsFormattedNumbers(t *testing.T) {
	t.Parallel()

	for _, value := range []string{
		"", "12", "-12", "+12", "-1,234.50", "-$1,234.50", "-1.234,50 €", "-12.5%",
		"-1.2K", "($99.10)", "-0", "- 5",
	} {
		assert.Equal(t, value, SafeNumberCell(value), "%q", value)
	}
}

func TestSafeNumberCell_NeutralisesAnythingElse(t *testing.T) {
	t.Parallel()

	for _, value := range []string{
		"=1+1", "@SUM(A1)", "\t5", "\r5", "-1+1", "-2+cmd|' /C calc'!A0", "-1h 30m",
		"-", "+", "-$", "-5K5", "-HYPERLINK(\"http://x\")",
	} {
		assert.Equal(t, "'"+value, SafeNumberCell(value), "%q", value)
	}
}
