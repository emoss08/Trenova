-- A shared server gives every tenant its own business unit, and every row of a tenant lives in
-- that business unit. Internal EDI and organization membership already run inside one business
-- unit in the application; their tables named another organization by id alone, so the database
-- would have kept a row of one tenant that names another tenant's organization, user or business
-- unit. Each such reference now carries the row's own business unit, so one tenant's rows can
-- name only its own organizations and people. A tenant's removal, archive and import therefore
-- never meet a row that belongs to two tenants.

CREATE UNIQUE INDEX IF NOT EXISTS "uq_organizations_id_business_unit"
    ON "organizations"("id", "business_unit_id");

CREATE UNIQUE INDEX IF NOT EXISTS "uq_users_id_business_unit"
    ON "users"("id", "business_unit_id");

ALTER TABLE "edi_connections"
    DROP CONSTRAINT IF EXISTS "fk_edi_connections_source_org",
    DROP CONSTRAINT IF EXISTS "fk_edi_connections_target_org";

ALTER TABLE "edi_connections"
    ADD CONSTRAINT "fk_edi_connections_source_org" FOREIGN KEY ("source_organization_id", "business_unit_id") REFERENCES "organizations"("id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    ADD CONSTRAINT "fk_edi_connections_target_org" FOREIGN KEY ("target_organization_id", "business_unit_id") REFERENCES "organizations"("id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE;

ALTER TABLE "edi_shipment_links"
    DROP CONSTRAINT IF EXISTS "fk_edi_shipment_links_source_org",
    DROP CONSTRAINT IF EXISTS "fk_edi_shipment_links_target_org";

ALTER TABLE "edi_shipment_links"
    ADD CONSTRAINT "fk_edi_shipment_links_source_org" FOREIGN KEY ("source_organization_id", "business_unit_id") REFERENCES "organizations"("id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    ADD CONSTRAINT "fk_edi_shipment_links_target_org" FOREIGN KEY ("target_organization_id", "business_unit_id") REFERENCES "organizations"("id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT;

ALTER TABLE "edi_partners"
    DROP CONSTRAINT IF EXISTS "fk_edi_partners_internal_org";

ALTER TABLE "edi_partners"
    ADD CONSTRAINT "fk_edi_partners_internal_org" FOREIGN KEY ("internal_organization_id", "business_unit_id") REFERENCES "organizations"("id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL ("internal_organization_id");

ALTER TABLE "edi_load_tender_transfers"
    ADD CONSTRAINT "ck_edi_load_tender_transfers_one_business_unit" CHECK ("target_business_unit_id" = "source_business_unit_id");

ALTER TABLE "edi_tender_recipients"
    ADD CONSTRAINT "fk_edi_tender_recipients_recipient_org" FOREIGN KEY ("recipient_organization_id", "business_unit_id") REFERENCES "organizations"("id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL ("recipient_organization_id");

ALTER TABLE "edi_tender_changes"
    ADD CONSTRAINT "fk_edi_tender_changes_source_org" FOREIGN KEY ("source_organization_id", "business_unit_id") REFERENCES "organizations"("id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE;

ALTER TABLE "user_organization_memberships"
    DROP CONSTRAINT IF EXISTS "user_organization_memberships_user_id_fkey",
    DROP CONSTRAINT IF EXISTS "user_organization_memberships_organization_id_fkey";

ALTER TABLE "user_organization_memberships"
    ADD CONSTRAINT "fk_user_organization_memberships_user" FOREIGN KEY ("user_id", "business_unit_id") REFERENCES "users"("id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    ADD CONSTRAINT "fk_user_organization_memberships_organization" FOREIGN KEY ("organization_id", "business_unit_id") REFERENCES "organizations"("id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE;

-- An imported or restored tenant's audit seals and AI audit chain are signed again under this
-- install's chain key when the key that signed them is not configured here, so the audit log's
-- verification passes and goes on catching a changed row. The values they arrived with are kept
-- beside them, so a dispute can still be checked against the install that first signed them.
ALTER TABLE "audit_seals"
    ADD COLUMN IF NOT EXISTS "source_prev_hash" varchar(64),
    ADD COLUMN IF NOT EXISTS "source_hash" varchar(64),
    ADD COLUMN IF NOT EXISTS "source_hash_key_id" varchar(40),
    ADD COLUMN IF NOT EXISTS "source_hash_version" smallint,
    ADD COLUMN IF NOT EXISTS "resealed_at" bigint;

COMMENT ON COLUMN "audit_seals"."source_hash" IS 'The seal''s hash as the install that first signed it wrote it, kept when an imported tenant''s seals are signed again under this install''s key';

ALTER TABLE "ai_audit_events"
    ADD COLUMN IF NOT EXISTS "source_prev_hash" varchar(64),
    ADD COLUMN IF NOT EXISTS "source_hash" varchar(64),
    ADD COLUMN IF NOT EXISTS "source_hash_key_id" varchar(40),
    ADD COLUMN IF NOT EXISTS "source_hash_version" smallint,
    ADD COLUMN IF NOT EXISTS "resealed_at" bigint;

COMMENT ON COLUMN "ai_audit_events"."source_hash" IS 'The row''s hash as the install that first signed it wrote it, kept when an imported tenant''s chain is signed again under this install''s key';

ALTER TABLE "ai_audit_seals"
    ADD COLUMN IF NOT EXISTS "source_head_hash" varchar(64),
    ADD COLUMN IF NOT EXISTS "source_hash_key_id" varchar(40);

-- The chain is rewritten only by a tenant archive load, which only a role that bypasses
-- row-level security can mark; every other update is refused as before.
CREATE OR REPLACE FUNCTION "ai_audit_events_append_only"()
    RETURNS TRIGGER
    AS $$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        IF public.trenova_archive_loading() THEN
            RETURN NEW;
        END IF;
        RAISE EXCEPTION 'ai_audit_events rows cannot be changed'
            USING ERRCODE = 'integrity_constraint_violation';
    END IF;
    IF TG_OP = 'TRUNCATE' THEN
        RAISE EXCEPTION 'ai_audit_events cannot be truncated'
            USING ERRCODE = 'integrity_constraint_violation';
    END IF;
    IF COALESCE(current_setting('trenova.ai_audit_prune', TRUE), '') <> 'on' AND pg_trigger_depth() <= 1 THEN
        RAISE EXCEPTION 'ai_audit_events rows are removed only by the retention sweep'
            USING ERRCODE = 'integrity_constraint_violation';
    END IF;
    RETURN OLD;
END;
$$
LANGUAGE plpgsql;
