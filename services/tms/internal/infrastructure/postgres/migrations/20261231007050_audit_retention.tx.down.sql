GRANT UPDATE, DELETE ON "audit_entries", "auth_events" TO trenova_tenant;

--bun:split
DROP INDEX IF EXISTS "idx_auth_events_org_occurred_at";

--bun:split
DROP INDEX IF EXISTS "idx_auth_events_occurred_at";

--bun:split
DROP TRIGGER IF EXISTS enforce_auth_event_append_only ON "auth_events";

--bun:split
DROP FUNCTION IF EXISTS prevent_auth_event_modification();

--bun:split
CREATE OR REPLACE FUNCTION prevent_audit_modification()
    RETURNS TRIGGER
    AS $$
BEGIN
    RAISE EXCEPTION 'Modifications are not allowed on audit_entries (append-only table)';
END;
$$
LANGUAGE plpgsql;

--bun:split
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
            AND c.relname IN ('bun_migrations', 'bun_migration_locks', 'seed_created_entities', 'seed_history')
    LOOP
        EXECUTE format('REVOKE INSERT, UPDATE, DELETE ON %s FROM trenova_tenant', candidate.target);
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

--bun:split
REVOKE ALL ON FUNCTION trenova_rls.reconcile() FROM PUBLIC;

--bun:split
DROP TABLE IF EXISTS trenova_rls.append_only_tables;
