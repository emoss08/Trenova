-- The vector half of semantic retrieval storage. Everything here needs pgvector
-- 0.8 or newer, so it is created only when that is installed and skipped
-- otherwise; keyword search does not need it. `trenova db enable-vector` runs
-- this same file after installing the extension, so it must stay idempotent
-- and must not be split.
DO $vector$
DECLARE
    installed text;
BEGIN
    SELECT extversion INTO installed FROM pg_extension WHERE extname = 'vector';

    IF installed IS NULL THEN
        RAISE NOTICE 'pgvector is not installed; semantic retrieval stays on keyword search';
        RETURN;
    END IF;

    IF string_to_array(regexp_replace(installed, '[^0-9.].*$', ''), '.')::int[] < ARRAY[0, 8] THEN
        RAISE NOTICE 'pgvector % is older than 0.8; semantic retrieval stays on keyword search', installed;
        RETURN;
    END IF;

    EXECUTE $ddl$
        CREATE TABLE IF NOT EXISTS "ai_embeddings" (
            "organization_id" varchar(100) NOT NULL,
            "business_unit_id" varchar(100) NOT NULL,
            "source_type" varchar(30) NOT NULL,
            "source_id" varchar(100) NOT NULL,
            "chunk_index" integer NOT NULL,
            "model_key" varchar(300) NOT NULL,
            "dimensions" integer NOT NULL,
            "content_hash" varchar(64) NOT NULL,
            "embedding" halfvec NOT NULL,
            "created_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
            "updated_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
            CONSTRAINT "pk_ai_embeddings" PRIMARY KEY ("organization_id", "business_unit_id", "source_type", "source_id", "chunk_index", "model_key"),
            CONSTRAINT "ck_ai_embeddings_source_type" CHECK ("source_type" IN ('Memory', 'Document', 'InboundMessage')),
            CONSTRAINT "ck_ai_embeddings_chunk_index" CHECK ("chunk_index" >= 0),
            CONSTRAINT "ck_ai_embeddings_dimensions" CHECK ("dimensions" IN (768, 1024, 1536)),
            CONSTRAINT "ck_ai_embeddings_embedding_dimensions" CHECK (vector_dims("embedding") = "dimensions"),
            CONSTRAINT "fk_ai_embeddings_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
            CONSTRAINT "fk_ai_embeddings_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
        )
    $ddl$;

    EXECUTE $ddl$
        CREATE INDEX IF NOT EXISTS "idx_ai_embeddings_model"
            ON "ai_embeddings"("organization_id", "business_unit_id", "model_key")
    $ddl$;

    EXECUTE $ddl$
        CREATE INDEX IF NOT EXISTS "idx_ai_embeddings_hnsw_768"
            ON "ai_embeddings" USING hnsw (("embedding"::halfvec(768)) halfvec_cosine_ops)
            WHERE "dimensions" = 768
    $ddl$;

    EXECUTE $ddl$
        CREATE INDEX IF NOT EXISTS "idx_ai_embeddings_hnsw_1024"
            ON "ai_embeddings" USING hnsw (("embedding"::halfvec(1024)) halfvec_cosine_ops)
            WHERE "dimensions" = 1024
    $ddl$;

    EXECUTE $ddl$
        CREATE INDEX IF NOT EXISTS "idx_ai_embeddings_hnsw_1536"
            ON "ai_embeddings" USING hnsw (("embedding"::halfvec(1536)) halfvec_cosine_ops)
            WHERE "dimensions" = 1536
    $ddl$;

    EXECUTE $ddl$
        COMMENT ON TABLE "ai_embeddings" IS 'Embeddings of agent memories, documents and inbound email, one row per chunk and embedding model'
    $ddl$;

    EXECUTE $ddl$
        COMMENT ON COLUMN "ai_embeddings"."chunk_index" IS 'The chunk''s place in its source, from zero'
    $ddl$;

    EXECUTE $ddl$
        COMMENT ON COLUMN "ai_embeddings"."model_key" IS 'The embedding model that produced the row: provider host, model and dimensions'
    $ddl$;

    EXECUTE $ddl$
        COMMENT ON COLUMN "ai_embeddings"."content_hash" IS 'SHA-256 of the chunk text and chunker version; an unchanged hash is never embedded again'
    $ddl$;

    EXECUTE $ddl$
        COMMENT ON COLUMN "ai_embeddings"."embedding" IS 'The chunk embedding at half precision; each allowed dimension has its own partial HNSW index'
    $ddl$;

    EXECUTE $ddl$
        CREATE TABLE IF NOT EXISTS "ai_catalog_embeddings" (
            "model_key" varchar(300) NOT NULL,
            "corpus" varchar(30) NOT NULL,
            "item_key" varchar(200) NOT NULL,
            "content_hash" varchar(64) NOT NULL,
            "dimensions" integer NOT NULL,
            "embedding" halfvec NOT NULL,
            "created_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
            CONSTRAINT "pk_ai_catalog_embeddings" PRIMARY KEY ("model_key", "corpus", "item_key", "content_hash"),
            CONSTRAINT "ck_ai_catalog_embeddings_corpus" CHECK ("corpus" IN ('Tools', 'ProductGuide')),
            CONSTRAINT "ck_ai_catalog_embeddings_dimensions" CHECK ("dimensions" IN (768, 1024, 1536)),
            CONSTRAINT "ck_ai_catalog_embeddings_embedding_dimensions" CHECK (vector_dims("embedding") = "dimensions")
        )
    $ddl$;

    EXECUTE $ddl$
        COMMENT ON TABLE "ai_catalog_embeddings" IS 'Embeddings of the tool catalog and product guide, which are the same for every organization, per embedding model and content hash'
    $ddl$;

    EXECUTE $ddl$
        CREATE OR REPLACE FUNCTION ai_embeddings_purge_source()
        RETURNS TRIGGER
        AS $fn$
        BEGIN
            DELETE FROM "ai_embeddings" AS e
            USING deleted_rows AS d
            WHERE e."organization_id" = d."organization_id"
              AND e."business_unit_id" = d."business_unit_id"
              AND e."source_type" = TG_ARGV[0]
              AND e."source_id" = d."id";
            RETURN NULL;
        END;
        $fn$
        LANGUAGE plpgsql
    $ddl$;

    EXECUTE $ddl$
        DROP TRIGGER IF EXISTS "trg_agent_memories_ai_embeddings_purge" ON "agent_memories"
    $ddl$;

    EXECUTE $ddl$
        CREATE TRIGGER "trg_agent_memories_ai_embeddings_purge"
            AFTER DELETE ON "agent_memories"
            REFERENCING OLD TABLE AS deleted_rows
            FOR EACH STATEMENT
            EXECUTE FUNCTION ai_embeddings_purge_source('Memory')
    $ddl$;

    EXECUTE $ddl$
        DROP TRIGGER IF EXISTS "trg_documents_ai_embeddings_purge" ON "documents"
    $ddl$;

    EXECUTE $ddl$
        CREATE TRIGGER "trg_documents_ai_embeddings_purge"
            AFTER DELETE ON "documents"
            REFERENCING OLD TABLE AS deleted_rows
            FOR EACH STATEMENT
            EXECUTE FUNCTION ai_embeddings_purge_source('Document')
    $ddl$;

    EXECUTE $ddl$
        DROP TRIGGER IF EXISTS "trg_inbound_messages_ai_embeddings_purge" ON "inbound_messages"
    $ddl$;

    EXECUTE $ddl$
        CREATE TRIGGER "trg_inbound_messages_ai_embeddings_purge"
            AFTER DELETE ON "inbound_messages"
            REFERENCING OLD TABLE AS deleted_rows
            FOR EACH STATEMENT
            EXECUTE FUNCTION ai_embeddings_purge_source('InboundMessage')
    $ddl$;
END
$vector$;
