-- Semantic retrieval storage. The extension is optional: a server without
-- pgvector keeps every table below and runs keyword search only. The vector
-- tables themselves are created by 20261231006110_ai_retrieval_vector, which
-- is also what `trenova db enable-vector` runs once pgvector is installed.
DO $$
BEGIN
    CREATE EXTENSION IF NOT EXISTS vector;
EXCEPTION
    WHEN undefined_file OR insufficient_privilege OR feature_not_supported THEN
        RAISE NOTICE 'pgvector is unavailable, semantic retrieval stays on keyword search';
END $$;

--bun:split
-- One row per source and embedding model: the outbox the indexer drains. A
-- write marks its source Pending; the indexer claims a batch, embeds it and
-- records the outcome against the generation it claimed, so a write that lands
-- while indexing is under way leaves the row Pending instead of Indexed.
CREATE TABLE IF NOT EXISTS "ai_index_entries" (
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "source_type" varchar(30) NOT NULL,
    "source_id" varchar(100) NOT NULL,
    "model_key" varchar(300) NOT NULL,
    "status" varchar(20) NOT NULL DEFAULT 'Pending',
    "generation" bigint NOT NULL DEFAULT 1,
    "attempts" integer NOT NULL DEFAULT 0,
    "chunk_count" integer NOT NULL DEFAULT 0,
    "last_error" text,
    "lease_expires_at" bigint,
    "next_attempt_at" bigint,
    "last_attempt_at" bigint,
    "indexed_at" bigint,
    "created_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT "pk_ai_index_entries" PRIMARY KEY ("organization_id", "business_unit_id", "source_type", "source_id", "model_key"),
    CONSTRAINT "ck_ai_index_entries_source_type" CHECK ("source_type" IN ('Memory', 'Document', 'InboundMessage')),
    CONSTRAINT "ck_ai_index_entries_status" CHECK ("status" IN ('Pending', 'Indexed', 'Failed', 'Skipped')),
    CONSTRAINT "ck_ai_index_entries_counts" CHECK ("generation" >= 1 AND "attempts" >= 0 AND "chunk_count" >= 0),
    CONSTRAINT "ck_ai_index_entries_model_key" CHECK (length(btrim("model_key")) > 0),
    CONSTRAINT "fk_ai_index_entries_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_index_entries_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_ai_index_entries_pending"
    ON "ai_index_entries"("organization_id", "business_unit_id", "model_key", "updated_at")
    WHERE "status" = 'Pending';

--bun:split
CREATE INDEX IF NOT EXISTS "idx_ai_index_entries_model"
    ON "ai_index_entries"("organization_id", "business_unit_id", "model_key", "source_type", "status");

COMMENT ON TABLE "ai_index_entries" IS 'One row per source record and embedding model: whether its embeddings are current, and the outbox the indexer drains';

COMMENT ON COLUMN "ai_index_entries"."source_type" IS 'Memory, Document or InboundMessage';

COMMENT ON COLUMN "ai_index_entries"."source_id" IS 'The id of the agent memory, document or inbound message';

COMMENT ON COLUMN "ai_index_entries"."model_key" IS 'The embedding model the entry is for: provider host, model and dimensions';

COMMENT ON COLUMN "ai_index_entries"."status" IS 'Pending until indexed; Indexed, Failed after the last attempt, or Skipped when there was nothing to embed';

COMMENT ON COLUMN "ai_index_entries"."generation" IS 'Raised each time the source is marked stale; an outcome recorded against an older generation leaves the entry Pending';

COMMENT ON COLUMN "ai_index_entries"."attempts" IS 'How many times the indexer has claimed the entry since it was last marked stale';

COMMENT ON COLUMN "ai_index_entries"."chunk_count" IS 'How many chunks the source was embedded as';

COMMENT ON COLUMN "ai_index_entries"."last_error" IS 'Why the last attempt failed';

COMMENT ON COLUMN "ai_index_entries"."lease_expires_at" IS 'Until when a claimed entry is held by the indexer that claimed it';

COMMENT ON COLUMN "ai_index_entries"."next_attempt_at" IS 'The earliest a failed attempt may be retried';

COMMENT ON COLUMN "ai_index_entries"."last_attempt_at" IS 'When the indexer last claimed the entry';

COMMENT ON COLUMN "ai_index_entries"."indexed_at" IS 'When the source was last indexed or skipped; the sweep compares it with the source''s updated_at';

--bun:split
-- One row per organization says what is indexed and how much indexing may
-- spend. An organization without a row runs on the defaults, which index
-- nothing until an embedding model is active.
CREATE TABLE IF NOT EXISTS "ai_retrieval_settings" (
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "memory_enabled" boolean NOT NULL DEFAULT true,
    "documents_enabled" boolean NOT NULL DEFAULT true,
    "inbound_messages_enabled" boolean NOT NULL DEFAULT true,
    "monthly_indexing_budget_usd" numeric(14, 2) NOT NULL DEFAULT 10.00,
    "paused" boolean NOT NULL DEFAULT false,
    "paused_reason" varchar(20),
    "paused_at" bigint,
    "active_model_key" varchar(300),
    "dimensions" integer,
    "pending_model_key" varchar(300),
    "pending_dimensions" integer,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT "pk_ai_retrieval_settings" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "ck_ai_retrieval_settings_budget" CHECK ("monthly_indexing_budget_usd" >= 0),
    CONSTRAINT "ck_ai_retrieval_settings_paused_reason" CHECK ("paused_reason" IS NULL OR "paused_reason" IN ('Manual', 'Budget')),
    CONSTRAINT "ck_ai_retrieval_settings_paused" CHECK (("paused" AND "paused_reason" IS NOT NULL) OR (NOT "paused" AND "paused_reason" IS NULL)),
    CONSTRAINT "ck_ai_retrieval_settings_active_model" CHECK (("active_model_key" IS NULL) = ("dimensions" IS NULL)),
    CONSTRAINT "ck_ai_retrieval_settings_pending_model" CHECK (("pending_model_key" IS NULL) = ("pending_dimensions" IS NULL)),
    CONSTRAINT "ck_ai_retrieval_settings_dimensions" CHECK (("dimensions" IS NULL OR "dimensions" IN (768, 1024, 1536)) AND ("pending_dimensions" IS NULL OR "pending_dimensions" IN (768, 1024, 1536))),
    CONSTRAINT "ck_ai_retrieval_settings_model_change" CHECK ("pending_model_key" IS NULL OR "active_model_key" IS NULL OR "pending_model_key" <> "active_model_key"),
    CONSTRAINT "fk_ai_retrieval_settings_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_retrieval_settings_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_ai_retrieval_settings_tenant"
    ON "ai_retrieval_settings"("organization_id", "business_unit_id");

COMMENT ON TABLE "ai_retrieval_settings" IS 'What semantic retrieval indexes for an organization, which embedding model it uses and how much indexing may spend';

COMMENT ON COLUMN "ai_retrieval_settings"."memory_enabled" IS 'Whether agent memories are indexed for search by meaning';

COMMENT ON COLUMN "ai_retrieval_settings"."documents_enabled" IS 'Whether document text is indexed for search by meaning';

COMMENT ON COLUMN "ai_retrieval_settings"."inbound_messages_enabled" IS 'Whether inbound email is indexed for search by meaning';

COMMENT ON COLUMN "ai_retrieval_settings"."monthly_indexing_budget_usd" IS 'The most indexing may spend in one calendar month, in US dollars at the provider''s configured price';

COMMENT ON COLUMN "ai_retrieval_settings"."paused" IS 'Whether indexing is stopped; searches keep using what is already indexed';

COMMENT ON COLUMN "ai_retrieval_settings"."paused_reason" IS 'Manual when an administrator paused indexing, Budget when the monthly budget ran out';

COMMENT ON COLUMN "ai_retrieval_settings"."paused_at" IS 'When indexing was paused';

COMMENT ON COLUMN "ai_retrieval_settings"."active_model_key" IS 'The embedding model searches use: provider host, model and dimensions';

COMMENT ON COLUMN "ai_retrieval_settings"."dimensions" IS 'The dimensions of the active model''s embeddings';

COMMENT ON COLUMN "ai_retrieval_settings"."pending_model_key" IS 'The model being indexed beside the active one; it replaces it once every source is indexed';

COMMENT ON COLUMN "ai_retrieval_settings"."pending_dimensions" IS 'The dimensions of the pending model''s embeddings';

--bun:split
-- Deleting a source removes its outbox rows in the same statement. The
-- statement-level trigger reads the deleted rows once, so a bulk delete costs
-- one join rather than one lookup per row.
CREATE OR REPLACE FUNCTION ai_index_entries_purge_source()
RETURNS TRIGGER
AS $$
BEGIN
    DELETE FROM "ai_index_entries" AS e
    USING deleted_rows AS d
    WHERE e."organization_id" = d."organization_id"
      AND e."business_unit_id" = d."business_unit_id"
      AND e."source_type" = TG_ARGV[0]
      AND e."source_id" = d."id";
    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS "trg_agent_memories_ai_index_entries_purge" ON "agent_memories";

CREATE TRIGGER "trg_agent_memories_ai_index_entries_purge"
    AFTER DELETE ON "agent_memories"
    REFERENCING OLD TABLE AS deleted_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION ai_index_entries_purge_source('Memory');

--bun:split
DROP TRIGGER IF EXISTS "trg_documents_ai_index_entries_purge" ON "documents";

CREATE TRIGGER "trg_documents_ai_index_entries_purge"
    AFTER DELETE ON "documents"
    REFERENCING OLD TABLE AS deleted_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION ai_index_entries_purge_source('Document');

--bun:split
DROP TRIGGER IF EXISTS "trg_inbound_messages_ai_index_entries_purge" ON "inbound_messages";

CREATE TRIGGER "trg_inbound_messages_ai_index_entries_purge"
    AFTER DELETE ON "inbound_messages"
    REFERENCING OLD TABLE AS deleted_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION ai_index_entries_purge_source('InboundMessage');
