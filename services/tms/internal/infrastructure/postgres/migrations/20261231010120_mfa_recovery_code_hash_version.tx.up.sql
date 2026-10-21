-- Recovery codes issued from here on are HMAC-SHA256 digests keyed by
-- security.mfa.recoveryCodePepper (version 2). Rows already stored are unsalted
-- SHA-256 digests (version 1) and keep working until their owner replaces them.
ALTER TABLE "mfa_recovery_codes"
    ADD COLUMN IF NOT EXISTS "hash_version" smallint NOT NULL DEFAULT 1;

--bun:split
ALTER TABLE "mfa_recovery_codes"
    ALTER COLUMN "hash_version" SET DEFAULT 2;
