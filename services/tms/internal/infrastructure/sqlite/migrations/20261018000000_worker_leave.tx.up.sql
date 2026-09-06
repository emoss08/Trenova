-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261018000000_worker_leave.tx.up.sql

CREATE TABLE IF NOT EXISTS "leave_controls"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "measurement_method" TEXT NOT NULL DEFAULT 'RollingBackward',
    "entitlement_weeks" REAL NOT NULL DEFAULT 12,
    "military_caregiver_weeks" REAL NOT NULL DEFAULT 26,
    "workweek_hours" REAL NOT NULL DEFAULT 40,
    "eligibility_months" INTEGER NOT NULL DEFAULT 12,
    "eligibility_hours" INTEGER NOT NULL DEFAULT 1250,
    "certification_due_days" INTEGER NOT NULL DEFAULT 15,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_leave_controls" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_leave_controls_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_leave_controls_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_leave_controls_weeks" CHECK ("entitlement_weeks" > 0 AND "military_caregiver_weeks" > 0),
    CONSTRAINT "chk_leave_controls_hours" CHECK ("workweek_hours" > 0 AND "eligibility_hours" >= 0),
    CONSTRAINT "chk_leave_controls_days" CHECK ("certification_due_days" > 0)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_leave_controls_tenant" ON "leave_controls" ("organization_id", "business_unit_id");

--bun:split

CREATE TABLE IF NOT EXISTS "worker_leave_cases"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "leave_type" TEXT NOT NULL DEFAULT 'FMLA',
    "status" TEXT NOT NULL DEFAULT 'Pending',
    "frequency" TEXT NOT NULL DEFAULT 'Continuous',
    "reason" TEXT,
    "fmla_designated" INTEGER NOT NULL DEFAULT 0,
    "military_caregiver" INTEGER NOT NULL DEFAULT 0,
    "requested_at" INTEGER NOT NULL,
    "starts_at" INTEGER NOT NULL,
    "ends_at" INTEGER,
    "decided_at" INTEGER,
    "closed_at" INTEGER,
    "certification_status" TEXT NOT NULL DEFAULT 'NotRequired',
    "certification_requested_at" INTEGER,
    "certification_due_at" INTEGER,
    "certification_received_at" INTEGER,
    "recertification_due_at" INTEGER,
    "eligibility_hours_worked" INTEGER,
    "employment_event_id" TEXT,
    "document_id" TEXT,
    "notes" TEXT,
    "decided_by_id" TEXT,
    "recorded_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_worker_leave_cases" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_leave_cases_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_leave_cases_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_leave_cases_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_leave_cases_event" FOREIGN KEY ("employment_event_id", "organization_id", "business_unit_id") REFERENCES "worker_employment_events"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_worker_leave_cases_document" FOREIGN KEY ("document_id", "organization_id", "business_unit_id") REFERENCES "documents"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_worker_leave_cases_window" CHECK ("ends_at" IS NULL OR "ends_at" >= "starts_at"),
    CONSTRAINT "chk_worker_leave_cases_decided" CHECK ("status" IN ('Pending') OR "decided_at" IS NOT NULL),
    CONSTRAINT "chk_worker_leave_cases_closed" CHECK (("status" = 'Closed') = ("closed_at" IS NOT NULL)),
    CONSTRAINT "chk_worker_leave_cases_certification" CHECK ("certification_status" NOT IN ('Requested', 'Received', 'Insufficient', 'Overdue') OR "certification_requested_at" IS NOT NULL),
    CONSTRAINT "chk_worker_leave_cases_eligibility_hours" CHECK ("eligibility_hours_worked" IS NULL OR "eligibility_hours_worked" >= 0)
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_leave_cases_worker" ON "worker_leave_cases" ("worker_id", "starts_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_leave_cases_open" ON "worker_leave_cases" ("organization_id", "business_unit_id", "status")WHERE
    "status" IN ('Pending', 'Approved');

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_leave_cases_certification" ON "worker_leave_cases" ("certification_due_at")WHERE
    "certification_status" = 'Requested';

--bun:split

CREATE TABLE IF NOT EXISTS "worker_leave_entries"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "leave_case_id" TEXT NOT NULL,
    "used_on" INTEGER NOT NULL,
    "hours" REAL NOT NULL,
    "counts_against_entitlement" INTEGER NOT NULL DEFAULT 1,
    "pto_id" TEXT,
    "notes" TEXT,
    "recorded_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_worker_leave_entries" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_leave_entries_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_leave_entries_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_leave_entries_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_leave_entries_case" FOREIGN KEY ("leave_case_id", "organization_id", "business_unit_id") REFERENCES "worker_leave_cases"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_leave_entries_pto" FOREIGN KEY ("pto_id", "worker_id", "organization_id", "business_unit_id") REFERENCES "worker_pto"("id", "worker_id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_worker_leave_entries_hours" CHECK ("hours" > 0 AND "hours" <= 24)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_leave_entries_day" ON "worker_leave_entries" ("organization_id", "business_unit_id", "leave_case_id", "used_on");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_leave_entries_worker" ON "worker_leave_entries" ("worker_id", "used_on" DESC);
