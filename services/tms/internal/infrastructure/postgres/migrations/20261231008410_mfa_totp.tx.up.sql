-- Time-based one-time passwords as a second sign-in factor. An authenticator belongs to
-- the person rather than to the organization they enrolled from, so the policy also
-- shows a person their own rows under any organization's scope: sign-in reads them before
-- the organization being entered is settled, and a factor enrolled from one organization
-- must still be asked for when the same person signs into another.
ALTER TABLE "mfa_authenticators"
    ADD COLUMN IF NOT EXISTS "last_used_step" bigint NOT NULL DEFAULT 0;

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_mfa_authenticators_user_totp" ON "mfa_authenticators"("user_id")
WHERE
    "type" = 'totp';

--bun:split
CREATE INDEX IF NOT EXISTS "idx_mfa_authenticators_user" ON "mfa_authenticators"("user_id", "enabled");

--bun:split
CREATE TABLE IF NOT EXISTS "mfa_recovery_codes"(
    "id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "code_hash" varchar(64) NOT NULL,
    "used_at" bigint,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_mfa_recovery_codes" PRIMARY KEY ("id"),
    CONSTRAINT "fk_mfa_recovery_codes_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_mfa_recovery_codes_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uq_mfa_recovery_codes_user_hash" UNIQUE ("user_id", "code_hash")
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_mfa_recovery_codes_user_unused" ON "mfa_recovery_codes"("user_id")
WHERE
    "used_at" IS NULL;

--bun:split
SELECT
    trenova_rls.apply_policy('public.mfa_authenticators', '(' || trenova_rls.standard_expr('organization_id', NULL) || ') OR user_id = (SELECT trenova_rls.user_id())');

--bun:split
SELECT
    trenova_rls.apply_policy('public.mfa_recovery_codes', '(' || trenova_rls.standard_expr('organization_id', NULL) || ') OR user_id = (SELECT trenova_rls.user_id())');

--bun:split
SELECT
    trenova_rls.reconcile();
