DROP INDEX IF EXISTS "idx_fuel_cards_search";

ALTER TABLE "fuel_cards"
    DROP COLUMN IF EXISTS "search_vector";

ALTER TABLE "fuel_cards"
    ADD COLUMN IF NOT EXISTS search_vector tsvector GENERATED ALWAYS AS (
        setweight(immutable_to_tsvector('simple', COALESCE("label", '')), 'A') ||
        setweight(immutable_to_tsvector('simple', COALESCE("last_four", '')), 'B')
    ) STORED;

SELECT trenova_rls.reconcile();
