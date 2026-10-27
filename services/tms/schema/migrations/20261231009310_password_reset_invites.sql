ALTER TABLE "password_reset_tokens"
    ADD COLUMN IF NOT EXISTS "purpose" varchar(20) NOT NULL DEFAULT 'reset';

ALTER TABLE "password_reset_tokens"
    DROP CONSTRAINT IF EXISTS "chk_password_reset_tokens_purpose";

ALTER TABLE "password_reset_tokens"
    ADD CONSTRAINT "chk_password_reset_tokens_purpose" CHECK ("purpose" IN ('reset', 'invite'));

COMMENT ON COLUMN password_reset_tokens.purpose IS 'Why the link went out: reset for a forgotten password, invite for a new account whose owner has not chosen a password yet. Both redeem through the same set-password page; the purpose decides the email wording, the expiry and the authentication event recorded.';
