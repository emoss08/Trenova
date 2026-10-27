//go:build integration

package schemalint

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const rlsGuidance = "Every table in public holds tenant data unless trenova_rls.global_tables " +
	"says otherwise. End the migration that creates a tenant table with " +
	"SELECT trenova_rls.reconcile(); (organization_id and business_unit_id) or " +
	"SELECT trenova_rls.apply_policy(...); for any other shape. See " +
	"docs/engineering/row-level-security.md."

func TestEveryTenantTableIsForcedUnderRowLevelSecurity(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	unprotected, err := keel.QueryAll(ctx, db, pgx.RowTo[string], `
		SELECT c.relname
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public'
			AND c.relkind IN ('r', 'p')
			AND NOT (c.relrowsecurity AND c.relforcerowsecurity)
			AND NOT EXISTS (SELECT 1 FROM trenova_rls.global_tables g WHERE g.table_name = c.relname)
		ORDER BY c.relname`)
	require.NoError(t, err)

	assert.Empty(t, unprotected, "tables without forced row-level security: %s\n%s",
		strings.Join(unprotected, ", "), rlsGuidance)
}

func TestEveryProtectedTableHasTenantAndBypassPolicies(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	missing, err := keel.QueryAll(ctx, db, pgx.RowTo[string], `
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
		ORDER BY c.relname`)
	require.NoError(t, err)

	assert.Empty(t, missing, "tables with incomplete policies: %s\n%s", strings.Join(missing, "; "), rlsGuidance)
}

func TestTenantPoliciesNeverAdmitEveryRow(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	permissive, err := keel.QueryAll(ctx, db, pgx.RowTo[string], `
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
		ORDER BY 1`)
	require.NoError(t, err)

	assert.Empty(t, permissive, "tenant policies that do not consult the signed scope: %s",
		strings.Join(permissive, ", "))
}

func TestStandardTenantPoliciesVerifyTheScopeOnce(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	twice, err := keel.QueryAll(ctx, db, pgx.RowTo[string], `
		SELECT tablename
		FROM pg_policies
		WHERE schemaname = 'public'
			AND policyname = 'tenant_isolation'
			AND regexp_replace(regexp_replace(qual, '\s+', '', 'g'), '::text', '', 'g') IN (
				'(((organization_id)=(SELECTtrenova_rls.org_id()ASorg_id))AND((business_unit_id)=(SELECTtrenova_rls.bu_id()ASbu_id)))',
				'((organization_id=(SELECTtrenova_rls.org_id()ASorg_id))AND(business_unit_id=(SELECTtrenova_rls.bu_id()ASbu_id)))'
			)
		ORDER BY 1`)
	require.NoError(t, err)
	assert.Empty(t, twice,
		"tenant policies that verify the signed scope twice instead of once through "+
			"trenova_rls.tenant(): %s", strings.Join(twice, ", "))

	once, err := keel.QueryValue[int64](ctx, db, `
		SELECT count(*)
		FROM pg_policies
		WHERE schemaname = 'public'
			AND policyname = 'tenant_isolation'
			AND qual LIKE '%trenova_rls.tenant()%'`)
	require.NoError(t, err)
	assert.Positive(t, once, "no tenant policy compares the row with trenova_rls.tenant()")
}

func TestGlobalTablesHoldNoTenantColumns(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	tenanted, err := keel.QueryAll(ctx, db, pgx.RowTo[string], `
		SELECT g.table_name || '.' || a.attname
		FROM trenova_rls.global_tables g
		JOIN pg_class c ON c.relname = g.table_name
		JOIN pg_namespace n ON n.oid = c.relnamespace AND n.nspname = 'public'
		JOIN pg_attribute a ON a.attrelid = c.oid AND NOT a.attisdropped AND a.attnum > 0
		WHERE a.attname IN ('organization_id', 'business_unit_id')
			AND g.table_name <> 'business_units'
		ORDER BY 1`)
	require.NoError(t, err)

	assert.Empty(t, tenanted, "tables registered as global carry tenant columns: %s",
		strings.Join(tenanted, ", "))
}

func TestViewsRunAsTheQueryingRole(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	definerViews, err := keel.QueryAll(ctx, db, pgx.RowTo[string], `
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
		ORDER BY c.relname`)
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

func securityDefinerFunctions(t *testing.T, ctx context.Context, db *keel.DB) []string {
	t.Helper()

	definers, err := keel.QueryAll(ctx, db, pgx.RowTo[string], `
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
		ORDER BY 1`)
	require.NoError(t, err)

	return definers
}
