-- Nothing leaves the organization at Trenova support's hand. A support principal (a user id
-- starting sup_ in the transaction's signed scope) may not send or schedule a message to anyone
-- outside the organization, create or change a share, tracking or tender link or a scheduled
-- report, or change the text and recipients the organization's customers read. With no column
-- arguments the guard refuses any write it is attached to; with columns, a change to one of them,
-- or a new row carrying a value where it runs on insert. supportaccess.OutboundGuards in the TMS
-- names the same tables and columns. A send the system or a person started is not refused, since
-- its transaction's scope names them, not support.
CREATE OR REPLACE FUNCTION "support_outbound_guard"()
    RETURNS TRIGGER
    LANGUAGE plpgsql
    SET search_path = pg_catalog, pg_temp
AS $$
DECLARE
    scope_token text := current_setting('trenova.scope', true);
    after_row jsonb;
    before_row jsonb;
    guarded text;
BEGIN
    IF scope_token IS NULL OR split_part(scope_token, '.', 5) NOT LIKE 'sup\_%' THEN
        RETURN NEW;
    END IF;
    IF COALESCE(trenova_rls.user_id(), '') NOT LIKE 'sup\_%' THEN
        RETURN NEW;
    END IF;

    IF TG_NARGS = 0 THEN
        RAISE EXCEPTION 'Trenova support cannot send outside the organization (%)', TG_TABLE_NAME
            USING ERRCODE = 'TRS02';
    END IF;

    after_row := to_jsonb(NEW);
    IF TG_OP = 'UPDATE' THEN
        before_row := to_jsonb(OLD);
    END IF;

    FOREACH guarded IN ARRAY TG_ARGV LOOP
        IF (TG_OP = 'UPDATE' AND (after_row -> guarded) IS DISTINCT FROM (before_row -> guarded))
            OR (TG_OP = 'INSERT' AND COALESCE(after_row ->> guarded, '') NOT IN ('', '[]', '{}')) THEN
            RAISE EXCEPTION 'Trenova support cannot change what reaches people outside the organization (%.%)', TG_TABLE_NAME, guarded
                USING ERRCODE = 'TRS02';
        END IF;
    END LOOP;

    RETURN NEW;
END
$$;

COMMENT ON FUNCTION "support_outbound_guard"() IS 'Refuses a Trenova support principal a send, a link or a schedule that leaves the organization, and a change to text or recipients outside readers see';

DROP TRIGGER IF EXISTS "guard_support_outbound" ON "email_messages";

CREATE TRIGGER "guard_support_outbound"
    BEFORE INSERT ON "email_messages"
    FOR EACH ROW
    EXECUTE FUNCTION "support_outbound_guard"();

DROP TRIGGER IF EXISTS "guard_support_outbound" ON "edi_messages";

CREATE TRIGGER "guard_support_outbound"
    BEFORE INSERT OR UPDATE ON "edi_messages"
    FOR EACH ROW
    WHEN (NEW."direction" = 'Outbound')
    EXECUTE FUNCTION "support_outbound_guard"();

DROP TRIGGER IF EXISTS "guard_support_outbound" ON "factoring_submissions";

CREATE TRIGGER "guard_support_outbound"
    BEFORE INSERT OR UPDATE ON "factoring_submissions"
    FOR EACH ROW
    EXECUTE FUNCTION "support_outbound_guard"();

DROP TRIGGER IF EXISTS "guard_support_outbound" ON "factoring_transports";

CREATE TRIGGER "guard_support_outbound"
    BEFORE INSERT OR UPDATE ON "factoring_transports"
    FOR EACH ROW
    EXECUTE FUNCTION "support_outbound_guard"();

DROP TRIGGER IF EXISTS "guard_support_outbound" ON "detention_notices";

CREATE TRIGGER "guard_support_outbound"
    BEFORE INSERT OR UPDATE ON "detention_notices"
    FOR EACH ROW
    EXECUTE FUNCTION "support_outbound_guard"();

DROP TRIGGER IF EXISTS "guard_support_outbound" ON "inbound_replies";

