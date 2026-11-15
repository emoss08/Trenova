-- Trenova support access on a managed instance. An administrator grants one named Trenova staff
-- member access to their organization for a fixed time; staff enter only with a single-use
-- assertion the control plane signs, and only while a grant for exactly that email and
-- organization is live. Everything done under a grant is written to the audit log with the
-- support principal (principal_type 'support', principal_id the grant). Self-hosted installs
-- never read or write these tables.

CREATE TABLE IF NOT EXISTS "support_access_grants"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "staff_email" varchar(254) NOT NULL,
    "reason" text NOT NULL,
    "scope" varchar(20) NOT NULL,
    "duration_hours" integer NOT NULL,
    "principal_user_id" varchar(100) NOT NULL,
    "granted_by_id" varchar(100) NOT NULL,
    "expires_at" bigint NOT NULL,
    "revoked_at" bigint,
    "revoked_by_id" varchar(100),
    "revoke_reason" text,
    "expiry_recorded_at" bigint,
    "last_entered_at" bigint,
    "last_staff_name" varchar(255),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL,
    "updated_at" bigint NOT NULL,
    CONSTRAINT "pk_support_access_grants" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_support_access_grants_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_support_access_grants_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_support_access_grants_principal_user" FOREIGN KEY ("principal_user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT "fk_support_access_grants_granted_by" FOREIGN KEY ("granted_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT "fk_support_access_grants_revoked_by" FOREIGN KEY ("revoked_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT "ck_support_access_grants_scope" CHECK ("scope" IN ('ReadOnly', 'Full')),
    CONSTRAINT "ck_support_access_grants_duration" CHECK ("duration_hours" IN (1, 4, 24, 72)),
    CONSTRAINT "ck_support_access_grants_expiry" CHECK ("expires_at" > "created_at" AND "expires_at" <= "created_at" + 72 * 3600),
    CONSTRAINT "ck_support_access_grants_staff_email" CHECK (
        "staff_email" = lower("staff_email")
        AND "staff_email" ~ '^[a-z0-9._+-]{1,64}@[a-z0-9-]+(\.[a-z0-9-]+)+$'
    ),
    CONSTRAINT "ck_support_access_grants_reason" CHECK (length(btrim("reason")) BETWEEN 1 AND 500),
    CONSTRAINT "ck_support_access_grants_principal" CHECK ("principal_user_id" LIKE 'sup\_%'),
    CONSTRAINT "ck_support_access_grants_revocation" CHECK (
        ("revoked_at" IS NULL AND "revoked_by_id" IS NULL AND "revoke_reason" IS NULL)
        OR ("revoked_at" IS NOT NULL AND "revoked_by_id" IS NOT NULL AND length(btrim("revoke_reason")) BETWEEN 1 AND 500)
    )
);

COMMENT ON TABLE "support_access_grants" IS 'Time-limited access to one organization an administrator gave one named Trenova staff member on a managed instance';

COMMENT ON COLUMN "support_access_grants"."staff_email" IS 'The staff member''s work email, lower-cased, at the Trenova staff domain the control plane configured';

COMMENT ON COLUMN "support_access_grants"."principal_user_id" IS 'The system-owned, inactive support principal every action under this grant is attributed to; never a person of the organization';

COMMENT ON COLUMN "support_access_grants"."expiry_recorded_at" IS 'When the expiry sweep wrote the security entry saying the grant ended on its own';

CREATE UNIQUE INDEX IF NOT EXISTS "uq_support_access_grants_principal" ON "support_access_grants"("principal_user_id");

CREATE INDEX IF NOT EXISTS "idx_support_access_grants_live" ON "support_access_grants"("organization_id", "staff_email", "expires_at")
WHERE "revoked_at" IS NULL;

CREATE INDEX IF NOT EXISTS "idx_support_access_grants_tenant_created" ON "support_access_grants"("organization_id", "business_unit_id", "created_at" DESC, "id" DESC);

CREATE INDEX IF NOT EXISTS "idx_support_access_grants_unrecorded_expiry" ON "support_access_grants"("expires_at")
WHERE "revoked_at" IS NULL AND "expiry_recorded_at" IS NULL;

-- The control plane's support-signing public keys, delivered by the operator with this instance's
-- id and the Trenova staff email domain. One row, replaced on every delivery.
CREATE TABLE IF NOT EXISTS "managed_support_access_states"(
    "id" varchar(20) NOT NULL,
    "instance_id" varchar(100) NOT NULL,
    "staff_domain" varchar(253) NOT NULL,
    "keys" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "received_at" bigint NOT NULL,
    CONSTRAINT "pk_managed_support_access_states" PRIMARY KEY ("id"),
    CONSTRAINT "ck_managed_support_access_states_singleton" CHECK ("id" = 'current'),
    CONSTRAINT "ck_managed_support_access_states_keys" CHECK (jsonb_typeof("keys") = 'array'),
    CONSTRAINT "ck_managed_support_access_states_domain" CHECK ("staff_domain" = lower("staff_domain") AND "staff_domain" ~ '^[a-z0-9-]+(\.[a-z0-9-]+)+$')
);

