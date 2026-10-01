DO $$
DECLARE
    secured record;
    existing record;
BEGIN
    FOR secured IN
        SELECT c.oid::regclass AS target
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = 'public'
            AND c.relkind IN ('r', 'p')
            AND (c.relrowsecurity OR c.relforcerowsecurity)
    LOOP
        FOR existing IN
            SELECT p.polname
            FROM pg_policy p
            WHERE p.polrelid = secured.target
                AND p.polname IN ('tenant_isolation', 'tenant_read', 'tenant_insert', 'tenant_update', 'tenant_delete', 'rls_bypass')
        LOOP
            EXECUTE format('DROP POLICY %I ON %s', existing.polname, secured.target);
        END LOOP;

        EXECUTE format('ALTER TABLE %s NO FORCE ROW LEVEL SECURITY', secured.target);
        EXECUTE format('ALTER TABLE %s DISABLE ROW LEVEL SECURITY', secured.target);
    END LOOP;
END
$$;

--bun:split
ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES FROM trenova_tenant, trenova_rls_bypass;

--bun:split
ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE USAGE, SELECT, UPDATE ON SEQUENCES FROM trenova_tenant, trenova_rls_bypass;

--bun:split
REVOKE ALL ON ALL TABLES IN SCHEMA public FROM trenova_tenant, trenova_rls_bypass;

--bun:split
REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM trenova_tenant, trenova_rls_bypass;

--bun:split
REVOKE USAGE ON SCHEMA public FROM trenova_tenant, trenova_rls_bypass;

--bun:split
DROP SCHEMA IF EXISTS trenova_rls CASCADE;
