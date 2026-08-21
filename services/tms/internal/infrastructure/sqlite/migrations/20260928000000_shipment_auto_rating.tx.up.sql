-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260928000000_shipment_auto_rating.tx.up.sql
ALTER TABLE "shipments"
ADD COLUMN "auto_rated" INTEGER NOT NULL DEFAULT 0;

--bun:split
ALTER TABLE "shipments"
ADD COLUMN "auto_rated_at" INTEGER;

--bun:split
UPDATE "shipments"
SET
    "auto_rated" = 1,
    "auto_rated_at" = "updated_at"
WHERE
    "rate_agreement_id" IS NOT NULL
    AND "rate_override_amount" IS NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_shipments_auto_rated" ON "shipments" ("organization_id", "auto_rated")
WHERE
    "auto_rated";