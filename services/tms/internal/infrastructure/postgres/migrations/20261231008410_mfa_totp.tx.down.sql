DROP TABLE IF EXISTS "mfa_recovery_codes";

--bun:split
DROP INDEX IF EXISTS "idx_mfa_authenticators_user";

--bun:split
DROP INDEX IF EXISTS "uq_mfa_authenticators_user_totp";

--bun:split
ALTER TABLE "mfa_authenticators"
    DROP COLUMN IF EXISTS "last_used_step";

--bun:split
SELECT
    trenova_rls.apply_policy('public.mfa_authenticators', trenova_rls.standard_expr('organization_id', NULL));
