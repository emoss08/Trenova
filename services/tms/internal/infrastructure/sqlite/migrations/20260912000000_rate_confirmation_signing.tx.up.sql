-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260912000000_rate_confirmation_signing.tx.up.sql

ALTER TABLE "rate_confirmations" ADD COLUMN "generated_via" TEXT NOT NULL DEFAULT 'Dispatcher';

--bun:split

ALTER TABLE "rate_confirmations" ADD COLUMN "source_tender_offer_id" TEXT;

--bun:split

ALTER TABLE "rate_confirmations" ADD COLUMN "confirmed_via" TEXT;

--bun:split

ALTER TABLE "rate_confirmations" ADD COLUMN "confirmed_by_title" TEXT;

--bun:split

CREATE TABLE IF NOT EXISTS "rate_confirmation_tokens"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "rate_confirmation_id" TEXT NOT NULL,
    "token_hash" TEXT NOT NULL,
    "email" TEXT NOT NULL,
    "expires_at" INTEGER NOT NULL,
    "used_at" INTEGER,
    "revoked_at" INTEGER,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_rate_confirmation_tokens" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_rate_confirmation_tokens_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_confirmation_tokens_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_confirmation_tokens_rate_confirmation" FOREIGN KEY ("rate_confirmation_id", "business_unit_id", "organization_id") REFERENCES "rate_confirmations"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_confirmation_tokens_hash" ON "rate_confirmation_tokens" ("token_hash");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_rate_confirmation_tokens_rate_confirmation" ON "rate_confirmation_tokens" ("rate_confirmation_id", "organization_id");
