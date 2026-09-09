package fuelpurchase_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validBatch() *fuelpurchase.ImportBatch {
	return &fuelpurchase.ImportBatch{
		Provider:        fuelpurchase.CardProviderEFS,
		Status:          fuelpurchase.ImportStatusPending,
		DefaultCurrency: "USD",
	}
}

func TestImportBatch_PendingBatchPasses(t *testing.T) {
	t.Parallel()

	batch := validBatch()
	batch.Normalize()

	multiErr := errortypes.NewMultiError()
	batch.Validate(multiErr)

	assert.False(t, multiErr.HasErrors(), multiErr.Error())
	assert.True(t, batch.CanStage())
	assert.False(t, batch.CanCommit())
	assert.True(t, batch.CanDiscard())
	assert.False(t, batch.HasRowsToCommit())
}

func TestImportBatch_ValidateRejections(t *testing.T) {
	t.Parallel()

	committedAt := int64(1_800_000_000)
	documentID := pulid.MustNew("doc_")

	tests := []struct {
		name   string
		mutate func(b *fuelpurchase.ImportBatch)
		field  string
	}{
		{
			name:   "parsed without a document",
			mutate: func(b *fuelpurchase.ImportBatch) { b.Status = fuelpurchase.ImportStatusParsed },
			field:  "documentId",
		},
		{
			name: "committed without a stamp",
			mutate: func(b *fuelpurchase.ImportBatch) {
				b.Status = fuelpurchase.ImportStatusCommitted
				b.DocumentID = &documentID
			},
			field: "committedAt",
		},
		{
			name: "committed without a committer",
			mutate: func(b *fuelpurchase.ImportBatch) {
				b.Status = fuelpurchase.ImportStatusCommitted
				b.DocumentID = &documentID
				b.CommittedAt = &committedAt
			},
			field: "committedById",
		},
		{
			name:   "pending with a commit stamp",
			mutate: func(b *fuelpurchase.ImportBatch) { b.CommittedAt = &committedAt },
			field:  "committedAt",
		},
		{
			name:   "unknown provider",
			mutate: func(b *fuelpurchase.ImportBatch) { b.Provider = "Fleetcor" },
			field:  "provider",
		},
		{
			name:   "bad default fuel type",
			mutate: func(b *fuelpurchase.ImportBatch) { b.DefaultFuelType = "Kerosene" },
			field:  "defaultFuelType",
		},
		{
			name:   "bad default currency",
			mutate: func(b *fuelpurchase.ImportBatch) { b.DefaultCurrency = "US" },
			field:  "defaultCurrency",
		},
		{
			name:   "negative counts",
			mutate: func(b *fuelpurchase.ImportBatch) { b.ErrorCount = -1 },
			field:  "rowCount",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			batch := validBatch()
			tt.mutate(batch)

			multiErr := errortypes.NewMultiError()
			batch.Validate(multiErr)

			require.True(t, multiErr.HasErrors(), "expected a validation error")
			assert.Contains(t, fieldErrors(multiErr), tt.field)
		})
	}
}

func TestImportBatch_CommittedBatchPasses(t *testing.T) {
	t.Parallel()

	committedAt := int64(1_800_000_000)
	documentID := pulid.MustNew("doc_")

	batch := validBatch()
	batch.Status = fuelpurchase.ImportStatusCommitted
	batch.DocumentID = &documentID
	batch.CommittedAt = &committedAt
	batch.CommittedByID = pulid.MustNew("usr_")
	batch.Summary = &fuelpurchase.ImportSummary{NewCount: 3}

	multiErr := errortypes.NewMultiError()
	batch.Validate(multiErr)

	assert.False(t, multiErr.HasErrors(), multiErr.Error())
	assert.True(t, batch.IsTerminal())
	assert.Equal(t, 3, batch.NewRowCount())
}

func TestImportRow_WillCommit(t *testing.T) {
	t.Parallel()

	row := &fuelpurchase.ImportRow{
		ImportBatchID: pulid.MustNew("fpib_"),
		RowNumber:     2,
		Status:        fuelpurchase.ImportRowStatusNew,
		Parsed:        validPurchase(),
	}
	assert.True(t, row.WillCommit())
	assert.False(t, row.Failed())

	row.Error = "tractor not found"
	assert.False(t, row.WillCommit())
	assert.True(t, row.Failed())

	row.Error = ""
	row.Parsed = nil
	assert.False(t, row.WillCommit())

	row.Parsed = validPurchase()
	row.Status = fuelpurchase.ImportRowStatusDuplicateInFile
	assert.False(t, row.WillCommit())
}

func TestImportRow_ValidateRejections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(r *fuelpurchase.ImportRow)
		field  string
	}{
		{
			name:   "error row without a message",
			mutate: func(r *fuelpurchase.ImportRow) { r.Status = fuelpurchase.ImportRowStatusError },
			field:  "error",
		},
		{
			name: "committed row without a purchase",
			mutate: func(r *fuelpurchase.ImportRow) {
				r.Status = fuelpurchase.ImportRowStatusCommitted
			},
			field: "fuelPurchaseId",
		},
		{
			name:   "row number zero",
			mutate: func(r *fuelpurchase.ImportRow) { r.RowNumber = 0 },
			field:  "rowNumber",
		},
		{
			name:   "missing batch",
			mutate: func(r *fuelpurchase.ImportRow) { r.ImportBatchID = pulid.Nil },
			field:  "importBatchId",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			row := &fuelpurchase.ImportRow{
				ImportBatchID: pulid.MustNew("fpib_"),
				RowNumber:     2,
				Status:        fuelpurchase.ImportRowStatusNew,
			}
			tt.mutate(row)

			multiErr := errortypes.NewMultiError()
			row.Validate(multiErr)

			require.True(t, multiErr.HasErrors(), "expected a validation error")
			assert.Contains(t, fieldErrors(multiErr), tt.field)
		})
	}
}
