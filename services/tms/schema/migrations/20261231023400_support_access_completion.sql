-- Support access, completed. A grant now says whether staff may use the assistant (every model
-- call is paid for by the organization's own AI providers, so it is off unless an administrator
-- turns it on), and an administrator may extend a live grant: its new end is counted from the
-- extension, never more than 72 hours ahead.
ALTER TABLE "support_access_grants"
    ADD COLUMN IF NOT EXISTS "allows_assistant" boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS "extended_at" bigint,
    ADD COLUMN IF NOT EXISTS "extended_by_id" varchar(100);

ALTER TABLE "support_access_grants"
    DROP CONSTRAINT IF EXISTS "fk_support_access_grants_extended_by";

ALTER TABLE "support_access_grants"
    ADD CONSTRAINT "fk_support_access_grants_extended_by" FOREIGN KEY ("extended_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE NO ACTION;

ALTER TABLE "support_access_grants"
    DROP CONSTRAINT IF EXISTS "ck_support_access_grants_expiry";

ALTER TABLE "support_access_grants"
    ADD CONSTRAINT "ck_support_access_grants_expiry" CHECK (
        "expires_at" > "created_at"
        AND "expires_at" <= COALESCE("extended_at", "created_at") + 72 * 3600
    );

ALTER TABLE "support_access_grants"
    DROP CONSTRAINT IF EXISTS "ck_support_access_grants_extension";

ALTER TABLE "support_access_grants"
    ADD CONSTRAINT "ck_support_access_grants_extension" CHECK (
        ("extended_at" IS NULL AND "extended_by_id" IS NULL)
        OR ("extended_at" IS NOT NULL AND "extended_by_id" IS NOT NULL AND "extended_at" >= "created_at")
    );

COMMENT ON COLUMN "support_access_grants"."allows_assistant" IS 'Whether staff under this grant may use the assistant, whose model calls the organization pays for';

COMMENT ON COLUMN "support_access_grants"."extended_at" IS 'When an administrator last extended the grant; its end is counted from then';

-- Where money is sent is never Trenova support's to change, whatever the grant's scope and
-- whichever path writes the row: a form, an import, an agent tool or a workflow run for the
-- session. The guard reads the transaction's signed tenant scope and refuses only a support
-- principal (a user id starting sup_). The columns it guards are the trigger's arguments, and
-- supportaccess.PaymentDestinations in the TMS names the same ones.
CREATE OR REPLACE FUNCTION "support_payment_destination_guard"()
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

    after_row := to_jsonb(NEW);
    IF TG_OP = 'UPDATE' THEN
        before_row := to_jsonb(OLD);
    END IF;

    FOREACH guarded IN ARRAY TG_ARGV LOOP
        IF (TG_OP = 'UPDATE' AND (after_row -> guarded) IS DISTINCT FROM (before_row -> guarded))
            OR (TG_OP = 'INSERT' AND COALESCE(after_row ->> guarded, '') <> '') THEN
            RAISE EXCEPTION 'Trenova support cannot change where money is sent (%.%)', TG_TABLE_NAME, guarded
                USING ERRCODE = 'TRS01';
        END IF;
    END LOOP;

    RETURN NEW;
END
$$;

COMMENT ON FUNCTION "support_payment_destination_guard"() IS 'Refuses a Trenova support principal any write to the payment destination columns named as the trigger''s arguments';

DROP TRIGGER IF EXISTS "guard_support_payment_destination" ON "invoices";

CREATE TRIGGER "guard_support_payment_destination"
    BEFORE UPDATE ON "invoices"
    FOR EACH ROW
    EXECUTE FUNCTION "support_payment_destination_guard"('remittance_instructions', 'factoring_override');

DROP TRIGGER IF EXISTS "guard_support_payment_destination" ON "billing_controls";

CREATE TRIGGER "guard_support_payment_destination"
    BEFORE UPDATE ON "billing_controls"
    FOR EACH ROW
    EXECUTE FUNCTION "support_payment_destination_guard"('default_invoice_terms');

DROP TRIGGER IF EXISTS "guard_support_payment_destination" ON "organizations";

CREATE TRIGGER "guard_support_payment_destination"
    BEFORE UPDATE ON "organizations"
    FOR EACH ROW
    EXECUTE FUNCTION "support_payment_destination_guard"('address_line1', 'address_line2', 'city', 'state_id', 'postal_code');

DROP TRIGGER IF EXISTS "guard_support_payment_destination" ON "carriers";

CREATE TRIGGER "guard_support_payment_destination"
    BEFORE INSERT OR UPDATE ON "carriers"
    FOR EACH ROW
    EXECUTE FUNCTION "support_payment_destination_guard"('remit_to_name', 'remit_address_line_1', 'remit_address_line_2', 'remit_city', 'remit_state_id', 'remit_postal_code');

DROP TRIGGER IF EXISTS "guard_support_payment_destination" ON "customer_billing_profiles";

CREATE TRIGGER "guard_support_payment_destination"
    BEFORE INSERT OR UPDATE ON "customer_billing_profiles"
    FOR EACH ROW
    EXECUTE FUNCTION "support_payment_destination_guard"('factoring_company_id');

DROP TRIGGER IF EXISTS "guard_support_payment_destination" ON "factoring_companies";

CREATE TRIGGER "guard_support_payment_destination"
    BEFORE INSERT OR UPDATE ON "factoring_companies"
    FOR EACH ROW
    EXECUTE FUNCTION "support_payment_destination_guard"('address_line_1', 'address_line_2', 'city', 'state_id', 'postal_code', 'remittance_instructions', 'notice_of_assignment');
