//go:build integration

package db

import (
	"slices"
	"testing"

	"github.com/emoss08/keel"
	"github.com/jackc/pgx/v5"

	"github.com/emoss08/trenova/internal/testutil/seededtest"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/accounting/app/journalrepairservice"
	"github.com/emoss08/trenova/internal/testutil/fiscaltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepairJournalsWritesTheMissingDriverPaymentJournal(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	defer cleanup()

	conn := postgres.WrapDB(db)

	var settlement struct {
		ID             pulid.ID
		OrganizationID pulid.ID
		BusinessUnitID pulid.ID
		PaidAt         int64
	}
	require.NoError(t, keel.QueryRows(ctx, db, func(row pgx.CollectableRow) error {
		return row.Scan(
			&settlement.ID,
			&settlement.OrganizationID,
			&settlement.BusinessUnitID,
			&settlement.PaidAt,
		)
	}, `SELECT id, organization_id, business_unit_id, paid_at FROM driver_settlements
		WHERE status = 'Paid' AND paid_journal_batch_id IS NULL LIMIT 1`))
	require.True(t, settlement.ID.IsNotNil())

	_, err := keel.Exec(ctx, db,
		"UPDATE accounting_controls SET default_cash_account_id = NULL WHERE organization_id = $1",
		settlement.OrganizationID.String())
	require.NoError(t, err)

	preview, err := repairJournals(ctx, conn, settlement.OrganizationID, true)
	require.NoError(t, err)
	assert.Zero(t, preview.PaymentsJournaled)
	assert.True(t, slices.ContainsFunc(preview.Skipped, func(skip journalrepairservice.Skip) bool {
		return skip.Kind == journalrepairservice.KindDriverSettlement && skip.ID == settlement.ID
	}))

	account := func(code string) string {
		id, accountErr := keel.QueryValue[string](ctx, db,
			`SELECT id FROM gl_accounts
			WHERE organization_id = $1 AND business_unit_id = $2 AND account_code = $3`,
			settlement.OrganizationID.String(), settlement.BusinessUnitID.String(), code)
		require.NoError(t, accountErr)
		return id
	}
	_, err = keel.Exec(ctx, db,
		"UPDATE driver_settlements SET posted_payable_account_id = $1 WHERE id = $2",
		account("2010"), settlement.ID.String())
	require.NoError(t, err)
	_, err = keel.Exec(ctx, db,
		"UPDATE accounting_controls SET default_cash_account_id = $1 WHERE organization_id = $2",
		account("1010"), settlement.OrganizationID.String())
	require.NoError(t, err)

	fiscaltest.OpenPeriod(t, ctx, fiscaltest.OpenPeriodParams{
		DB:             db,
		OrganizationID: settlement.OrganizationID,
		BusinessUnitID: settlement.BusinessUnitID,
		At:             settlement.PaidAt,
	})

	report, err := repairJournals(ctx, conn, settlement.OrganizationID, false)
	require.NoError(t, err)
	assert.Equal(t, 1, report.PaymentsJournaled)
	assert.Empty(t, report.Skipped)

	batchID, err := keel.QueryValue[*string](ctx, db,
		"SELECT paid_journal_batch_id FROM driver_settlements WHERE id = $1", settlement.ID.String())
	require.NoError(t, err)
	require.NotNil(t, batchID)

	batchNumber, err := keel.QueryValue[string](ctx, db,
		"SELECT batch_number FROM journal_batches WHERE id = $1", *batchID)
	require.NoError(t, err)
	assert.NotEmpty(t, batchNumber)

	again, err := repairJournals(ctx, conn, settlement.OrganizationID, false)
	require.NoError(t, err)
	assert.Zero(t, again.PaymentsJournaled)
	assert.Zero(t, again.PaymentsLinked)
}
