ALTER TABLE "password_reset_tokens"
    DROP CONSTRAINT IF EXISTS "chk_password_reset_tokens_purpose";

ALTER TABLE "password_reset_tokens"
    DROP COLUMN IF EXISTS "purpose";
