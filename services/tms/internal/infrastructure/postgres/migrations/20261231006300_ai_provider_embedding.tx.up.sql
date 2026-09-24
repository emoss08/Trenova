-- An embedding provider turns text into vectors for semantic retrieval. The
-- dimension is part of what a stored vector means, so it is fixed per
-- provider and limited to the sizes the vector indexes are built for. The
-- input style says how the endpoint wants documents and queries told apart:
-- Voyage takes an input_type field, nomic-embed-text a text prefix.
ALTER TABLE "ai_providers"
    ADD COLUMN IF NOT EXISTS "embedding_dimensions" integer;

--bun:split
ALTER TABLE "ai_providers"
    ADD COLUMN IF NOT EXISTS "embedding_input_style" varchar(50) NOT NULL DEFAULT 'None';

--bun:split
ALTER TABLE "ai_providers"
    ADD CONSTRAINT "ck_ai_providers_embedding_dimensions" CHECK ("embedding_dimensions" IS NULL OR "embedding_dimensions" IN (768, 1024, 1536));

--bun:split
ALTER TABLE "ai_providers"
    ADD CONSTRAINT "ck_ai_providers_embedding_input_style" CHECK ("embedding_input_style" IN ('None', 'VoyageInputType', 'NomicPrefix'));

--bun:split
-- The Anthropic Messages protocol has no embedding endpoint, and a vector of
-- unknown size cannot be stored, so neither can be routed the Embedding task.
ALTER TABLE "ai_providers"
    ADD CONSTRAINT "ck_ai_providers_embedding_task" CHECK (NOT ('Embedding' = ANY (COALESCE("tasks", '{}'::text[]))) OR ("kind" <> 'AnthropicMessages' AND "embedding_dimensions" IS NOT NULL));

--bun:split
COMMENT ON COLUMN "ai_providers"."embedding_dimensions" IS 'Vector size an embedding provider returns: 768, 1024 or 1536; null for a provider that serves no embeddings';

COMMENT ON COLUMN "ai_providers"."embedding_input_style" IS 'How documents and queries are told apart on the wire: None, VoyageInputType (input_type field) or NomicPrefix (search_document/search_query prefixes)';
