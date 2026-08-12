-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250331182926_document_type.tx.up.sql

CREATE TABLE IF NOT EXISTS "document_types" (
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "code" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "color" TEXT,
    "document_classification" TEXT NOT NULL DEFAULT 'Public',
    "document_category" TEXT NOT NULL DEFAULT 'Other',
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_document_types" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_document_types_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_document_types_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_document_types_code" ON "document_types" (lower("code"), "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_document_types_name" ON "document_types" ("name");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_document_types_business_unit_organization" ON "document_types" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_document_types_created_updated" ON "document_types" ("created_at", "updated_at");

--bun:split

ALTER TABLE "documents" ADD COLUMN "document_type_id" TEXT NOT NULL;
