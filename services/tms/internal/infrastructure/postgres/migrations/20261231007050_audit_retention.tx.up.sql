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
