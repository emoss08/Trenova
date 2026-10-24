//go:build integration

package migrations_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/schema/migrations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

const ediCatalogConflictTargetUp = "20261231009540_edi_document_type_catalog_conflict_target"

type catalogRow struct {
	ID        string `bun:"id"`
	Code      string `bun:"code"`
	Direction string `bun:"direction"`
	UpdatedAt int64  `bun:"updated_at"`
}

func catalogConflictTargetUp(t *testing.T) string {
	t.Helper()

	body, err := migrations.File(ediCatalogConflictTargetUp)
	require.NoError(t, err)

	return body
}

func catalogRows(ctx context.Context, t *testing.T, db *bun.DB) map[string]catalogRow {
	t.Helper()

	var rows []catalogRow
	require.NoError(t, db.NewRaw(
		`SELECT "id", "code", "direction", "updated_at" FROM "edi_document_types"`,
	).Scan(ctx, &rows))

	byID := make(map[string]catalogRow, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}

	return byID
}

func TestEDIDocumentTypeCatalogConflictTarget_LeavesACompleteCatalogAlone(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	up := catalogConflictTargetUp(t)
	before := catalogRows(ctx, t, db)

	_, err := db.ExecContext(ctx, up)
	require.NoError(t, err)

	assert.Equal(t, before, catalogRows(ctx, t, db))
}

func TestEDIDocumentTypeCatalogConflictTarget_RestoresAMissingCatalogRow(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	up := catalogConflictTargetUp(t)
	_, err := db.ExecContext(ctx,
		`DELETE FROM "edi_document_types" WHERE "id" = 'edidt_x12_214_outbound'`)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, up)
	require.NoError(t, err)

	row, ok := catalogRows(ctx, t, db)["edidt_x12_214_outbound"]
	require.True(t, ok, "the outbound 214 is back in the catalog")
	assert.Equal(t, "X12-214-OUT", row.Code)
	assert.Equal(t, "Outbound", row.Direction)
}

func TestEDIDocumentTypeCatalogConflictTarget_RefusesARowHoldingTheSameCodeUnderAnotherID(
	t *testing.T,
) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	up := catalogConflictTargetUp(t)
	_, err := db.ExecContext(ctx,
		`DELETE FROM "edi_document_types" WHERE "id" = 'edidt_x12_204_inbound'`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO "edi_document_types"
		("id", "code", "name", "standard", "transaction_set", "transaction_set_id", "direction")
		VALUES ('edidt_local_204_in', 'X12-204-IN', 'Local 204', 'X12', '204', 'edits_x12_204', 'Inbound')`)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, up)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "uq_edi_document_types_code")

	_, present := catalogRows(ctx, t, db)["edidt_x12_204_inbound"]
	assert.False(t, present)
}
