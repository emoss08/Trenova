-- Trenova Cloud staff support access. An organization's administrator grants Trenova
-- support time-boxed access (support_access_grants); a platform staff member
-- (platform_staff_members) opens a short-lived support session inside that organization
-- (support_sessions) and acts as a support principal (support_principals), a sign-in-less
-- user row in the organization named after the staff member so every write is attributed.
CREATE TABLE IF NOT EXISTS "platform_staff_members"(
    "id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "role" varchar(20) NOT NULL,
    "active" boolean NOT NULL DEFAULT TRUE,
    "added_by" varchar(255) NOT NULL,
    "deactivated_by" varchar(255),
    "deactivated_at" bigint,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_platform_staff_members" PRIMARY KEY ("id"),
    CONSTRAINT "fk_platform_staff_members_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uq_platform_staff_members_user" UNIQUE ("user_id"),
    CONSTRAINT "ck_platform_staff_members_role" CHECK ("role" IN ('support', 'engineer')),
    CONSTRAINT "ck_platform_staff_members_deactivation" CHECK ("active" OR "deactivated_at" IS NOT NULL)
);

--bun:split
CREATE TABLE IF NOT EXISTS "support_access_grants"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "granted_by_id" varchar(100) NOT NULL,
    "access_mode" varchar(20) NOT NULL,
    "note" varchar(500),
    "starts_at" bigint NOT NULL,
    "expires_at" bigint NOT NULL,
    "revoked_at" bigint,
    "revoked_by_id" varchar(100),
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_support_access_grants" PRIMARY KEY ("id"),
    CONSTRAINT "fk_support_access_grants_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_support_access_grants_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_support_access_grants_granted_by" FOREIGN KEY ("granted_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_support_access_grants_revoked_by" FOREIGN KEY ("revoked_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ck_support_access_grants_access_mode" CHECK ("access_mode" IN ('read_only', 'read_write')),
    CONSTRAINT "ck_support_access_grants_window" CHECK ("expires_at" > "starts_at" AND "expires_at" - "starts_at" <= 1209600)
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_support_access_grants_open" ON "support_access_grants"("organization_id", "business_unit_id", "expires_at")
WHERE
    "revoked_at" IS NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_support_access_grants_history" ON "support_access_grants"("organization_id", "business_unit_id", "created_at" DESC);

--bun:split
CREATE TABLE IF NOT EXISTS "support_principals"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "staff_user_id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_support_principals" PRIMARY KEY ("id"),
    CONSTRAINT "fk_support_principals_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_support_principals_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_support_principals_staff_user" FOREIGN KEY ("staff_user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_support_principals_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uq_support_principals_staff" UNIQUE ("organization_id", "staff_user_id"),
    CONSTRAINT "uq_support_principals_user" UNIQUE ("user_id")
);

--bun:split
CREATE TABLE IF NOT EXISTS "support_sessions"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "grant_id" varchar(100) NOT NULL,
    "staff_user_id" varchar(100) NOT NULL,
    "staff_name" varchar(255) NOT NULL,
    "principal_user_id" varchar(100) NOT NULL,
    "base_session_id" varchar(100) NOT NULL,
    "secret_hash" varchar(64) NOT NULL,
    "reason" varchar(500) NOT NULL,
    "ticket_reference" varchar(100),
    "started_at" bigint NOT NULL,
    "expires_at" bigint NOT NULL,
    "elevated_at" bigint,
    "elevated_until" bigint,
    "elevation_reason" varchar(500),
    "elevation_ticket" varchar(100),
    "elevation_count" integer NOT NULL DEFAULT 0,
    "ended_at" bigint,
    "end_reason" varchar(30),
    "last_seen_at" bigint NOT NULL,
    "client_ip" varchar(64),
    "user_agent" varchar(512),
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_support_sessions" PRIMARY KEY ("id"),
    CONSTRAINT "fk_support_sessions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_support_sessions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_support_sessions_grant" FOREIGN KEY ("grant_id") REFERENCES "support_access_grants"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_support_sessions_staff_user" FOREIGN KEY ("staff_user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_support_sessions_principal_user" FOREIGN KEY ("principal_user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_support_sessions_window" CHECK ("expires_at" > "started_at"),
    CONSTRAINT "ck_support_sessions_elevation" CHECK ("elevated_until" IS NULL OR ("elevated_at" IS NOT NULL AND "elevated_until" > "elevated_at")),
    CONSTRAINT "ck_support_sessions_end_reason" CHECK ("end_reason" IS NULL OR "end_reason" IN ('exited', 'expired', 'grant_revoked', 'grant_expired', 'staff_removed', 'signed_out', 'replaced', 'assurance_lost')),
    CONSTRAINT "ck_support_sessions_ended" CHECK (("ended_at" IS NULL) = ("end_reason" IS NULL))
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_support_sessions_org_started" ON "support_sessions"("organization_id", "business_unit_id", "started_at" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_support_sessions_open_staff" ON "support_sessions"("staff_user_id")
WHERE
    "ended_at" IS NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_support_sessions_open_grant" ON "support_sessions"("grant_id")
WHERE
    "ended_at" IS NULL;

--bun:split
SELECT
    trenova_rls.apply_policy('public.platform_staff_members', 'user_id = (SELECT trenova_rls.user_id())', '(SELECT trenova_rls.org_id()) IS NULL');

--bun:split
SELECT
    trenova_rls.reconcile();
