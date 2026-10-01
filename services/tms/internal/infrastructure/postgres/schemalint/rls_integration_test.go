//go:build integration

package schemalint

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

const rlsGuidance = "Every table in public holds tenant data unless trenova_rls.global_tables " +
	"says otherwise. End the migration that creates a tenant table with " +
	"SELECT trenova_rls.reconcile(); (organization_id and business_unit_id) or " +
	"SELECT trenova_rls.apply_policy(...); for any other shape. See " +
	"docs/engineering/row-level-security.md."

func TestEveryTenantTableIsForcedUnderRowLevelSecurity(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	var unprotected []string
	err := db.NewRaw(`
		SELECT c.relname
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public'
			AND c.relkind IN ('r', 'p')
			AND NOT (c.relrowsecurity AND c.relforcerowsecurity)
			AND NOT EXISTS (SELECT 1 FROM trenova_rls.global_tables g WHERE g.table_name = c.relname)
		ORDER BY c.relname`).Scan(ctx, &unprotected)
	require.NoError(t, err)

	assert.Empty(t, unprotected, "tables without forced row-level security: %s\n%s",
		strings.Join(unprotected, ", "), rlsGuidance)
}

func TestEveryProtectedTableHasTenantAndBypassPolicies(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	var missing []string
	err := db.NewRaw(`
		SELECT c.relname || ' (' ||
			CASE
				WHEN NOT EXISTS (
					SELECT 1 FROM pg_policies p
					WHERE p.schemaname = 'public' AND p.tablename = c.relname
						AND 'trenova_rls_bypass' = ANY (p.roles)
				) THEN 'no bypass policy'
				ELSE 'tenant policies do not cover ' || array_to_string(ARRAY(
					SELECT cmd FROM unnest(ARRAY['SELECT', 'INSERT', 'UPDATE', 'DELETE']) AS cmd
					WHERE NOT EXISTS (
						SELECT 1 FROM pg_policies p
						WHERE p.schemaname = 'public' AND p.tablename = c.relname
							AND 'trenova_tenant' = ANY (p.roles)
							AND p.cmd IN ('ALL', cmd)
					)
				), ', ')
			END || ')'
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public'
			AND c.relkind IN ('r', 'p')
			AND c.relrowsecurity
			AND (
				NOT EXISTS (
					SELECT 1 FROM pg_policies p
					WHERE p.schemaname = 'public' AND p.tablename = c.relname
						AND 'trenova_rls_bypass' = ANY (p.roles)
				)
				OR EXISTS (
					SELECT 1 FROM unnest(ARRAY['SELECT', 'INSERT', 'UPDATE', 'DELETE']) AS cmd
					WHERE NOT EXISTS (
						SELECT 1 FROM pg_policies p
						WHERE p.schemaname = 'public' AND p.tablename = c.relname
							AND 'trenova_tenant' = ANY (p.roles)
							AND p.cmd IN ('ALL', cmd)
					)
				)
			)
		ORDER BY c.relname`).Scan(ctx, &missing)
	require.NoError(t, err)

	assert.Empty(t, missing, "tables with incomplete policies: %s\n%s", strings.Join(missing, "; "), rlsGuidance)
}

func TestTenantPoliciesNeverAdmitEveryRow(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	var permissive []string
	err := db.NewRaw(`
		SELECT tablename || '.' || policyname
		FROM pg_policies
		WHERE schemaname = 'public'
			AND 'trenova_tenant' = ANY (roles)
			AND (
				btrim(coalesce(qual, '')) IN ('true', '(true)')
				OR btrim(coalesce(with_check, '')) IN ('true', '(true)')
				OR (qual IS NOT NULL AND qual NOT LIKE '%trenova_rls.%')
				OR (with_check IS NOT NULL AND with_check NOT LIKE '%trenova_rls.%')
			)
		ORDER BY 1`).Scan(ctx, &permissive)
	require.NoError(t, err)

	assert.Empty(t, permissive, "tenant policies that do not consult the signed scope: %s",
		strings.Join(permissive, ", "))
}

func TestGlobalTablesHoldNoTenantColumns(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	var tenanted []string
	err := db.NewRaw(`
		SELECT g.table_name || '.' || a.attname
		FROM trenova_rls.global_tables g
		JOIN pg_class c ON c.relname = g.table_name
		JOIN pg_namespace n ON n.oid = c.relnamespace AND n.nspname = 'public'
		JOIN pg_attribute a ON a.attrelid = c.oid AND NOT a.attisdropped AND a.attnum > 0
		WHERE a.attname IN ('organization_id', 'business_unit_id')
			AND g.table_name <> 'business_units'
		ORDER BY 1`).Scan(ctx, &tenanted)
	require.NoError(t, err)

	assert.Empty(t, tenanted, "tables registered as global carry tenant columns: %s",
		strings.Join(tenanted, ", "))
}

func TestViewsRunAsTheQueryingRole(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	var definerViews []string
	err := db.NewRaw(`
		SELECT c.relname
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public'
			AND c.relkind IN ('v', 'm')
			AND NOT EXISTS (
				SELECT 1 FROM pg_depend d
				WHERE d.objid = c.oid AND d.deptype = 'e'
			)
			AND NOT coalesce('security_invoker=true' = ANY (c.reloptions), false)
		ORDER BY c.relname`).Scan(ctx, &definerViews)
	require.NoError(t, err)

	assert.Empty(t, definerViews, "views that would read tenant tables as their owner and bypass "+
		"row-level security; create them WITH (security_invoker = true): %s",
		strings.Join(definerViews, ", "))
}

func TestNoFunctionRunsAsItsOwnerOutsideTheScopeSchema(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	definers := securityDefinerFunctions(t, ctx, db)

	assert.Empty(t, definers, "SECURITY DEFINER functions bypass row-level security for whoever "+
		"calls them: %s", strings.Join(definers, ", "))
}

func securityDefinerFunctions(t *testing.T, ctx context.Context, db *bun.DB) []string {
	t.Helper()

	var definers []string
	err := db.NewRaw(`
		SELECT n.nspname || '.' || p.proname
		FROM pg_proc p
		JOIN pg_namespace n ON n.oid = p.pronamespace
		WHERE p.prosecdef
			AND n.nspname NOT IN ('pg_catalog', 'information_schema')
			AND NOT (n.nspname = 'trenova_rls' AND p.proname = 'claims')
			AND NOT EXISTS (
				SELECT 1 FROM pg_depend d
				WHERE d.objid = p.oid AND d.deptype = 'e'
			)
		ORDER BY 1`).Scan(ctx, &definers)
	require.NoError(t, err)

	return definers
}
