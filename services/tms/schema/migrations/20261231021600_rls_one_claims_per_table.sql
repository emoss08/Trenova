-- The standard tenant policy verified the signed scope twice per table reference: once through
-- trenova_rls.org_id() and again through trenova_rls.bu_id(), each a separate InitPlan calling
-- trenova_rls.claims() (a plpgsql function that reads the scope key and computes two SHA-256s).
-- Measured on PostgreSQL 18 that cost about 120 microseconds per tenant table per statement.
-- trenova_rls.tenant() returns both claims from one verification, and the standard expression
-- compares the row with it in a single InitPlan, which roughly halves the per-table cost.

CREATE OR REPLACE FUNCTION trenova_rls.tenant(OUT org_id text, OUT bu_id text)
    LANGUAGE sql
    STABLE
    PARALLEL RESTRICTED
AS $$
    SELECT c[3], c[4] FROM trenova_rls.claims() AS c
$$;

REVOKE ALL ON FUNCTION trenova_rls.tenant() FROM PUBLIC;

GRANT EXECUTE ON FUNCTION trenova_rls.tenant() TO trenova_tenant, trenova_rls_bypass;

CREATE OR REPLACE FUNCTION trenova_rls.standard_expr(org_column text DEFAULT 'organization_id', bu_column text DEFAULT 'business_unit_id', bu_nullable boolean DEFAULT false)
    RETURNS text
    LANGUAGE sql
    IMMUTABLE
AS $$
    SELECT CASE
        WHEN bu_column IS NULL THEN format('%I = (SELECT trenova_rls.org_id())', org_column)
        WHEN bu_nullable THEN format(
            '%1$I = (SELECT trenova_rls.org_id()) AND (%2$I IS NULL OR %2$I = (SELECT trenova_rls.bu_id()))',
            org_column, bu_column
        )
        ELSE format('(%I, %I) = (SELECT t.org_id, t.bu_id FROM trenova_rls.tenant() AS t)', org_column, bu_column)
    END
$$;

-- Re-apply the standard policy to every table that carries the old two-InitPlan form: a
-- tenant_isolation policy on non-null organization_id and business_unit_id columns whose
-- expression is exactly the old standard comparison, as pg_get_expr prints it (with the
-- ::text casts of varchar columns, or without them for text columns). Hand-written policies
-- (memberships, cross-organization rows, user-owned rows) do not match and keep their own
-- expressions.
DO $reapply$
DECLARE
    candidate record;
BEGIN
    FOR candidate IN
        SELECT c.oid::regclass AS target
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        JOIN pg_policy p ON p.polrelid = c.oid AND p.polname = 'tenant_isolation'
        JOIN pg_attribute org ON org.attrelid = c.oid AND org.attname = 'organization_id' AND NOT org.attisdropped
        JOIN pg_attribute bu ON bu.attrelid = c.oid AND bu.attname = 'business_unit_id' AND NOT bu.attisdropped
        WHERE n.nspname = 'public'
            AND c.relkind IN ('r', 'p')
            AND org.attnotnull
            AND bu.attnotnull
            AND regexp_replace(pg_get_expr(p.polqual, p.polrelid), '\s+', '', 'g') IN (
                '(((organization_id)::text=(SELECTtrenova_rls.org_id()ASorg_id))AND((business_unit_id)::text=(SELECTtrenova_rls.bu_id()ASbu_id)))',
                '((organization_id=(SELECTtrenova_rls.org_id()ASorg_id))AND(business_unit_id=(SELECTtrenova_rls.bu_id()ASbu_id)))'
            )
    LOOP
        PERFORM trenova_rls.apply_policy(candidate.target, trenova_rls.standard_expr());
    END LOOP;
END
$reapply$;
