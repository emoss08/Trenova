-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260713120000_edi_retention.tx.up.sql

ALTER TABLE "data_retention" ADD COLUMN "edi_inbound_file_retention_period" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "data_retention" ADD COLUMN "edi_message_retention_period" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "edi_inbound_files" ADD COLUMN "raw_purged_at" INTEGER;

--bun:split

ALTER TABLE "edi_messages" ADD COLUMN "raw_purged_at" INTEGER;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_inbound_files_raw_retention" ON "edi_inbound_files" ("organization_id", "received_at")WHERE
    "raw_purged_at" IS NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_messages_raw_retention" ON "edi_messages" ("organization_id", "created_at")WHERE
    "raw_purged_at" IS NULL;
