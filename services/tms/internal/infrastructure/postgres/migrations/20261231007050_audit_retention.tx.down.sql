DELETE FROM trenova_rls.append_only_tables
WHERE "table_name" IN ('audit_entries', 'auth_events');

--bun:split
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
