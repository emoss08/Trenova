DROP INDEX IF EXISTS "idx_fuel_cards_search";

--bun:split
ALTER TABLE "fuel_cards"
    DROP COLUMN IF EXISTS "search_vector";

--bun:split
ALTER TABLE "fuel_cards"
    ADD COLUMN IF NOT EXISTS search_vector tsvector GENERATED ALWAYS AS (
        setweight(immutable_to_tsvector('simple', COALESCE("label", '')), 'A') ||
        setweight(immutable_to_tsvector('simple', COALESCE("last_four", '')), 'B') ||
        setweight(immutable_to_tsvector('simple', COALESCE("external_card_id", '')), 'B')
    ) STORED;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_fuel_cards_search" ON "fuel_cards" USING GIN("search_vector");

--bun:split
SELECT trenova_rls.reconcile();
