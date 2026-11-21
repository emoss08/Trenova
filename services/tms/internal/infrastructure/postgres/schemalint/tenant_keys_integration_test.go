//go:build integration

package schemalint

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantkeys"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	tenantKeysOutEnv     = "TRENOVA_TENANT_KEYS_OUT"
	tenantKeysVersionEnv = "TRENOVA_TENANT_KEYS_VERSION"

	tenantKeysGuidance = "A foreign key between tenant tables carries the row's tenant, so a write " +
		"that names another tenant's record fails at the database: FOREIGN KEY (\"customer_id\", " +
		"\"business_unit_id\") REFERENCES \"customers\"(\"id\", \"business_unit_id\"), with ON " +
		"DELETE SET NULL (\"customer_id\") where the reference is cleared " +
		"(docs/engineering/backend/data-access.md, \"Tenant keys\"). A key that cannot carry it " +
		"is listed with its reason in internal/infrastructure/postgres/tenantkeys/exceptions.go. " +
		"Run this test with " + tenantKeysOutEnv + "=<dir> and " + tenantKeysVersionEnv +
		"=<first version> to write the migrations that convert and index the keys below."
)

func loadTenantKeys(t *testing.T) *tenantkeys.Inventory {
	t.Helper()
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	cat, err := tenantkeys.Load(ctx, db)
	require.NoError(t, err)
	return tenantkeys.Classify(cat)
}

func writeTenantKeyMigrations(t *testing.T, inv *tenantkeys.Inventory) {
	t.Helper()
	dir := os.Getenv(tenantKeysOutEnv)
	if dir == "" {
		return
	}
	version, err := strconv.ParseUint(os.Getenv(tenantKeysVersionEnv), 10, 64)
	require.NoError(t, err, "%s names the first migration's version", tenantKeysVersionEnv)
	migrations, err := tenantkeys.Render(inv)
	require.NoError(t, err)
	for i, m := range migrations {
		name := m.File(strconv.FormatUint(version+uint64(i)*10, 10))
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(m.SQL), 0o600))
		t.Logf("wrote %s", name)
	}
}

func TestForeignKeysBetweenTenantTablesCarryTheTenant(t *testing.T) {
	inv := loadTenantKeys(t)
	writeTenantKeyMigrations(t, inv)

	t.Logf("foreign keys between tenant tables: %d tenant column, %d held, %d convertible, "+
		"%d declared exceptions, %d undeclared",
		inv.Count(tenantkeys.ClassTenantColumn), inv.Count(tenantkeys.ClassHeld),
		inv.Count(tenantkeys.ClassConvertible), inv.Count(tenantkeys.ClassDeclared),
		inv.Count(tenantkeys.ClassUndeclared))

	loose := make([]string, 0)
	for _, k := range inv.Keys {
		switch k.Class {
		case tenantkeys.ClassConvertible:
			loose = append(
				loose,
				k.FK.Name+": "+k.FK.Child+"("+strings.Join(k.FK.ChildColumns, ", ")+
					") names "+k.FK.Parent+" by id alone; carry "+k.Form.Child,
			)
		case tenantkeys.ClassUndeclared:
			loose = append(
				loose,
				k.FK.Name+": "+k.FK.Child+"("+strings.Join(k.FK.ChildColumns, ", ")+
					") names "+k.FK.Parent+" by id alone and neither table can carry the tenant; "+
					"declare it with its reason",
			)
		}
	}
	assert.Empty(t, loose, "%s\n%s", tenantKeysGuidance, strings.Join(loose, "\n"))
	assert.Empty(t, inv.Stale, "declared exceptions that no longer apply; remove them from "+
		"tenantkeys/exceptions.go:\n%s", strings.Join(inv.Stale, "\n"))
	assert.Empty(t, inv.Unsupported, "keys the generator cannot convert; write their migration "+
		"by hand:\n%s", strings.Join(inv.Unsupported, "\n"))
}

func TestEveryForeignKeyBetweenTenantTablesIsIndexed(t *testing.T) {
	inv := loadTenantKeys(t)

	missing := make([]string, 0, len(inv.Indexes))
	for _, idx := range inv.Indexes {
		line := idx.Table + "(" + strings.Join(idx.Columns, ", ") + ")"
		if idx.Predicate != "" {
			line += " WHERE " + idx.Predicate
		}
		missing = append(missing, line)
	}
	assert.Empty(t, missing, "foreign keys between tenant tables whose referring columns lead no "+
		"index: PostgreSQL's delete checks and a tenant removal scan every tenant's rows of "+
		"the table. Add an index whose leading columns are the key's reference columns (partial "+
		"WHERE ... IS NOT NULL when they may be empty), or its whole key.\n%s\n%s",
		tenantKeysGuidance, strings.Join(missing, "\n"))
	for _, u := range inv.Uniques {
		t.Errorf("parent %s needs a unique index on (%s)", u.Table, strings.Join(u.Columns, ", "))
	}
}
