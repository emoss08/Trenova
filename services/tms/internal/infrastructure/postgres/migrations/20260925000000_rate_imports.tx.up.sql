CREATE TYPE "rate_import_status_enum" AS ENUM(
    'Pending',
    'Parsed',
    'Committed',
    'Failed',
    'Discarded'
);

--bun:split
CREATE TYPE "rate_import_format_enum" AS ENUM(
    'CSV',
    'XLSX'
);

--bun:split
CREATE TABLE IF NOT EXISTS "rate_import_batches"(
    "id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "rate_agreement_id" VARCHAR(100) NOT NULL,
    "file_name" VARCHAR(255) NOT NULL,
    "source_format" rate_import_format_enum NOT NULL,
    "status" rate_import_status_enum NOT NULL DEFAULT 'Pending',
    "effective_from" BIGINT NOT NULL,
    "mapping" JSONB,
    "unmapped_headers" JSONB,
    "changes" JSONB,
    "summary" JSONB,
    "row_count" INTEGER NOT NULL DEFAULT 0,
    "error_count" INTEGER NOT NULL DEFAULT 0,
    "error" TEXT,
    "uploaded_by_id" VARCHAR(100),
    "committed_at" BIGINT,
    "committed_by_id" VARCHAR(100),
    "version" BIGINT NOT NULL DEFAULT 0,
    "created_at" BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_rate_import_batches" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_rate_import_batches_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_import_batches_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_rate_import_batches_counts" CHECK ("row_count" >= 0 AND "error_count" >= 0),
    -- A committed import has to say when and by whom. An import that changed a
    -- contract with nobody's name on it is exactly what an audit asks about.
    CONSTRAINT "chk_rate_import_batches_committed" CHECK ("status" <> 'Committed' OR ("committed_at" IS NOT NULL AND "committed_by_id" IS NOT NULL))
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_rate_import_batches_agreement" ON "rate_import_batches"("organization_id", "business_unit_id", "rate_agreement_id", "created_at" DESC);

--bun:split
-- The imports still waiting to be reviewed are what a screen asks for, and the
-- committed ones outnumber them permanently.
CREATE INDEX IF NOT EXISTS "idx_rate_import_batches_open" ON "rate_import_batches"("organization_id", "business_unit_id", "status")
WHERE
    "status" IN ('Pending', 'Parsed');

--bun:split
CREATE TABLE IF NOT EXISTS "rate_import_rows"(
    "id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "rate_import_batch_id" VARCHAR(100) NOT NULL,
    "row_number" INTEGER NOT NULL,
    "cells" JSONB,
    "rule" JSONB,
    "lane_key" VARCHAR(255),
    "error" TEXT,
    "created_at" BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_rate_import_rows" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_rate_import_rows_batch" FOREIGN KEY ("rate_import_batch_id", "business_unit_id", "organization_id") REFERENCES "rate_import_batches"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_rate_import_rows_batch" ON "rate_import_rows"("rate_import_batch_id", "row_number");

--bun:split
-- The rows that would not read are what somebody opens the import to fix, and
-- on a good sheet there are none of them among thousands.
CREATE INDEX IF NOT EXISTS "idx_rate_import_rows_failed" ON "rate_import_rows"("rate_import_batch_id")
WHERE
    "error" IS NOT NULL;
