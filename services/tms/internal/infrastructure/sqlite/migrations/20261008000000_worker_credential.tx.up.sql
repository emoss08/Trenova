-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261008000000_worker_credential.tx.up.sql

CREATE TABLE IF NOT EXISTS "worker_credential_types"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "code" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "category" TEXT NOT NULL DEFAULT 'Other',
    "status" TEXT NOT NULL DEFAULT 'Active',
    "is_required" INTEGER NOT NULL DEFAULT 0,
    "required_for_driver_types" TEXT NOT NULL DEFAULT '[]',
    "renewal_window_days" INTEGER NOT NULL DEFAULT 30,
    "validity_months" INTEGER,
    "requires_number" INTEGER NOT NULL DEFAULT 0,
    "requires_document" INTEGER NOT NULL DEFAULT 0,
    "profile_field" TEXT,
    "is_system" INTEGER NOT NULL DEFAULT 0,
    "sort_order" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_worker_credential_types" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_credential_types_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_credential_types_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_worker_credential_types_renewal_window" CHECK ("renewal_window_days" >= 0),
    CONSTRAINT "chk_worker_credential_types_validity" CHECK ("validity_months" IS NULL OR "validity_months" > 0)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_credential_types_code" ON "worker_credential_types" ("organization_id", "business_unit_id", LOWER("code"));

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_credential_types_profile_field" ON "worker_credential_types" ("organization_id", "business_unit_id", "profile_field")WHERE
    "profile_field" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_credential_types_status" ON "worker_credential_types" ("organization_id", "business_unit_id", "status", "sort_order");

--bun:split

CREATE TABLE IF NOT EXISTS "worker_credentials"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "credential_type_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "number" TEXT,
    "issuing_authority" TEXT,
    "issued_at" INTEGER,
    "expires_at" INTEGER,
    "document_id" TEXT,
    "notes" TEXT,
    "verified_by_id" TEXT,
    "verified_at" INTEGER,
    "archived_by_id" TEXT,
    "archived_at" INTEGER,
    "archive_reason" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_worker_credentials" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_credentials_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_credentials_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_credentials_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_credentials_type" FOREIGN KEY ("credential_type_id", "organization_id", "business_unit_id") REFERENCES "worker_credential_types"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_worker_credentials_document" FOREIGN KEY ("document_id", "organization_id", "business_unit_id") REFERENCES "documents"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_worker_credentials_dates" CHECK ("issued_at" IS NULL OR "expires_at" IS NULL OR "expires_at" > "issued_at"),
    CONSTRAINT "chk_worker_credentials_archive" CHECK (("status" = 'Archived') = ("archived_at" IS NOT NULL))
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_credentials_active_type" ON "worker_credentials" ("organization_id", "business_unit_id", "worker_id", "credential_type_id")WHERE
    "status" = 'Active';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_credentials_worker" ON "worker_credentials" ("worker_id", "status", "expires_at");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_credentials_expiry" ON "worker_credentials" ("expires_at")WHERE
    "status" = 'Active' AND "expires_at" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_credentials_document" ON "worker_credentials" ("document_id")WHERE
    "document_id" IS NOT NULL;
