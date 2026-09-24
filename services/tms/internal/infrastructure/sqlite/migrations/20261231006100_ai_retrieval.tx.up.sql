-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231006100_ai_retrieval.tx.up.sql

CREATE TABLE IF NOT EXISTS "ai_index_entries" (
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "source_type" TEXT NOT NULL,
    "source_id" TEXT NOT NULL,
    "model_key" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Pending',
    "generation" INTEGER NOT NULL DEFAULT 1,
    "attempts" INTEGER NOT NULL DEFAULT 0,
    "chunk_count" INTEGER NOT NULL DEFAULT 0,
    "last_error" TEXT,
    "lease_expires_at" INTEGER,
    "next_attempt_at" INTEGER,
    "last_attempt_at" INTEGER,
    "indexed_at" INTEGER,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_ai_index_entries" PRIMARY KEY ("organization_id", "business_unit_id", "source_type", "source_id", "model_key"),
    CONSTRAINT "ck_ai_index_entries_source_type" CHECK ("source_type" IN ('Memory', 'Document', 'InboundMessage')),
    CONSTRAINT "ck_ai_index_entries_status" CHECK ("status" IN ('Pending', 'Indexed', 'Failed', 'Skipped')),
    CONSTRAINT "ck_ai_index_entries_counts" CHECK ("generation" >= 1 AND "attempts" >= 0 AND "chunk_count" >= 0),
    CONSTRAINT "fk_ai_index_entries_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_index_entries_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_index_entries_pending"
    ON "ai_index_entries" ("organization_id", "business_unit_id", "model_key", "updated_at")WHERE "status" = 'Pending';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_index_entries_model"
    ON "ai_index_entries" ("organization_id", "business_unit_id", "model_key", "source_type", "status");

--bun:split

CREATE TABLE IF NOT EXISTS "ai_retrieval_settings" (
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "memory_enabled" INTEGER NOT NULL DEFAULT 1,
    "documents_enabled" INTEGER NOT NULL DEFAULT 1,
    "inbound_messages_enabled" INTEGER NOT NULL DEFAULT 1,
    "monthly_indexing_budget_usd" REAL NOT NULL DEFAULT 10.00,
    "paused" INTEGER NOT NULL DEFAULT 0,
    "paused_reason" TEXT,
    "paused_at" INTEGER,
    "active_model_key" TEXT,
    "dimensions" INTEGER,
    "pending_model_key" TEXT,
    "pending_dimensions" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
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
    ON "ai_retrieval_settings" ("organization_id", "business_unit_id");
