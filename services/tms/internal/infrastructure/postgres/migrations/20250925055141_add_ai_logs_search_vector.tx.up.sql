ALTER TABLE "ai_logs"
    ADD COLUMN IF NOT EXISTS search_vector tsvector GENERATED ALWAYS AS (
        setweight(immutable_to_tsvector('english', COALESCE("prompt", '')), 'A') ||
        setweight(immutable_to_tsvector('english', COALESCE(enum_to_text("operation"), '')), 'B') ||
        setweight(immutable_to_tsvector('english', COALESCE("response", '')), 'B') ||
        setweight(immutable_to_tsvector('english', COALESCE(enum_to_text("model"), '')), 'B')
    ) STORED;

--bun:split
CREATE INDEX IF NOT EXISTS idx_ai_logs_search_vector ON "ai_logs" USING GIN(search_vector);

CREATE INDEX IF NOT EXISTS idx_ai_logs_user_id ON "ai_logs"("user_id");