CREATE TRIGGER "guard_support_outbound"
    BEFORE INSERT OR UPDATE ON "inbound_replies"
    FOR EACH ROW
    EXECUTE FUNCTION "support_outbound_guard"();

DROP TRIGGER IF EXISTS "guard_support_outbound" ON "invoice_document_share_tokens";

CREATE TRIGGER "guard_support_outbound"
    BEFORE INSERT OR UPDATE ON "invoice_document_share_tokens"
    FOR EACH ROW
    EXECUTE FUNCTION "support_outbound_guard"();

DROP TRIGGER IF EXISTS "guard_support_outbound" ON "customer_tracking_links";

CREATE TRIGGER "guard_support_outbound"
    BEFORE INSERT OR UPDATE ON "customer_tracking_links"
    FOR EACH ROW
    EXECUTE FUNCTION "support_outbound_guard"();

DROP TRIGGER IF EXISTS "guard_support_outbound" ON "carrier_tracking_links";

CREATE TRIGGER "guard_support_outbound"
    BEFORE INSERT OR UPDATE ON "carrier_tracking_links"
    FOR EACH ROW
    EXECUTE FUNCTION "support_outbound_guard"();

DROP TRIGGER IF EXISTS "guard_support_outbound" ON "tender_offer_tokens";

CREATE TRIGGER "guard_support_outbound"
    BEFORE INSERT OR UPDATE ON "tender_offer_tokens"
    FOR EACH ROW
    EXECUTE FUNCTION "support_outbound_guard"();

DROP TRIGGER IF EXISTS "guard_support_outbound" ON "rate_confirmation_tokens";

CREATE TRIGGER "guard_support_outbound"
    BEFORE INSERT OR UPDATE ON "rate_confirmation_tokens"
    FOR EACH ROW
    EXECUTE FUNCTION "support_outbound_guard"();

DROP TRIGGER IF EXISTS "guard_support_outbound" ON "report_schedules";

CREATE TRIGGER "guard_support_outbound"
    BEFORE INSERT OR UPDATE ON "report_schedules"
    FOR EACH ROW
    EXECUTE FUNCTION "support_outbound_guard"();

DROP TRIGGER IF EXISTS "guard_support_outbound" ON "document_template_versions";

CREATE TRIGGER "guard_support_outbound"
    BEFORE INSERT OR UPDATE ON "document_template_versions"
    FOR EACH ROW
    EXECUTE FUNCTION "support_outbound_guard"();

DROP TRIGGER IF EXISTS "guard_support_outbound" ON "invoices";

CREATE TRIGGER "guard_support_outbound"
    BEFORE UPDATE ON "invoices"
    FOR EACH ROW
    EXECUTE FUNCTION "support_outbound_guard"('memo', 'email_subject_snapshot', 'email_body_snapshot', 'email_to_snapshot', 'email_cc_snapshot', 'email_bcc_snapshot');

DROP TRIGGER IF EXISTS "guard_support_outbound" ON "invoice_lines";

CREATE TRIGGER "guard_support_outbound"
    BEFORE UPDATE ON "invoice_lines"
    FOR EACH ROW
    EXECUTE FUNCTION "support_outbound_guard"('description');

DROP TRIGGER IF EXISTS "guard_support_outbound" ON "billing_controls";

CREATE TRIGGER "guard_support_outbound"
    BEFORE UPDATE ON "billing_controls"
    FOR EACH ROW
    EXECUTE FUNCTION "support_outbound_guard"('default_invoice_footer');

DROP TRIGGER IF EXISTS "guard_support_outbound" ON "customers";

CREATE TRIGGER "guard_support_outbound"
    BEFORE INSERT OR UPDATE ON "customers"
    FOR EACH ROW
    EXECUTE FUNCTION "support_outbound_guard"('status_update_recipients');

DROP TRIGGER IF EXISTS "guard_support_outbound" ON "customer_email_profiles";

CREATE TRIGGER "guard_support_outbound"
    BEFORE INSERT OR UPDATE ON "customer_email_profiles"
    FOR EACH ROW
    EXECUTE FUNCTION "support_outbound_guard"('subject', 'comment', 'to_recipients', 'cc_recipients', 'bcc_recipients');
