-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261115000000_password_reset.tx.up.sql

CREATE TABLE IF NOT EXISTS "password_reset_tokens"(
    "id" TEXT NOT NULL,
    "user_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "token_hash" TEXT NOT NULL,
    "expires_at" INTEGER NOT NULL,
    "used_at" INTEGER,
    "invalidated_at" INTEGER,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_password_reset_tokens" PRIMARY KEY ("id"),
    CONSTRAINT "fk_password_reset_tokens_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_password_reset_tokens_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_password_reset_tokens_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_password_reset_tokens_token_hash" ON "password_reset_tokens" ("token_hash");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_password_reset_tokens_user_created" ON "password_reset_tokens" ("user_id", "created_at");
