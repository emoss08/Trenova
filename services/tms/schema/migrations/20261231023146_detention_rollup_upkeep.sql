-- keel's rollup state holds each rollup's name and when it was built: no tenant data.
INSERT INTO trenova_rls.global_tables("table_name", "reason")
VALUES ('keel_rollup_state', 'Rollup bookkeeping: each rollup''s name and when it was built')
ON CONFLICT ("table_name") DO UPDATE SET "reason" = EXCLUDED."reason";

-- The dirty marks name a tenant and the days it had activity, so a tenant reads none of them
-- and writes only its own. A source table's triggers append the marks as the writing role
-- (INSERT … ON CONFLICT DO NOTHING checks the insert policy alone); only the system-scoped
-- refresh, which bypasses the policy, reads and removes them. The key is keel's tenant key: the
-- business unit and organization ids, '\' and '/' escaped, joined by '/'.
SELECT trenova_rls.apply_policy(
    'public.keel_rollup_dirty',
    'false',
    'tenant_key = (SELECT replace(replace(t.bu_id, ''\'', ''\\''), ''/'', ''\/'') || ''/'' || '
        || 'replace(replace(t.org_id, ''\'', ''\\''), ''/'', ''\/'') FROM trenova_rls.tenant() AS t)'
);

INSERT INTO trenova_rls.append_only_tables("table_name", "reason")
VALUES ('keel_rollup_dirty', 'Source triggers append refresh marks; only the system refresh removes them')
ON CONFLICT ("table_name") DO UPDATE SET "reason" = EXCLUDED."reason";

-- A rollup's rows are computed from its source by the system refresh alone: the application
-- role reads them and never writes them, or a tenant could rewrite its own figures. reconcile()
-- finds every rollup table through keel_rollup_state, so a new rollup is covered by the
-- migration that creates it.
CREATE OR REPLACE FUNCTION trenova_rls.reconcile()
    RETURNS integer
    LANGUAGE plpgsql
    SET search_path = pg_catalog, pg_temp
AS $$
DECLARE
    candidate record;
    applied integer := 0;
BEGIN
    EXECUTE 'GRANT USAGE ON SCHEMA public TO trenova_tenant, trenova_rls_bypass';
    EXECUTE 'GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO trenova_tenant, trenova_rls_bypass';
    EXECUTE 'GRANT USAGE, SELECT, UPDATE ON ALL SEQUENCES IN SCHEMA public TO trenova_tenant, trenova_rls_bypass';
    FOR candidate IN
        SELECT c.oid::regclass AS target
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = 'public'
            AND c.relname IN (
                'keel_migrations', 'keel_seeds', 'keel_seed_tenants', 'keel_meta',
                'seed_created_entities', 'seed_history', 'keel_rollup_state'
            )
    LOOP
        EXECUTE format('REVOKE INSERT, UPDATE, DELETE ON %s FROM trenova_tenant', candidate.target);
    END LOOP;

    IF to_regclass('public.keel_rollup_state') IS NOT NULL THEN
        FOR candidate IN
            SELECT c.oid::regclass AS target
            FROM pg_class c
            JOIN pg_namespace n ON n.oid = c.relnamespace
            JOIN public.keel_rollup_state s ON s.rollup = c.relname
            WHERE n.nspname = 'public'
                AND c.relkind IN ('r', 'p')
        LOOP
            EXECUTE format('REVOKE INSERT, UPDATE, DELETE ON %s FROM trenova_tenant', candidate.target);
        END LOOP;
    END IF;

    FOR candidate IN
        SELECT c.oid::regclass AS target
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        JOIN trenova_rls.append_only_tables a ON a.table_name = c.relname
        WHERE n.nspname = 'public'
    LOOP
        EXECUTE format('REVOKE UPDATE, DELETE ON %s FROM trenova_tenant', candidate.target);
    END LOOP;

    FOR candidate IN
        SELECT c.oid::regclass AS target, bu.attnotnull AS bu_not_null
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        JOIN pg_attribute org ON org.attrelid = c.oid AND org.attname = 'organization_id' AND NOT org.attisdropped
        JOIN pg_attribute bu ON bu.attrelid = c.oid AND bu.attname = 'business_unit_id' AND NOT bu.attisdropped
        WHERE n.nspname = 'public'
            AND c.relkind IN ('r', 'p')
            AND org.attnotnull
            AND NOT c.relrowsecurity
            AND NOT EXISTS (SELECT 1 FROM trenova_rls.global_tables g WHERE g.table_name = c.relname)
    LOOP
        PERFORM trenova_rls.apply_policy(
            candidate.target,
            trenova_rls.standard_expr('organization_id', 'business_unit_id', NOT candidate.bu_not_null)
        );
        applied := applied + 1;
    END LOOP;

    RETURN applied;
END
$$;

REVOKE ALL ON FUNCTION trenova_rls.reconcile() FROM PUBLIC;

-- A rollup bucketed in the organization's time zone holds days that end at that zone's
-- midnight. When the zone changes, every rollup marks the organization for a whole rebuild
-- ('-infinity'), which the next refresh carries out. The update runs in the organization's own
-- scope or the system scope, so the mark passes the dirty marks' insert policy.
CREATE OR REPLACE FUNCTION trenova_mark_rollups_for_timezone()
    RETURNS trigger
    LANGUAGE plpgsql
AS $$
BEGIN
    INSERT INTO keel_rollup_dirty (rollup, tenant_key, bucket, xid)
    SELECT s.rollup,
           replace(replace(NEW."business_unit_id"::text, '\', '\\'), '/', '\/') || '/' ||
           replace(replace(NEW."id"::text, '\', '\\'), '/', '\/'),
           '-infinity'::timestamptz,
           pg_current_xact_id()
    FROM keel_rollup_state s
    ON CONFLICT DO NOTHING;
    RETURN NULL;
END
$$;

CREATE OR REPLACE TRIGGER "trg_organizations_timezone_rollups"
    AFTER UPDATE OF "timezone" ON "organizations"
    FOR EACH ROW
    WHEN (OLD."timezone" IS DISTINCT FROM NEW."timezone")
    EXECUTE FUNCTION trenova_mark_rollups_for_timezone();
