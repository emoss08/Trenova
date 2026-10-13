//go:build integration

package db

import (
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/fiscaltest"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/accounting/app/journalrepairservice"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepairJournalsWritesTheMissingDriverPaymentJournal(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	seedRegistry := seeder.NewRegistry()
	seeds.Register(seedRegistry)
	engine := seeder.NewEngine(
		db,
		seedRegistry,
		&config.Config{System: config.SystemConfig{SystemUserPassword: "test-system-password"}},
	)
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)
	conn := postgres.WrapDB(db)

	var settlement struct {
		ID             pulid.ID `bun:"id"`
		OrganizationID pulid.ID `bun:"organization_id"`
		BusinessUnitID pulid.ID `bun:"business_unit_id"`
		PaidAt         int64    `bun:"paid_at"`
	}
	require.NoError(t, db.NewSelect().
		Table("driver_settlements").
		Column("id", "organization_id", "business_unit_id", "paid_at").
		Where("status = ?", "Paid").
		Where("paid_journal_batch_id IS NULL").
		Limit(1).
		Scan(ctx, &settlement))

	preview, err := repairJournals(ctx, conn, settlement.OrganizationID, true)
	require.NoError(t, err)
	assert.Zero(t, preview.PaymentsJournaled)
	assert.True(t, slices.ContainsFunc(preview.Skipped, func(skip journalrepairservice.Skip) bool {
		return skip.Kind == journalrepairservice.KindDriverSettlement && skip.ID == settlement.ID
	}))

	account := func(code string) pulid.ID {
		var row struct {
			ID pulid.ID `bun:"id"`
		}
		require.NoError(t, db.NewSelect().
			Table("gl_accounts").
			Column("id").
			Where("organization_id = ?", settlement.OrganizationID).
			Where("business_unit_id = ?", settlement.BusinessUnitID).
			Where("account_code = ?", code).
			Scan(ctx, &row))
		return row.ID
	}
	_, err = db.NewUpdate().
		Table("driver_settlements").
		Set("posted_payable_account_id = ?", account("2010")).
		Where("id = ?", settlement.ID).
		Exec(ctx)
	require.NoError(t, err)
	_, err = db.NewUpdate().
		Table("accounting_controls").
		Set("default_cash_account_id = ?", account("1010")).
		Where("organization_id = ?", settlement.OrganizationID).
		Exec(ctx)
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

	var stamped struct {
		BatchID *pulid.ID `bun:"paid_journal_batch_id"`
	}
	require.NoError(t, db.NewSelect().
		Table("driver_settlements").
		Column("paid_journal_batch_id").
		Where("id = ?", settlement.ID).
		Scan(ctx, &stamped))
	require.NotNil(t, stamped.BatchID)

	var batch struct {
		BatchNumber string `bun:"batch_number"`
	}
	require.NoError(t, db.NewSelect().
		Table("journal_batches").
		Column("batch_number").
		Where("id = ?", *stamped.BatchID).
		Scan(ctx, &batch))
	assert.NotEmpty(t, batch.BatchNumber)

	again, err := repairJournals(ctx, conn, settlement.OrganizationID, false)
	require.NoError(t, err)
	assert.Zero(t, again.PaymentsJournaled)
	assert.Zero(t, again.PaymentsLinked)
}
