DROP TABLE IF EXISTS "bun_migrations", "bun_migration_locks";

DELETE FROM trenova_rls.global_tables
WHERE "table_name" IN ('bun_migrations', 'bun_migration_locks');

INSERT INTO trenova_rls.global_tables("table_name", "reason")
VALUES
    ('keel_migrations', 'Migration bookkeeping'),
    ('keel_seeds', 'Seed bookkeeping'),
    ('keel_seed_tenants', 'Seed bookkeeping'),
    ('keel_meta', 'Database environment mark')
ON CONFLICT ("table_name") DO UPDATE SET "reason" = EXCLUDED."reason";

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
                'seed_created_entities', 'seed_history'
            )
    LOOP
        EXECUTE format('REVOKE INSERT, UPDATE, DELETE ON %s FROM trenova_tenant', candidate.target);
    END LOOP;

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
