package reportrows_test

import (
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/pkg/reportrows"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRawValueKeepsDecimalsExactAndInstantsAsUnixSeconds(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, time.October, 26, 9, 14, 0, 0, time.UTC)

	assert.Equal(t, "2840.1", reportrows.RawValue(decimal.RequireFromString("2840.10")))
	assert.Equal(t, at.Unix(), reportrows.RawValue(at))
	assert.Equal(t, "ACME", reportrows.RawValue("ACME"))
	assert.Nil(t, reportrows.RawValue(nil))
	assert.Equal(t, []any{"1.5", at.Unix(), nil}, reportrows.RawRow([]any{
		decimal.RequireFromString("1.5"), at, nil,
	}))
	assert.Nil(t, reportrows.RawRow(nil))
}

func TestFitDropsTrailingRowsUntilTheSampleIsSmallEnough(t *testing.T) {
	t.Parallel()

	wide := strings.Repeat("x", 6*1024)
	rows := make([][]any, 0, 20)
	for range 20 {
		rows = append(rows, []any{wide})
	}
	sample := (&reportrows.Sample{
		Schema: []reportrows.Column{{ID: "note", Label: "Note"}},
		Rows:   rows,
	}).Fit()

	require.NotNil(t, sample)
	encoded, err := sonic.Marshal(sample)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(encoded), reportrows.MaxSampleBytes)
	assert.Len(t, sample.Rows, 7)
	assert.True(t, sample.More)
}

func TestFitKeepsASampleThatAlreadyFits(t *testing.T) {
	t.Parallel()

	sample := (&reportrows.Sample{
		Schema: []reportrows.Column{{ID: "note", Label: "Note"}},
		Rows:   [][]any{{"a"}, {"b"}},
	}).Fit()

	require.NotNil(t, sample)
	assert.Len(t, sample.Rows, 2)
	assert.False(t, sample.More)
}

func TestFitGivesUpOnASchemaTooWideToKeep(t *testing.T) {
	t.Parallel()

	columns := make([]reportrows.Column, 0, 10)
	for range 10 {
		columns = append(columns, reportrows.Column{ID: strings.Repeat("c", 6*1024)})
	}

	assert.Nil(t, (&reportrows.Sample{Schema: columns, Rows: [][]any{{"a"}}}).Fit())
	assert.Nil(t, (*reportrows.Sample)(nil).Fit())
}
