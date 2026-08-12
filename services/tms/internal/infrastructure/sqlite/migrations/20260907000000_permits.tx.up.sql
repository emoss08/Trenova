-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260907000000_permits.tx.up.sql

CREATE TABLE IF NOT EXISTS "permits"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "shipment_id" TEXT NOT NULL,
    "state_id" TEXT NOT NULL,
    "permit_number" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Pending',
    "issued_at" INTEGER,
    "expires_at" INTEGER,
    "cost" REAL,
    "document_id" TEXT,
    "notes" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_permits" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_permits_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_permits_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_permits_shipment" FOREIGN KEY ("shipment_id", "organization_id", "business_unit_id") REFERENCES "shipments"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_permits_state" FOREIGN KEY ("state_id") REFERENCES "us_states"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_permits_dates" CHECK ("issued_at" IS NULL OR "expires_at" IS NULL OR "expires_at" > "issued_at"),
    CONSTRAINT "chk_permits_active_expiry" CHECK ("status" <> 'Active' OR "expires_at" IS NOT NULL),
    CONSTRAINT "chk_permits_cost" CHECK ("cost" IS NULL OR "cost" >= 0)
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_permits_shipment" ON "permits" ("shipment_id", "state_id", "status");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_permits_expiring" ON "permits" ("organization_id", "business_unit_id", "expires_at")WHERE
    "status" = 'Active';

--bun:split

CREATE TABLE IF NOT EXISTS "permit_requirements"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "shipment_id" TEXT NOT NULL,
    "state_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Open',
    "route_sequence" INTEGER NOT NULL DEFAULT 0,
    "exceedances" TEXT,
    "escorts" TEXT,
    "restrictions" TEXT,
    "lead_time_days" INTEGER NOT NULL DEFAULT 0,
    "validity_days" INTEGER NOT NULL DEFAULT 0,
    "estimated_fee" REAL,
    "is_superload" INTEGER NOT NULL DEFAULT 0,
    "satisfied_by_permit_id" TEXT,
    "waived_by_id" TEXT,
    "waiver_reason" TEXT,
    "provenance" TEXT,
    "derived_at" INTEGER NOT NULL,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_permit_requirements" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_permit_requirements_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_permit_requirements_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_permit_requirements_shipment" FOREIGN KEY ("shipment_id", "organization_id", "business_unit_id") REFERENCES "shipments"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_permit_requirements_state" FOREIGN KEY ("state_id") REFERENCES "us_states"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_permit_requirements_permit" FOREIGN KEY ("satisfied_by_permit_id", "organization_id", "business_unit_id") REFERENCES "permits"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_permit_requirements_waived_by" FOREIGN KEY ("waived_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_permit_requirements_satisfied" CHECK ("status" <> 'Satisfied' OR "satisfied_by_permit_id" IS NOT NULL),
    CONSTRAINT "chk_permit_requirements_waived" CHECK ("status" <> 'Waived' OR ("waiver_reason" IS NOT NULL AND length("waiver_reason") >= 10))
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_permit_requirements_shipment" ON "permit_requirements" ("shipment_id", "route_sequence");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_permit_requirements_open" ON "permit_requirements" ("organization_id", "business_unit_id", "derived_at" DESC)WHERE
    "status" = 'Open';
