-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261022000000_scheduling.tx.up.sql

CREATE TABLE IF NOT EXISTS "shift_templates"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "code" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "color" TEXT,
    "days_of_week" TEXT NOT NULL DEFAULT '0111110',
    "start_minute" INTEGER NOT NULL DEFAULT 360,
    "duration_minutes" INTEGER NOT NULL DEFAULT 600,
    "cycle_weeks" INTEGER NOT NULL DEFAULT 1,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_shift_templates" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_shift_templates_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shift_templates_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_shift_templates_start" CHECK ("start_minute" >= 0 AND "start_minute" < 1440),
    CONSTRAINT "chk_shift_templates_duration" CHECK ("duration_minutes" > 0 AND "duration_minutes" <= 1440),
    CONSTRAINT "chk_shift_templates_cycle" CHECK ("cycle_weeks" >= 1 AND "cycle_weeks" <= 8)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_shift_templates_code" ON "shift_templates" ("organization_id", "business_unit_id", lower("code"));

--bun:split

CREATE TABLE IF NOT EXISTS "worker_shift_assignments"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "shift_template_id" TEXT NOT NULL,
    "effective_from" INTEGER NOT NULL,
    "effective_to" INTEGER,
    "cycle_offset_weeks" INTEGER NOT NULL DEFAULT 0,
    "notes" TEXT,
    "assigned_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_worker_shift_assignments" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_shift_assignments_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_shift_assignments_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_shift_assignments_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_shift_assignments_template" FOREIGN KEY ("shift_template_id", "organization_id", "business_unit_id") REFERENCES "shift_templates"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_worker_shift_assignments_window" CHECK ("effective_to" IS NULL OR "effective_to" >= "effective_from"),
    CONSTRAINT "chk_worker_shift_assignments_offset" CHECK ("cycle_offset_weeks" >= 0 AND "cycle_offset_weeks" <= 7)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_shift_assignments_open" ON "worker_shift_assignments" ("organization_id", "business_unit_id", "worker_id")WHERE
    "effective_to" IS NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_shift_assignments_worker" ON "worker_shift_assignments" ("organization_id", "business_unit_id", "worker_id", "effective_from" DESC);

--bun:split

CREATE TABLE IF NOT EXISTS "worker_availability_preferences"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "day_of_week" INTEGER NOT NULL,
    "preference" TEXT NOT NULL DEFAULT 'Available',
    "note" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_worker_availability_preferences" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_availability_preferences_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_availability_preferences_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_availability_preferences_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_worker_availability_preferences_day" CHECK ("day_of_week" >= 0 AND "day_of_week" <= 6)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_availability_preferences_day" ON "worker_availability_preferences" ("organization_id", "business_unit_id", "worker_id", "day_of_week");

--bun:split

CREATE TABLE IF NOT EXISTS "shift_swap_requests"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "requesting_worker_id" TEXT NOT NULL,
    "counterparty_worker_id" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Proposed',
    "shift_date" INTEGER NOT NULL,
    "counterparty_shift_date" INTEGER,
    "reason" TEXT,
    "response_note" TEXT,
    "responded_at" INTEGER,
    "decided_at" INTEGER,
    "decided_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_shift_swap_requests" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_shift_swap_requests_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shift_swap_requests_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shift_swap_requests_requester" FOREIGN KEY ("requesting_worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shift_swap_requests_counterparty" FOREIGN KEY ("counterparty_worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_shift_swap_requests_distinct" CHECK ("counterparty_worker_id" IS NULL OR "counterparty_worker_id" <> "requesting_worker_id")
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shift_swap_requests_open" ON "shift_swap_requests" ("organization_id", "business_unit_id", "status", "shift_date");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shift_swap_requests_worker" ON "shift_swap_requests" ("organization_id", "business_unit_id", "requesting_worker_id", "shift_date" DESC);
