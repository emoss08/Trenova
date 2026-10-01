CREATE TABLE IF NOT EXISTS trenova_rls.append_only_tables(
    "table_name" name NOT NULL,
    "reason" text NOT NULL,
    CONSTRAINT "pk_append_only_tables" PRIMARY KEY ("table_name"),
    CONSTRAINT "ck_append_only_tables_reason" CHECK (length(btrim("reason")) > 0)
);

--bun:split
REVOKE ALL ON trenova_rls.append_only_tables FROM PUBLIC;

--bun:split
GRANT SELECT ON trenova_rls.append_only_tables TO trenova_tenant, trenova_rls_bypass;

--bun:split
COMMENT ON TABLE trenova_rls.append_only_tables IS 'Tables in public the application role may only read and append to; reconcile revokes UPDATE and DELETE on them from trenova_tenant';

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

--bun:split
REVOKE ALL ON FUNCTION trenova_rls.reconcile() FROM PUBLIC;

--bun:split
CREATE OR REPLACE FUNCTION prevent_audit_modification()
    RETURNS TRIGGER
    LANGUAGE plpgsql
    SET search_path = pg_catalog, pg_temp
AS $$
BEGIN
    IF TG_OP = 'DELETE'
        AND current_setting('trenova.audit_retention', true) = 'on'
        AND pg_has_role(current_user, 'trenova_rls_bypass', 'MEMBER')
        AND (
            NOT OLD.critical
            OR OLD.timestamp < EXTRACT(epoch FROM current_timestamp - interval '365 days')::bigint
        )
    THEN
        RETURN OLD;
    END IF;

    RAISE EXCEPTION 'Modifications are not allowed on audit_entries (append-only table)'
        USING ERRCODE = 'insufficient_privilege',
              HINT = 'Only the retention sweep may delete audit entries, and never a critical entry younger than 365 days';
END
$$;

--bun:split
CREATE OR REPLACE FUNCTION prevent_auth_event_modification()
    RETURNS TRIGGER
    LANGUAGE plpgsql
    SET search_path = pg_catalog, pg_temp
AS $$
BEGIN
    IF TG_OP = 'DELETE'
        AND current_setting('trenova.audit_retention', true) = 'on'
        AND pg_has_role(current_user, 'trenova_rls_bypass', 'MEMBER')
        AND OLD.occurred_at < EXTRACT(epoch FROM current_timestamp - interval '365 days')::bigint
    THEN
        RETURN OLD;
    END IF;

    IF TG_OP = 'UPDATE'
        AND to_jsonb(NEW) - 'user_id' - 'identity_provider_id' = to_jsonb(OLD) - 'user_id' - 'identity_provider_id'
        AND (NEW.user_id IS NULL OR NEW.user_id = OLD.user_id)
        AND (NEW.identity_provider_id IS NULL OR NEW.identity_provider_id = OLD.identity_provider_id)
    THEN
        RETURN NEW;
    END IF;

    RAISE EXCEPTION 'Modifications are not allowed on auth_events (append-only table)'
        USING ERRCODE = 'insufficient_privilege',
              HINT = 'Authentication events are kept for 365 days and removed only by the retention sweep';
END
$$;

--bun:split
CREATE TRIGGER enforce_auth_event_append_only
    BEFORE UPDATE OR DELETE ON "auth_events"
    FOR EACH ROW
    EXECUTE FUNCTION prevent_auth_event_modification();

--bun:split
CREATE INDEX IF NOT EXISTS "idx_auth_events_occurred_at" ON "auth_events"("occurred_at");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_auth_events_org_occurred_at" ON "auth_events"("organization_id", "occurred_at" DESC);

--bun:split
INSERT INTO trenova_rls.append_only_tables("table_name", "reason")
VALUES
    ('audit_entries', 'Audit trail; only the retention sweep, as a bypass member, may delete expired rows'),
    ('auth_events', 'Authentication log; only the retention sweep, as a bypass member, may delete expired rows')
ON CONFLICT ("table_name") DO UPDATE SET "reason" = EXCLUDED."reason";

--bun:split
SELECT trenova_rls.reconcile();
