ALTER TYPE "sequence_type_enum" ADD VALUE IF NOT EXISTS 'location_code';

--bun:split
ALTER TABLE "sequence_configs"
    ADD COLUMN IF NOT EXISTS "location_code_strategy" jsonb;

--bun:split
DROP INDEX IF EXISTS "idx_locations_search";

--bun:split
ALTER TABLE "locations"
    DROP COLUMN IF EXISTS "search_vector";

--bun:split
ALTER TABLE "locations"
    ALTER COLUMN "code" TYPE varchar(32);

--bun:split
ALTER TABLE "locations"
    ADD COLUMN IF NOT EXISTS search_vector tsvector GENERATED ALWAYS AS (
        setweight(immutable_to_tsvector('simple', COALESCE("code", '')), 'A') ||
        setweight(immutable_to_tsvector('simple', COALESCE("name", '')), 'A') ||
        setweight(immutable_to_tsvector('simple', COALESCE("description", '')), 'B') ||
        setweight(immutable_to_tsvector('simple', COALESCE("address_line_1", '')), 'B') ||
        setweight(immutable_to_tsvector('simple', COALESCE("address_line_2", '')), 'B') ||
        setweight(immutable_to_tsvector('simple', COALESCE("city", '')), 'B') ||
        setweight(immutable_to_tsvector('simple', COALESCE("postal_code"::text, '')), 'C')
    ) STORED;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_locations_search" ON "locations" USING GIN("search_vector");
