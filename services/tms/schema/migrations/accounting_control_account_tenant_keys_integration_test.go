//go:build integration

package migrations_test

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/schema/migrations"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const accountingControlAccountTenantKeys = "20261231023449_accounting_control_account_tenant_keys"

type accountingControlAccountKey struct {
	column     string
	constraint string
}

var accountingControlAccountKeys = []accountingControlAccountKey{
	{"default_cash_account_id", "fk_accounting_controls_default_cash_account"},
	{"default_unapplied_cash_account_id", "fk_accounting_controls_default_unapplied_cash_account"},
	{
		"default_driver_pay_expense_account_id",
		"fk_accounting_controls_default_driver_pay_expense_account",
	},
	{
		"default_purchased_transportation_account_id",
		"fk_accounting_controls_default_purchased_transportation_account",
	},
	{
		"default_settlements_payable_account_id",
		"fk_accounting_controls_default_settlements_payable_account",
	},
	{"default_driver_advance_account_id", "fk_accounting_controls_default_driver_advance_account"},
	{
		"default_escrow_liability_account_id",
		"fk_accounting_controls_default_escrow_liability_account",
	},
	{
		"default_driver_reimbursement_account_id",
		"fk_accounting_controls_default_driver_reimbursement_account",
	},
	{
		"default_escrow_interest_expense_account_id",
		"fk_accounting_controls_default_escrow_interest_expense_account",
	},
}

func TestAccountingControlAccountTenantKeys_ClearsAccountsOutsideTheControlsTenant(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	pool := keel.Pool(db)
	own := seedtest.SeedFullTestData(t, ctx, db)
	other := seedtest.SeedAdditionalTenant(t, ctx, db, "ZZB")
	ownAccount := insertTestGLAccount(ctx, t, pool, own)
	otherAccount := insertTestGLAccount(ctx, t, pool, other)
	controlID := accountingControlOf(ctx, t, pool, own)

	for _, key := range accountingControlAccountKeys {
		_, err := pool.Exec(ctx,
			`ALTER TABLE "accounting_controls" DROP CONSTRAINT IF EXISTS `+
				pgx.Identifier{key.constraint}.Sanitize())
		require.NoError(t, err)
	}

	kept := accountingControlAccountKeys[2]
	missing := accountingControlAccountKeys[0]
	for _, key := range accountingControlAccountKeys {
		value := otherAccount
		switch key {
		case kept:
			value = ownAccount
		case missing:
			value = pulid.ID("gla_no_such_account")
		}
		setAccountingControlAccount(ctx, t, pool, controlID, key.column, value)
	}

	up, err := migrations.File(accountingControlAccountTenantKeys)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, up)
	require.NoError(t, err)

	for _, key := range accountingControlAccountKeys {
		stored := accountingControlAccount(ctx, t, pool, controlID, key.column)
		if key == kept {
			assert.Equal(t, ownAccount, stored, "%s names the control's own account", key.column)
			continue
		}
		assert.True(
			t,
			stored.IsNil(),
			"%s no longer names an account outside the tenant",
			key.column,
		)
	}

	for _, key := range accountingControlAccountKeys {
		_, err = pool.Exec(ctx,
			`UPDATE "accounting_controls" SET `+pgx.Identifier{key.column}.Sanitize()+` = $1
			WHERE "id" = $2`,
			otherAccount, controlID)
		pgErr, ok := errors.AsType[*pgconn.PgError](err)
		require.True(t, ok, "%s refuses another tenant's account, got %v", key.column, err)
		assert.Equal(t, "23503", pgErr.Code)
		assert.Equal(t, key.constraint, pgErr.ConstraintName)

		setAccountingControlAccount(ctx, t, pool, controlID, key.column, ownAccount)
	}
}

func insertTestGLAccount(
	ctx context.Context,
	t *testing.T,
	pool *pgxpool.Pool,
	tenant *seedtest.TestData,
) pulid.ID {
	t.Helper()

	accountTypeID := pulid.MustNew("at_")
	_, err := pool.Exec(
		ctx,
		`INSERT INTO "account_types" ("id", "organization_id", "business_unit_id", "code", "name", "category")
		VALUES ($1, $2, $3, 'TKEY', 'Tenant key test', 'Expense')`,
		accountTypeID,
		tenant.Organization.ID,
		tenant.BusinessUnit.ID,
	)
	require.NoError(t, err)

	accountID := pulid.MustNew("gla_")
	_, err = pool.Exec(
		ctx,
		`INSERT INTO "gl_accounts" ("id", "organization_id", "business_unit_id", "account_type_id", "account_code", "name")
		VALUES ($1, $2, $3, $4, 'TKEY-1', 'Tenant key test account')`,
		accountID,
		tenant.Organization.ID,
		tenant.BusinessUnit.ID,
		accountTypeID,
	)
	require.NoError(t, err)

	return accountID
}

func accountingControlOf(
	ctx context.Context,
	t *testing.T,
	pool *pgxpool.Pool,
	tenant *seedtest.TestData,
) pulid.ID {
	t.Helper()

	_, err := pool.Exec(ctx,
		`INSERT INTO "accounting_controls" ("id", "organization_id", "business_unit_id")
		VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
		pulid.MustNew("ac_"), tenant.Organization.ID, tenant.BusinessUnit.ID)
	require.NoError(t, err)

	var id pulid.ID
	require.NoError(t, pool.QueryRow(
		ctx,
		`SELECT "id" FROM "accounting_controls" WHERE "organization_id" = $1 AND "business_unit_id" = $2`,
		tenant.Organization.ID,
		tenant.BusinessUnit.ID,
	).Scan(&id))

	return id
}

func setAccountingControlAccount(
	ctx context.Context,
	t *testing.T,
	pool *pgxpool.Pool,
	controlID pulid.ID,
	column string,
	accountID pulid.ID,
) {
	t.Helper()

	_, err := pool.Exec(
		ctx,
		`UPDATE "accounting_controls" SET `+pgx.Identifier{
			column,
		}.Sanitize()+` = $1 WHERE "id" = $2`,
		accountID,
		controlID,
	)
	require.NoError(t, err)
}

func accountingControlAccount(
	ctx context.Context,
	t *testing.T,
	pool *pgxpool.Pool,
	controlID pulid.ID,
	column string,
) pulid.ID {
	t.Helper()

	var stored *string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT `+pgx.Identifier{column}.Sanitize()+` FROM "accounting_controls" WHERE "id" = $1`,
		controlID,
	).Scan(&stored))
	if stored == nil {
		return pulid.Nil
	}

	return pulid.ID(*stored)
}
