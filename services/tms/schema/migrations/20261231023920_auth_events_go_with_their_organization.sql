-- A company's sign-in history goes with the company. Removing a tenant from a shared server
-- deletes its organization, and the cascade reaches auth_events as it reaches audit_entries. The
-- audit log's guard lets a cascade through; this one refused it, so the organization could not
-- be deleted and the removal never finished. A delete that names auth_events itself is refused
-- as before, and so is every update but the two a deleted user or identity provider causes.
CREATE OR REPLACE FUNCTION prevent_auth_event_modification()
    RETURNS TRIGGER
    LANGUAGE plpgsql
    SET search_path = pg_catalog, pg_temp
AS $$
BEGIN
    IF TG_OP = 'DELETE' AND pg_trigger_depth() > 1 THEN
        RETURN OLD;
    END IF;

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
              HINT = 'Authentication events are kept for 365 days and removed only by the retention sweep or with their organization';
END
$$;
