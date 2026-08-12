-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20241211020004_worker.tx.up.sql

CREATE TABLE IF NOT EXISTS "workers"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "state_id" TEXT NOT NULL,
    "manager_id" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "first_name" TEXT NOT NULL,
    "last_name" TEXT NOT NULL,
    "type" TEXT NOT NULL DEFAULT 'Employee',
    "driver_type" TEXT NOT NULL DEFAULT 'OTR',
    "profile_pic_url" TEXT,
    "address_line1" TEXT NOT NULL,
    "address_line2" TEXT,
    "city" TEXT NOT NULL,
    "postal_code" TEXT NOT NULL,
    "email" TEXT,
    "phone_number" TEXT,
    "emergency_contact_name" TEXT,
    "emergency_contact_phone" TEXT,
    "gender" TEXT NOT NULL,
    "external_id" TEXT,
    "can_be_assigned" INTEGER NOT NULL DEFAULT 0,
    "available_for_dispatch" INTEGER NOT NULL DEFAULT 1,
    "assignment_blocked" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_workers" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_workers_state" FOREIGN KEY ("state_id") REFERENCES "us_states"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_workers_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_workers_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_workers_manager" FOREIGN KEY ("manager_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_workers_business_unit" ON "workers" ("business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_workers_organization" ON "workers" ("organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_workers_state" ON "workers" ("state_id")WHERE
    state_id IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_workers_status" ON "workers" ("status");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_workers_type" ON "workers" ("type");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_workers_name" ON "workers" ("last_name", "first_name");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_workers_created_updated" ON "workers" ("created_at", "updated_at");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_workers_org_bu" ON "workers" ("organization_id", "business_unit_id");

--bun:split

ALTER TABLE "workers" ADD COLUMN "whole_name" TEXT GENERATED ALWAYS AS (first_name || ' ' || last_name) VIRTUAL;

--bun:split

CREATE TABLE IF NOT EXISTS "worker_profiles"(
    "id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "license_state_id" TEXT,
    "dob" INTEGER NOT NULL,
    "license_number" TEXT NOT NULL,
    "cdl_class" TEXT NOT NULL DEFAULT 'A',
    "cdl_restrictions" TEXT,
    "endorsement" TEXT NOT NULL DEFAULT 'O',
    "hazmat_expiry" INTEGER,
    "license_expiry" INTEGER NOT NULL CHECK (license_expiry > 0),
    "medical_card_expiry" INTEGER,
    "medical_examiner_name" TEXT,
    "medical_examiner_npi" TEXT,
    "twic_card_number" TEXT,
    "twic_expiry" INTEGER,
    "hire_date" INTEGER NOT NULL CHECK (hire_date > 0),
    "termination_date" INTEGER,
    "physical_due_date" INTEGER,
    "mvr_due_date" INTEGER,
    "compliance_status" TEXT NOT NULL DEFAULT 'Pending',
    "is_qualified" INTEGER NOT NULL DEFAULT 0,
    "disqualification_reason" TEXT,
    "last_compliance_check" INTEGER NOT NULL DEFAULT 0,
    "last_mvr_check" INTEGER NOT NULL DEFAULT 0,
    "last_drug_test" INTEGER NOT NULL DEFAULT 0,
    "eld_exempt" INTEGER NOT NULL DEFAULT 0,
    "short_haul_exempt" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_worker_profiles" PRIMARY KEY ("id", "worker_id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_profiles_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_profiles_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_profiles_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_profiles_license_state" FOREIGN KEY ("license_state_id") REFERENCES "us_states"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "check_endorsement_hazmat" CHECK (endorsement NOT IN ('H', 'X') OR hazmat_expiry IS NOT NULL)
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_profiles_unit_org" ON "worker_profiles" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_profiles_compliance_status" ON "worker_profiles" ("compliance_status", "is_qualified");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_profiles_dates" ON "worker_profiles" ("license_expiry", "hire_date", "termination_date", "physical_due_date", "mvr_due_date")WHERE
    license_expiry > 0 OR hire_date > 0 OR termination_date > 0 OR physical_due_date > 0 OR mvr_due_date > 0;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_profiles_last_checks" ON "worker_profiles" ("last_compliance_check", "last_mvr_check", "last_drug_test");

--bun:split

CREATE TABLE IF NOT EXISTS "worker_pto"(
    "id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "approver_id" TEXT,
    "rejector_id" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Requested',
    "type" TEXT NOT NULL DEFAULT 'Vacation',
    "start_date" INTEGER NOT NULL CHECK (start_date > 0),
    "end_date" INTEGER NOT NULL CHECK (end_date > 0),
    "reason" TEXT NOT NULL,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_worker_pto" PRIMARY KEY ("id", "worker_id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_pto_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_pto_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_pto_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_pto_approver" FOREIGN KEY ("approver_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_worker_pto_rejector" FOREIGN KEY ("rejector_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "check_end_date_after_start_date" CHECK ("end_date" > "start_date")
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_pto_created_updated" ON "worker_pto" ("created_at", "updated_at");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_pto_tenant_dates" ON "worker_pto" ("organization_id", "business_unit_id", "start_date", "end_date");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_pto_tenant_worker_dates" ON "worker_pto" ("organization_id", "business_unit_id", "worker_id", "start_date", "end_date");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_pto_tenant_status_dates" ON "worker_pto" ("organization_id", "business_unit_id", "status", "start_date", "end_date");
