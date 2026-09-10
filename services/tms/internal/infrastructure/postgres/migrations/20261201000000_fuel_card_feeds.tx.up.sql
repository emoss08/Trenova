--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

ALTER TYPE "integration_type" ADD VALUE IF NOT EXISTS 'WEXFuel';

--bun:split
ALTER TYPE "integration_type" ADD VALUE IF NOT EXISTS 'ComdataFuel';

--bun:split
ALTER TYPE "integration_type" ADD VALUE IF NOT EXISTS 'RampFuel';

--bun:split
ALTER TYPE "integration_category" ADD VALUE IF NOT EXISTS 'FuelCards';

--bun:split
ALTER TYPE "fuel_card_provider_enum" ADD VALUE IF NOT EXISTS 'Ramp';

--bun:split
ALTER TYPE "fuel_import_format_enum" ADD VALUE IF NOT EXISTS 'FixedWidth';

--bun:split
ALTER TYPE "fuel_import_format_enum" ADD VALUE IF NOT EXISTS 'API';

--bun:split
CREATE TYPE "fuel_import_origin_enum" AS ENUM(
    'Upload',
    'Feed'
);

--bun:split
ALTER TABLE "fuel_purchase_import_batches"
    ADD COLUMN IF NOT EXISTS "origin" fuel_import_origin_enum NOT NULL DEFAULT 'Upload',
    ADD COLUMN IF NOT EXISTS "feed_reference" varchar(255);

--bun:split
COMMENT ON COLUMN fuel_purchase_import_batches.origin IS 'Upload batches came from a person choosing a file; Feed batches were opened by a scheduled sync. A feed batch has no uploader and no document, and commits itself.';

--bun:split
COMMENT ON COLUMN fuel_purchase_import_batches.feed_reference IS 'What the sync read: the remote file path for a file feed, or the request window for an API feed. Lets a run be traced back to its source without keeping the payload.';

--bun:split
ALTER TABLE "fuel_purchase_import_batches"
    DROP CONSTRAINT IF EXISTS "chk_fuel_purchase_import_batches_parsed";

--bun:split
ALTER TABLE "fuel_purchase_import_batches"
    ADD CONSTRAINT "chk_fuel_purchase_import_batches_parsed" CHECK ("status" <> 'Parsed' OR "document_id" IS NOT NULL OR "origin" = 'Feed');

--bun:split
ALTER TABLE "fuel_purchase_import_batches"
    DROP CONSTRAINT IF EXISTS "chk_fuel_purchase_import_batches_committed";

--bun:split
ALTER TABLE "fuel_purchase_import_batches"
    ADD CONSTRAINT "chk_fuel_purchase_import_batches_committed" CHECK ("status" <> 'Committed' OR ("committed_at" IS NOT NULL AND ("committed_by_id" IS NOT NULL OR "origin" = 'Feed')));

--bun:split
CREATE INDEX IF NOT EXISTS "idx_fuel_purchase_import_batches_feed" ON "fuel_purchase_import_batches"("organization_id", "business_unit_id", "provider", "created_at" DESC)
WHERE
    "origin" = 'Feed';

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_fuel_purchase_import_batches_feed_reference" ON "fuel_purchase_import_batches"("organization_id", "business_unit_id", "provider", "feed_reference")
WHERE
    "feed_reference" IS NOT NULL;

--bun:split
ALTER TABLE "fuel_cards"
    ADD COLUMN IF NOT EXISTS "discovered_at" bigint;

--bun:split
COMMENT ON COLUMN fuel_cards.discovered_at IS 'When a feed created this card from a transaction nobody had registered a card for. It stays set after assignment so the card''s origin is still legible.';

--bun:split
CREATE INDEX IF NOT EXISTS "idx_fuel_cards_unassigned" ON "fuel_cards"("organization_id", "business_unit_id", "created_at" DESC)
WHERE
    "assigned_tractor_id" IS NULL AND "assigned_worker_id" IS NULL AND "status" <> 'Cancelled';

--bun:split
CREATE TABLE IF NOT EXISTS "fuel_card_feed_states"(
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "provider" fuel_card_provider_enum NOT NULL,
    "feed_type" varchar(32) NOT NULL,
    "cursor" text,
    "last_polled_at" bigint,
    "last_success_at" bigint,
    "failure_count" integer NOT NULL DEFAULT 0,
    "last_error" text,
    CONSTRAINT "pk_fuel_card_feed_states" PRIMARY KEY ("organization_id", "business_unit_id", "provider", "feed_type"),
    CONSTRAINT "fk_fuel_card_feed_states_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_fuel_card_feed_states_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_fuel_card_feed_states_failure_count" CHECK ("failure_count" >= 0)
);

--bun:split
COMMENT ON TABLE fuel_card_feed_states IS 'How far each organization''s fuel card feed has been read. The window is re-requested with an overlap on every run; the transaction reference unique index is what actually prevents a purchase landing twice.';
