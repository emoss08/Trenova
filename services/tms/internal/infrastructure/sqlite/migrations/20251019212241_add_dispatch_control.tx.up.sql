-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20251019212241_add_dispatch_control.tx.up.sql

CREATE TABLE IF NOT EXISTS "dispatch_controls"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "enable_auto_assignment" INTEGER NOT NULL DEFAULT 0,
    "auto_assignment_strategy" TEXT NOT NULL DEFAULT 'Proximity',
    "enforce_worker_assign" INTEGER NOT NULL DEFAULT 0,
    "enforce_trailer_continuity" INTEGER NOT NULL DEFAULT 0,
    "enforce_hos_compliance" INTEGER NOT NULL DEFAULT 0,
    "enforce_worker_pta_restrictions" INTEGER NOT NULL DEFAULT 0,
    "enforce_worker_tractor_fleet_continuity" INTEGER NOT NULL DEFAULT 0,
    "enforce_driver_qualification_compliance" INTEGER NOT NULL DEFAULT 0,
    "enforce_medical_cert_compliance" INTEGER NOT NULL DEFAULT 0,
    "enforce_hazmat_compliance" INTEGER NOT NULL DEFAULT 0,
    "enforce_drug_and_alcohol_compliance" INTEGER NOT NULL DEFAULT 0,
    "compliance_enforcement_level" TEXT NOT NULL DEFAULT 'Warning',
    "record_service_failures" TEXT NOT NULL DEFAULT 'Never',
    "service_failure_target" TEXT,
    "service_failure_grace_period" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_dispatch_controls" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_dispatch_controls_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_dispatch_controls_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uq_dispatch_controls_organization" UNIQUE ("organization_id")
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_dispatch_controls_business_unit" ON "dispatch_controls" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_dispatch_controls_created_at" ON "dispatch_controls" ("created_at", "updated_at");
