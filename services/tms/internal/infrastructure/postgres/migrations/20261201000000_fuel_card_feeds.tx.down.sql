DROP TABLE IF EXISTS "fuel_card_feed_states";

--bun:split
DROP INDEX IF EXISTS "idx_fuel_cards_unassigned";

--bun:split
ALTER TABLE "fuel_cards"
    DROP COLUMN IF EXISTS "discovered_at";

--bun:split
DROP INDEX IF EXISTS "uq_fuel_purchase_import_batches_feed_reference";

--bun:split
DROP INDEX IF EXISTS "idx_fuel_purchase_import_batches_feed";

--bun:split
ALTER TABLE "fuel_purchase_import_batches"
    DROP CONSTRAINT IF EXISTS "chk_fuel_purchase_import_batches_committed";

--bun:split
ALTER TABLE "fuel_purchase_import_batches"
    ADD CONSTRAINT "chk_fuel_purchase_import_batches_committed" CHECK ("status" <> 'Committed' OR ("committed_at" IS NOT NULL AND "committed_by_id" IS NOT NULL));

--bun:split
ALTER TABLE "fuel_purchase_import_batches"
    DROP CONSTRAINT IF EXISTS "chk_fuel_purchase_import_batches_parsed";

--bun:split
ALTER TABLE "fuel_purchase_import_batches"
    ADD CONSTRAINT "chk_fuel_purchase_import_batches_parsed" CHECK ("status" <> 'Parsed' OR "document_id" IS NOT NULL);

--bun:split
ALTER TABLE "fuel_purchase_import_batches"
    DROP COLUMN IF EXISTS "feed_reference",
    DROP COLUMN IF EXISTS "origin";

--bun:split
DROP TYPE IF EXISTS "fuel_import_origin_enum";
