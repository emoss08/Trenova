DROP INDEX IF EXISTS "idx_password_reset_tokens_user_created";

--bun:split
DROP INDEX IF EXISTS "uq_password_reset_tokens_token_hash";

--bun:split
DROP TABLE IF EXISTS "password_reset_tokens";
