-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260925000000_rate_imports.tx.up.sql

CREATE TABLE IF NOT EXISTS "rate_import_batches"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "rate_agreement_id" TEXT NOT NULL,
    "file_name" TEXT NOT NULL,
    "source_format" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Pending',
    "effective_from" INTEGER NOT NULL,
    "mapping" TEXT,
    "unmapped_headers" TEXT,
    "changes" TEXT,
    "summary" TEXT,
    "row_count" INTEGER NOT NULL DEFAULT 0,
    "error_count" INTEGER NOT NULL DEFAULT 0,
    "error" TEXT,
    "uploaded_by_id" TEXT,
    "committed_at" INTEGER,
    "committed_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_rate_import_batches" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_rate_import_batches_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_import_batches_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_rate_import_batches_counts" CHECK ("row_count" >= 0 AND "error_count" >= 0),
    CONSTRAINT "chk_rate_import_batches_committed" CHECK ("status" <> 'Committed' OR ("committed_at" IS NOT NULL AND "committed_by_id" IS NOT NULL))
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_rate_import_batches_agreement" ON "rate_import_batches" ("organization_id", "business_unit_id", "rate_agreement_id", "created_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_rate_import_batches_open" ON "rate_import_batches" ("organization_id", "business_unit_id", "status")WHERE
    "status" IN ('Pending', 'Parsed');

--bun:split

CREATE TABLE IF NOT EXISTS "rate_import_rows"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "rate_import_batch_id" TEXT NOT NULL,
    "row_number" INTEGER NOT NULL,
    "cells" TEXT,
    "rule" TEXT,
    "lane_key" TEXT,
    "error" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_rate_import_rows" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_rate_import_rows_batch" FOREIGN KEY ("rate_import_batch_id", "business_unit_id", "organization_id") REFERENCES "rate_import_batches"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_rate_import_rows_batch" ON "rate_import_rows" ("rate_import_batch_id", "row_number");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_rate_import_rows_failed" ON "rate_import_rows" ("rate_import_batch_id")WHERE
    "error" IS NOT NULL;
