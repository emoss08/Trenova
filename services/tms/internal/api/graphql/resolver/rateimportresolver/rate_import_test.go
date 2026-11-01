package rateimportresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateimport"
	pkgrateimport "github.com/emoss08/trenova/pkg/rateimport"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateImportOperations_CheckTheAgreementPermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("rib_").String()
	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"rateImportRows", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).RateImportRows(ctx, id, nil, nil)
			return err
		}},
		{"commitRateImport", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).CommitRateImport(ctx, id)
			return err
		}},
		{"discardRateImport", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).DiscardRateImport(ctx, id)
			return err
		}},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			engine := resolvertest.RefusingEngine()
			deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
			ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

			resolvertest.RequireRefused(
				t,
				call.run(ctx, deps),
				engine,
				permission.ResourceRateAgreement,
				call.operation,
			)
		})
	}
}

func TestRateImportBatchToModel_StatesTheDryRun(t *testing.T) {
	t.Parallel()

	existing := pulid.MustNew("ragr_")
	batch := &rateimport.RateImportBatch{
		ID:              pulid.MustNew("rib_"),
		RateAgreementID: pulid.MustNew("rag_"),
		FileName:        "acme.csv",
		SourceFormat:    rateimport.SourceFormatCSV,
		Status:          rateimport.StatusParsed,
		EffectiveFrom:   1_767_225_600,
		Mapping:         map[string]int{"origin": 0},
		Changes: []pkgrateimport.Change{
			{
				Kind:       pkgrateimport.ChangeKindChanged,
				LaneKey:    "GA>IL",
				ExistingID: existing.String(),
				Fields: []pkgrateimport.FieldChange{
					{Field: "rate", Before: "2.10", After: "2.35"},
				},
			},
			{Kind: pkgrateimport.ChangeKindAdded, LaneKey: "TN>OH"},
		},
		Summary:    &pkgrateimport.Summary{Added: 1, Changed: 1},
		RowCount:   2,
		ErrorCount: 0,
	}

	model := rateImportBatchToModel(batch)

	assert.Equal(t, batch.ID.String(), model.ID)
	assert.Equal(t, rateimport.StatusParsed, model.Status)
	assert.Equal(t, map[string]any{"origin": 0}, model.Mapping)
	assert.Equal(t, []string{}, model.UnmappedHeaders)
	require.Len(t, model.Changes, 2)
	assert.Equal(t, existing.String(), *model.Changes[0].ExistingID)
	assert.Equal(t, "2.35", model.Changes[0].Fields[0].After)
	assert.Nil(t, model.Changes[1].ExistingID)
	assert.Equal(t, 1, model.Summary.Added)
	assert.Nil(t, model.CommittedAt)
}

func TestRateImportRowsToModel_KeepsTheCellsAsRead(t *testing.T) {
	t.Parallel()

	rows := rateImportRowsToModel([]*rateimport.RateImportRow{
		{RowNumber: 3, Cells: []string{"GA", "IL", "abc"}, Error: "rate is not a number"},
		nil,
		{RowNumber: 4},
	})

	require.Len(t, rows, 2)
	assert.Equal(t, 3, rows[0].RowNumber)
	assert.Equal(t, []string{"GA", "IL", "abc"}, rows[0].Cells)
	assert.Equal(t, "rate is not a number", rows[0].Error)
	assert.Equal(t, []string{}, rows[1].Cells)
}

func TestBoundedFirst_CapsAPageAtTheListLimit(t *testing.T) {
	t.Parallel()

	tooMany, none := 5000, 0
	assert.Equal(t, defaultRowPage, boundedFirst(nil))
	assert.Equal(t, 100, boundedFirst(&tooMany))
	assert.Equal(t, 1, boundedFirst(&none))
}
