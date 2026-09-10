-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261201000000_fuel_card_feeds.tx.up.sql

ALTER TABLE "fuel_purchase_import_batches" ADD COLUMN "origin" TEXT NOT NULL DEFAULT 'Upload';

--bun:split

ALTER TABLE "fuel_purchase_import_batches" ADD COLUMN "feed_reference" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_fuel_purchase_import_batches_feed" ON "fuel_purchase_import_batches" ("organization_id", "business_unit_id", "provider", "created_at" DESC)WHERE
    "origin" = 'Feed';

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_fuel_purchase_import_batches_feed_reference" ON "fuel_purchase_import_batches" ("organization_id", "business_unit_id", "provider", "feed_reference")WHERE
    "feed_reference" IS NOT NULL;

--bun:split

ALTER TABLE "fuel_cards" ADD COLUMN "discovered_at" INTEGER;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_fuel_cards_unassigned" ON "fuel_cards" ("organization_id", "business_unit_id", "created_at" DESC)WHERE
    "assigned_tractor_id" IS NULL AND "assigned_worker_id" IS NULL AND "status" <> 'Cancelled';

--bun:split

CREATE TABLE IF NOT EXISTS "fuel_card_feed_states"(
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "provider" TEXT NOT NULL,
    "feed_type" TEXT NOT NULL,
    "cursor" TEXT,
    "last_polled_at" INTEGER,
    "last_success_at" INTEGER,
    "failure_count" INTEGER NOT NULL DEFAULT 0,
    "last_error" TEXT,
    CONSTRAINT "pk_fuel_card_feed_states" PRIMARY KEY ("organization_id", "business_unit_id", "provider", "feed_type"),
    CONSTRAINT "fk_fuel_card_feed_states_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_fuel_card_feed_states_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_fuel_card_feed_states_failure_count" CHECK ("failure_count" >= 0)
);
