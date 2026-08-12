-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260327153000_add_document_control.tx.up.sql

CREATE TABLE IF NOT EXISTS "document_controls" (
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "enable_document_intelligence" INTEGER NOT NULL DEFAULT 1,
    "enable_ocr" INTEGER NOT NULL DEFAULT 1,
    "enable_auto_classification" INTEGER NOT NULL DEFAULT 1,
    "enable_auto_document_type_associate" INTEGER NOT NULL DEFAULT 1,
    "enable_auto_create_document_types" INTEGER NOT NULL DEFAULT 1,
    "enable_shipment_draft_extraction" INTEGER NOT NULL DEFAULT 1,
    "shipment_draft_allowed_resources" TEXT NOT NULL DEFAULT '{"shipment"}',
    "enable_full_text_indexing" INTEGER NOT NULL DEFAULT 1,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_document_controls" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_document_controls_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_document_controls_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uq_document_controls_organization" UNIQUE ("organization_id")
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_document_controls_tenant" ON "document_controls" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_document_controls_created_at" ON "document_controls" ("created_at");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_document_controls_updated_at" ON "document_controls" ("updated_at");
