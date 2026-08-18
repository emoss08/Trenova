-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260922000000_carrier_assignment_rating.tx.up.sql

ALTER TABLE "carrier_assignments" ADD COLUMN "rate_quote_id" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_carrier_assignments_rate_quote" ON "carrier_assignments" ("organization_id", "business_unit_id", "rate_quote_id")WHERE
    "rate_quote_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_rate_quotes_margin_reporting" ON "rate_quotes" ("organization_id", "business_unit_id", "party_type", "party_id", "rated_at")WHERE
    "margin_percent" IS NOT NULL;
