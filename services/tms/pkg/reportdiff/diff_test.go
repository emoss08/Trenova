package reportdiff_test

import (
	"testing"

	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/pkg/reportdiff"
	"github.com/emoss08/trenova/pkg/reportrows"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func schema() []reportrows.Column {
	return []reportrows.Column{
		{ID: "customer", Label: "Customer", Type: reportcatalog.FieldString},
		{ID: "status", Label: "Status", Type: reportcatalog.FieldEnum},
		{ID: "revenue", Label: "Revenue", Type: reportcatalog.FieldDecimal},
		{ID: "shipments", Label: "Shipments", Type: reportcatalog.FieldInt},
	}
}

func envelope(rows ...[]any) *reportrows.Envelope {
	return &reportrows.Envelope{Schema: schema(), Rows: rows}
}

func changeFor(t *testing.T, result *reportdiff.Result, customer string) reportdiff.Change {
	t.Helper()

	for _, change := range result.Changes {
		if len(change.KeyValues) > 0 && change.KeyValues[0] == customer {
			return change
		}
	}
	t.Fatalf("no change for %q in %#v", customer, result.Changes)

	return reportdiff.Change{}
}

func TestCompareClassifiesEveryKindOfChange(t *testing.T) {
	before := envelope(
		[]any{"ACME", "Billed", "1000.00", float64(10)},
		[]any{"Globex", "Billed", "500.00", float64(5)},
		[]any{"Initech", "Billed", "250.00", float64(2)},
	)
	after := envelope(
		[]any{"ACME", "Billed", "1250.50", float64(12)},
		[]any{"Initech", "Billed", "250.00", float64(2)},
		[]any{"Umbrella", "Billed", "80.00", float64(1)},
	)

	result, err := reportdiff.Compare(before, after, reportdiff.Options{})
	require.NoError(t, err)

	assert.Equal(t, 1, result.Summary.Added)
	assert.Equal(t, 1, result.Summary.Removed)
	assert.Equal(t, 1, result.Summary.Changed)
	assert.Equal(t, 1, result.Summary.Unchanged)
	assert.Equal(t, 0, result.Summary.Duplicate)

	// Unchanged rows are counted but not listed: a weekly report is mostly
	// unchanged, and listing them buries what moved.
	assert.Len(t, result.Changes, 3)
}

// Money is the reason this package exists. A total that is out by a cent
// because it went through a float is a total nobody can reconcile.
func TestCompareIsExactOnDecimals(t *testing.T) {
	before := envelope([]any{"ACME", "Billed", "0.10", float64(1)})
	after := envelope([]any{"ACME", "Billed", "0.30", float64(1)})

	result, err := reportdiff.Compare(before, after, reportdiff.Options{})
	require.NoError(t, err)

	change := changeFor(t, result, "ACME")
	require.Len(t, change.Measures, 1)
	assert.Equal(t, "revenue", change.Measures[0].Column)
	assert.Equal(t, "0.2", change.Measures[0].Delta)
}

// A row that moved from line 40 to line 12 has not changed. Matching by
// position instead of key would report every row of a re-sorted report.
func TestCompareMatchesByKeyNotPosition(t *testing.T) {
	before := envelope(
		[]any{"ACME", "Billed", "100.00", float64(1)},
		[]any{"Globex", "Billed", "200.00", float64(2)},
	)
	after := envelope(
		[]any{"Globex", "Billed", "200.00", float64(2)},
		[]any{"ACME", "Billed", "100.00", float64(1)},
	)

	result, err := reportdiff.Compare(before, after, reportdiff.Options{})
	require.NoError(t, err)

	assert.Equal(t, 2, result.Summary.Unchanged)
	assert.Equal(t, 0, result.Summary.Changed)
	assert.Empty(t, result.Changes)
}

// A repeated key means the chosen keys do not identify a row, so every number
// attached to that key is unreliable — which is worse news than a change, and
// is why it sorts first.
func TestCompareReportsDuplicateKeys(t *testing.T) {
	before := envelope([]any{"ACME", "Billed", "100.00", float64(1)})
	after := envelope(
		[]any{"ACME", "Billed", "60.00", float64(1)},
		[]any{"ACME", "Billed", "40.00", float64(1)},
	)

	result, err := reportdiff.Compare(before, after, reportdiff.Options{})
	require.NoError(t, err)

	assert.Equal(t, 1, result.Summary.Duplicate)
	require.Len(t, result.Changes, 1)
	assert.Equal(t, reportdiff.ChangeKindDuplicate, result.Changes[0].Kind)
	// The key is reported once and only as a duplicate. Also calling it
	// changed would be arithmetic on a figure that was never one row's.
	assert.Equal(t, 0, result.Summary.Changed)
	assert.Equal(t, 0, result.Summary.Removed)
}

// A key the earlier run repeated is still a duplicate when the later run drops
// it entirely: its numbers were already the sum of rows that should have been
// one, so reporting it as a clean removal would be a lie.
func TestCompareReportsDuplicateInTheEarlierRun(t *testing.T) {
	before := envelope(
		[]any{"ACME", "Billed", "60.00", float64(1)},
		[]any{"ACME", "Billed", "40.00", float64(1)},
	)
	after := envelope([]any{"Globex", "Billed", "10.00", float64(1)})

	result, err := reportdiff.Compare(before, after, reportdiff.Options{})
	require.NoError(t, err)

	assert.Equal(t, 1, result.Summary.Duplicate)
	assert.Equal(t, 1, result.Summary.Added)
	assert.Equal(t, 0, result.Summary.Removed)
}

// Removed sorts before added, and within changed the biggest movement leads:
// the order is what somebody reads top down.
func TestCompareOrdersByRisk(t *testing.T) {
	before := envelope(
		[]any{"Gone", "Billed", "10.00", float64(1)},
		[]any{"Small", "Billed", "10.00", float64(1)},
		[]any{"Big", "Billed", "10.00", float64(1)},
	)
	after := envelope(
		[]any{"New", "Billed", "10.00", float64(1)},
		[]any{"Small", "Billed", "11.00", float64(1)},
		[]any{"Big", "Billed", "900.00", float64(1)},
	)

	result, err := reportdiff.Compare(before, after, reportdiff.Options{})
	require.NoError(t, err)
	require.Len(t, result.Changes, 4)

	assert.Equal(t, "Gone", result.Changes[0].KeyValues[0])
	assert.Equal(t, "New", result.Changes[1].KeyValues[0])
	assert.Equal(t, "Big", result.Changes[2].KeyValues[0])
	assert.Equal(t, "Small", result.Changes[3].KeyValues[0])
}

func TestCompareTotalsEachMeasureOnBothSides(t *testing.T) {
	before := envelope(
		[]any{"ACME", "Billed", "100.25", float64(2)},
		[]any{"Globex", "Billed", "50.25", float64(3)},
	)
	after := envelope(
		[]any{"ACME", "Billed", "200.50", float64(4)},
		[]any{"Globex", "Billed", "50.25", float64(3)},
	)

	result, err := reportdiff.Compare(before, after, reportdiff.Options{})
	require.NoError(t, err)
	require.Len(t, result.Totals, 2)

	assert.Equal(t, "revenue", result.Totals[0].Column)
	assert.Equal(t, "150.5", result.Totals[0].Before)
	assert.Equal(t, "250.75", result.Totals[0].After)
	assert.Equal(t, "100.25", result.Totals[0].Delta)

	assert.Equal(t, "shipments", result.Totals[1].Column)
	assert.Equal(t, "2", result.Totals[1].Delta)
}

// Comparing different shapes is not a comparison: one added column shifts
// every position after it.
func TestCompareRefusesMismatchedSchemas(t *testing.T) {
	before := envelope([]any{"ACME", "Billed", "1.00", float64(1)})

	after := &reportrows.Envelope{
		Schema: append(schema(), reportrows.Column{
			ID: "margin", Label: "Margin", Type: reportcatalog.FieldDecimal,
		}),
		Rows: [][]any{{"ACME", "Billed", "1.00", float64(1), "0.10"}},
	}

	_, err := reportdiff.Compare(before, after, reportdiff.Options{})
	require.ErrorIs(t, err, reportdiff.ErrSchemaMismatch)
	assert.Contains(t, err.Error(), "4 columns")
}

func TestCompareRefusesRenamedColumn(t *testing.T) {
	after := envelope([]any{"ACME", "Billed", "1.00", float64(1)})
	before := envelope([]any{"ACME", "Billed", "1.00", float64(1)})
	before.Schema[0].ID = "account"

	_, err := reportdiff.Compare(before, after, reportdiff.Options{})
	require.ErrorIs(t, err, reportdiff.ErrSchemaMismatch)
	assert.Contains(t, err.Error(), "account")
}

func TestCompareRefusesRetypedColumn(t *testing.T) {
	after := envelope([]any{"ACME", "Billed", "1.00", float64(1)})
	before := envelope([]any{"ACME", "Billed", "1.00", float64(1)})
	before.Schema[2].Type = reportcatalog.FieldString

	_, err := reportdiff.Compare(before, after, reportdiff.Options{})
	require.ErrorIs(t, err, reportdiff.ErrSchemaMismatch)
}

func TestCompareRefusesAReportWithNothingToKeyOn(t *testing.T) {
	only := &reportrows.Envelope{
		Schema: []reportrows.Column{
			{ID: "revenue", Label: "Revenue", Type: reportcatalog.FieldDecimal},
		},
		Rows: [][]any{{"1.00"}},
	}

	_, err := reportdiff.Compare(only, only, reportdiff.Options{})
	require.ErrorIs(t, err, reportdiff.ErrNoKeys)
}

func TestCompareRefusesAReportWithNothingToMeasure(t *testing.T) {
	only := &reportrows.Envelope{
		Schema: []reportrows.Column{
			{ID: "customer", Label: "Customer", Type: reportcatalog.FieldString},
		},
		Rows: [][]any{{"ACME"}},
	}

	_, err := reportdiff.Compare(only, only, reportdiff.Options{})
	require.ErrorIs(t, err, reportdiff.ErrNoMeasures)
}

func TestCompareRefusesAnUnknownChosenColumn(t *testing.T) {
	one := envelope([]any{"ACME", "Billed", "1.00", float64(1)})

	_, err := reportdiff.Compare(one, one, reportdiff.Options{Keys: []string{"nope"}})
	require.ErrorIs(t, err, reportdiff.ErrUnknownColumn)
	assert.Contains(t, err.Error(), "nope")
}

// Narrowing the keys coarsens the rows on purpose: asking by status alone
// should fold the customers together, not silently keep keying by customer.
func TestCompareHonoursChosenKeysAndMeasures(t *testing.T) {
	before := envelope(
		[]any{"ACME", "Billed", "100.00", float64(1)},
		[]any{"Globex", "Billed", "100.00", float64(1)},
	)
	after := envelope(
		[]any{"ACME", "Billed", "150.00", float64(1)},
		[]any{"Globex", "Billed", "100.00", float64(1)},
	)

	result, err := reportdiff.Compare(before, after, reportdiff.Options{
		Keys:     []string{"status"},
		Measures: []string{"revenue"},
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"status"}, result.Keys)
	assert.Equal(t, []string{"revenue"}, result.Measures)
	// Two rows share the one key on each side, which is a duplicate, not a
	// change — and saying so is the honest answer.
	assert.Equal(t, 1, result.Summary.Duplicate)
}

// A key column whose value contains the separator must not be able to look
// like two columns.
func TestCompareKeysCannotCollideAcrossColumns(t *testing.T) {
	before := envelope(
		[]any{"A", "B", "1.00", float64(1)},
		[]any{"A\x1fB", "", "2.00", float64(1)},
	)

	result, err := reportdiff.Compare(before, before, reportdiff.Options{})
	require.NoError(t, err)
	assert.Equal(t, 0, result.Summary.Duplicate)
	assert.Equal(t, 2, result.Summary.Unchanged)
}

// A truncated side means a row reported removed may only be past the cap. The
// comparison still runs, and says so.
func TestCompareCarriesTruncation(t *testing.T) {
	before := envelope([]any{"ACME", "Billed", "1.00", float64(1)})
	before.Summary.Truncated = true
	after := envelope([]any{"ACME", "Billed", "1.00", float64(1)})

	result, err := reportdiff.Compare(before, after, reportdiff.Options{})
	require.NoError(t, err)
	assert.True(t, result.Truncated)
}

func TestCompareBoundsTheChangeList(t *testing.T) {
	before := envelope()
	after := envelope()
	for i := range 50 {
		after.Rows = append(after.Rows, []any{
			string(rune('a'+i%26)) + string(rune('a'+i/26)), "Billed", "1.00", float64(1),
		})
	}

	result, err := reportdiff.Compare(before, after, reportdiff.Options{MaxChanges: 10})
	require.NoError(t, err)

	assert.Len(t, result.Changes, 10)
	// The counts always cover every row, so a bounded list understates what is
	// shown, never what changed.
	assert.Equal(t, 50, result.Summary.Added)
}

func TestCompareCanKeepUnchangedRows(t *testing.T) {
	one := envelope([]any{"ACME", "Billed", "1.00", float64(1)})

	result, err := reportdiff.Compare(one, one, reportdiff.Options{IncludeUnchanged: true})
	require.NoError(t, err)

	require.Len(t, result.Changes, 1)
	assert.Equal(t, reportdiff.ChangeKindUnchanged, result.Changes[0].Kind)
}

// A null measure and a measure of zero contribute the same amount to a sum,
// and a row whose measure went from null to zero has not changed.
func TestCompareTreatsNullMeasuresAsZero(t *testing.T) {
	before := envelope([]any{"ACME", "Billed", nil, float64(1)})
	after := envelope([]any{"ACME", "Billed", "0", float64(1)})

	result, err := reportdiff.Compare(before, after, reportdiff.Options{})
	require.NoError(t, err)

	assert.Equal(t, 1, result.Summary.Unchanged)
	assert.Equal(t, "0", result.Totals[0].Delta)
}

// A null key cell is its own identity, not a match for the empty string in
// another column position — the key is built per column, so both stay distinct
// rows only if the other columns differ.
func TestCompareKeysNullAsEmpty(t *testing.T) {
	before := envelope([]any{nil, "Billed", "1.00", float64(1)})
	after := envelope([]any{"", "Billed", "1.00", float64(1)})

	result, err := reportdiff.Compare(before, after, reportdiff.Options{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Summary.Unchanged)
}
