-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260328113000_add_document_versioning_and_packet_rules.tx.up.sql

ALTER TABLE "documents" ADD COLUMN "lineage_id" TEXT;

--bun:split

ALTER TABLE "documents" ADD COLUMN "version_number" INTEGER NOT NULL DEFAULT 1;

--bun:split

ALTER TABLE "documents" ADD COLUMN "is_current_version" INTEGER NOT NULL DEFAULT 1;

--bun:split

ALTER TABLE "documents" ADD COLUMN "checksum_sha256" TEXT;

--bun:split

ALTER TABLE "documents" ADD COLUMN "storage_version_id" TEXT;

--bun:split

ALTER TABLE "documents" ADD COLUMN "storage_retention_mode" TEXT;

--bun:split

ALTER TABLE "documents" ADD COLUMN "storage_retention_until" INTEGER;

--bun:split

ALTER TABLE "documents" ADD COLUMN "storage_legal_hold" INTEGER NOT NULL DEFAULT 0;

--bun:split

UPDATE
    "documents"
SET
    "lineage_id" = "id"
WHERE
    "lineage_id" IS NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_documents_lineage_id" ON "documents" ("lineage_id", "organization_id", "business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_documents_current_version" ON "documents" ("resource_type", "resource_id", "is_current_version");

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_documents_current_lineage" ON "documents" ("lineage_id", "organization_id", "business_unit_id")WHERE
    "is_current_version" = TRUE;

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_documents_lineage_version_number" ON "documents" ("lineage_id", "organization_id", "business_unit_id", "version_number");

--bun:split

ALTER TABLE "document_upload_sessions" ADD COLUMN "lineage_id" TEXT;

--bun:split

CREATE TABLE IF NOT EXISTS "document_packet_rules"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "resource_type" TEXT NOT NULL DEFAULT 'Shipment',
    "document_type_id" TEXT NOT NULL,
    "required" INTEGER NOT NULL DEFAULT 0,
    "allow_multiple" INTEGER NOT NULL DEFAULT 0,
    "display_order" INTEGER NOT NULL DEFAULT 0,
    "expiration_required" INTEGER NOT NULL DEFAULT 0,
    "expiration_warning_days" INTEGER NOT NULL DEFAULT 30,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_document_packet_rules" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_document_packet_rules_document_type" FOREIGN KEY ("document_type_id", "organization_id", "business_unit_id") REFERENCES "document_types"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "ck_document_packet_rules_expiration_warning_days" CHECK ("expiration_warning_days" >= 0)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_document_packet_rules_resource_type_document_type" ON "document_packet_rules" ("resource_type", "document_type_id", "organization_id", "business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_document_packet_rules_resource_type" ON "document_packet_rules" ("resource_type", "organization_id", "business_unit_id");
