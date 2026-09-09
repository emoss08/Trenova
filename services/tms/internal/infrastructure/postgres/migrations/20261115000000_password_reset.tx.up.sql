--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TABLE IF NOT EXISTS "password_reset_tokens"(
    "id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "token_hash" varchar(64) NOT NULL,
    "expires_at" bigint NOT NULL,
    "used_at" bigint,
    "invalidated_at" bigint,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_password_reset_tokens" PRIMARY KEY ("id"),
    CONSTRAINT "fk_password_reset_tokens_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_password_reset_tokens_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_password_reset_tokens_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_password_reset_tokens_token_hash" ON "password_reset_tokens"("token_hash");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_password_reset_tokens_user_created" ON "password_reset_tokens"("user_id", "created_at");

--bun:split
COMMENT ON TABLE password_reset_tokens IS 'Outstanding password reset links. The row is the only record of a reset request: the account itself is untouched until a token is redeemed, so requesting a reset for somebody else''s address cannot lock them out.';

--bun:split
COMMENT ON COLUMN password_reset_tokens.token_hash IS 'SHA-256 hex digest of the token that went out in the email. The token itself is never stored, so a dump of this table yields nothing redeemable.';

--bun:split
COMMENT ON COLUMN password_reset_tokens.used_at IS 'When the token was redeemed. Set once; a second redemption of the same link is refused.';

--bun:split
COMMENT ON COLUMN password_reset_tokens.invalidated_at IS 'When the token was retired without being used - a newer request superseded it, or the password changed by another route. Distinct from used_at so a redemption attempt can be told apart from a stale link in the audit trail.';