COMMENT ON TABLE "managed_support_access_states" IS 'The keys that verify the control plane''s support assertions, this instance''s id and the staff domain; no tenant data';

INSERT INTO trenova_rls.global_tables("table_name", "reason")
VALUES ('managed_support_access_states', 'The instance-wide support assertion keys the control plane sets for a managed instance; no tenant data')
ON CONFLICT ("table_name") DO UPDATE SET "reason" = EXCLUDED."reason";

-- Every support assertion the instance accepted, kept until it could no longer be valid, so a
-- replayed assertion is refused. The primary key decides two uses of one assertion at once.
CREATE TABLE IF NOT EXISTS "support_assertion_uses"(
    "assertion_id" varchar(64) NOT NULL,
    "used_at" bigint NOT NULL,
    "expires_at" bigint NOT NULL,
    CONSTRAINT "pk_support_assertion_uses" PRIMARY KEY ("assertion_id"),
    CONSTRAINT "ck_support_assertion_uses_id" CHECK ("assertion_id" ~ '^[A-Za-z0-9_-]{16,64}$')
);

CREATE INDEX IF NOT EXISTS "idx_support_assertion_uses_expires" ON "support_assertion_uses"("expires_at");

COMMENT ON TABLE "support_assertion_uses" IS 'Support assertion ids already used to enter this instance, kept past their expiry so none is accepted twice';

INSERT INTO trenova_rls.global_tables("table_name", "reason")
VALUES ('support_assertion_uses', 'Ids of control plane support assertions already used; checked before any tenant is known and holds no tenant data')
ON CONFLICT ("table_name") DO UPDATE SET "reason" = EXCLUDED."reason";

-- Audit entries written in a support session credit the support principal: the grant is the
-- principal, the grant's system-owned support user is the user.
ALTER TABLE "audit_entries"
    DROP CONSTRAINT IF EXISTS "ck_audit_entries_principal_type";

ALTER TABLE "audit_entries"
    ADD CONSTRAINT "ck_audit_entries_principal_type" CHECK ("principal_type" IN ('session_user', 'api_key', 'system', 'agent', 'support'));

ALTER TABLE "audit_entries"
    DROP CONSTRAINT IF EXISTS "ck_audit_entries_principal_consistency";

ALTER TABLE "audit_entries"
    ADD CONSTRAINT "ck_audit_entries_principal_consistency" CHECK (
        ("principal_type" = 'session_user' AND "user_id" IS NOT NULL AND "api_key_id" IS NULL AND "principal_id" = "user_id")
        OR
        ("principal_type" = 'api_key' AND "user_id" IS NULL AND "api_key_id" IS NOT NULL AND "principal_id" = "api_key_id")
        OR
        ("principal_type" = 'system' AND "user_id" IS NULL AND "api_key_id" IS NULL)
        OR
        ("principal_type" = 'agent' AND "api_key_id" IS NULL AND ("user_id" IS NULL OR "principal_id" <> "user_id"))
        OR
        ("principal_type" = 'support' AND "user_id" IS NOT NULL AND "api_key_id" IS NULL AND "principal_id" <> "user_id")
    );

-- A support principal stands for Trenova support and must never read as anyone else or sign in.
-- Whatever writes the users table, a support principal's identity and standing cannot change.
CREATE OR REPLACE FUNCTION "users_support_principal_guard"()
    RETURNS TRIGGER
    LANGUAGE plpgsql
    SET search_path = pg_catalog, pg_temp
AS $$
BEGIN
    IF OLD."id" LIKE 'sup\_%' AND (
        NEW."id" IS DISTINCT FROM OLD."id"
        OR NEW."name" IS DISTINCT FROM OLD."name"
        OR NEW."username" IS DISTINCT FROM OLD."username"
        OR NEW."email_address" IS DISTINCT FROM OLD."email_address"
        OR NEW."password" IS DISTINCT FROM OLD."password"
        OR NEW."status" IS DISTINCT FROM OLD."status"
        OR NEW."is_locked" IS DISTINCT FROM OLD."is_locked"
        OR NEW."current_organization_id" IS DISTINCT FROM OLD."current_organization_id"
        OR NEW."business_unit_id" IS DISTINCT FROM OLD."business_unit_id"
        OR NEW."profile_pic_url" IS DISTINCT FROM OLD."profile_pic_url"
        OR NEW."thumbnail_url" IS DISTINCT FROM OLD."thumbnail_url"
    ) THEN
        RAISE EXCEPTION 'a Trenova support principal cannot be changed'
            USING ERRCODE = '42501';
    END IF;
    RETURN NEW;
END
$$;

COMMENT ON FUNCTION "users_support_principal_guard"() IS 'Keeps a Trenova support principal inactive and named as Trenova support, whoever writes the row';

DROP TRIGGER IF EXISTS "guard_users_support_principal" ON "users";

CREATE TRIGGER "guard_users_support_principal"
    BEFORE UPDATE ON "users"
    FOR EACH ROW
    EXECUTE FUNCTION "users_support_principal_guard"();
