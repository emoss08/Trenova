--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

DROP INDEX IF EXISTS "idx_password_reset_tokens_user_created";

--bun:split
DROP INDEX IF EXISTS "uq_password_reset_tokens_token_hash";

--bun:split
DROP TABLE IF EXISTS "password_reset_tokens";
