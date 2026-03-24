-- Search Vector and Relationship Indexes SQL for AILog
-- Generated for optimal query performance including relationships
-- 1. Search Vector Column
ALTER TABLE "ai_logs"
    ADD COLUMN IF NOT EXISTS search_vector tsvector;

--bun:split
-- 2. Search Vector Index (GIN for full-text search)
CREATE INDEX IF NOT EXISTS idx_ai_logs_search_vector ON "ai_logs" USING GIN(search_vector);

-- 3. Relationship Indexes (only commonly queried relationships)
-- Note: Only add these if you actually filter/join on these relationships
-- Index for User (belongs-to) - ADD ONLY if you JOIN or filter on this
CREATE INDEX IF NOT EXISTS idx_ai_logs_user_id ON "ai_logs"("user_id");

--bun:split
-- 4. Search Vector Trigger Function
CREATE OR REPLACE FUNCTION ai_logs_search_trigger()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.search_vector := setweight(to_tsvector('english', COALESCE(NEW.prompt, '')), 'A') || setweight(to_tsvector('english', COALESCE(CAST(NEW.operation AS text), '')), 'B') || setweight(to_tsvector('english', COALESCE(NEW.response, '')), 'B') || setweight(to_tsvector('english', COALESCE(CAST(NEW.model AS text), '')), 'B');
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
-- 5. Search Vector Trigger
DROP TRIGGER IF EXISTS ai_logs_search_update ON "ai_logs";

CREATE TRIGGER ai_logs_search_update
    BEFORE INSERT OR UPDATE ON "ai_logs"
    FOR EACH ROW
    EXECUTE FUNCTION ai_logs_search_trigger();

--bun:split
-- 6. Update Existing Rows
UPDATE
    "ai_logs"
SET
    search_vector = setweight(to_tsvector('english', COALESCE(prompt, '')), 'A') || setweight(to_tsvector('english', COALESCE(CAST(operation AS text), '')), 'B') || setweight(to_tsvector('english', COALESCE(response, '')), 'B') || setweight(to_tsvector('english', COALESCE(CAST(model AS text), '')), 'B');

