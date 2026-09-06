--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--
--
-- Scheduling and availability.
--
-- The rota itself is never stored. It is composed on read from the shift
-- pattern, approved time off, open leave, and the driver's own stated
-- preference — every one of which changes without the rota being touched. A
-- stored week would be wrong the moment somebody's time off was approved, and
-- the wrongness would be invisible.
CREATE TYPE "availability_preference_enum" AS ENUM(
    'Preferred',
    'Available',
    'Unavailable'
);

--bun:split
CREATE TYPE "shift_swap_status_enum" AS ENUM(
    'Proposed',
    'Accepted',
    'Declined',
    'Approved',
    'Rejected',
    'Withdrawn'
);

--bun:split
CREATE TABLE IF NOT EXISTS "shift_templates"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "status" status_enum NOT NULL DEFAULT 'Active',
    "code" varchar(20) NOT NULL,
    "name" varchar(100) NOT NULL,
    "description" text,
    "color" varchar(10),
    -- Days are a seven-character mask indexed from Sunday, so a pattern reads
    -- at a glance and sorts and compares without unpacking an array.
    "days_of_week" varchar(7) NOT NULL DEFAULT '0111110',
    "start_minute" smallint NOT NULL DEFAULT 360,
    "duration_minutes" smallint NOT NULL DEFAULT 600,
    -- CycleWeeks lets an A/B rotation be one template rather than two: the
    -- assignment carries which week of the cycle a worker starts on.
    "cycle_weeks" smallint NOT NULL DEFAULT 1,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_shift_templates" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_shift_templates_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shift_templates_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_shift_templates_days" CHECK ("days_of_week" ~ '^[01]{7}$'),
    CONSTRAINT "chk_shift_templates_start" CHECK ("start_minute" >= 0 AND "start_minute" < 1440),
    CONSTRAINT "chk_shift_templates_duration" CHECK ("duration_minutes" > 0 AND "duration_minutes" <= 1440),
    CONSTRAINT "chk_shift_templates_cycle" CHECK ("cycle_weeks" >= 1 AND "cycle_weeks" <= 8)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_shift_templates_code" ON "shift_templates"("organization_id", "business_unit_id", lower("code"));

--bun:split
COMMENT ON TABLE shift_templates IS 'A repeating pattern of working days. The rota is composed from these on read rather than stored, because the things that override a pattern — time off, leave, an assignment — change without the pattern being touched.';

--bun:split
COMMENT ON COLUMN shift_templates.days_of_week IS 'Seven characters indexed from Sunday: 0111110 is Monday to Friday.';

--bun:split
CREATE TABLE IF NOT EXISTS "worker_shift_assignments"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "worker_id" varchar(100) NOT NULL,
    "shift_template_id" varchar(100) NOT NULL,
    "effective_from" bigint NOT NULL,
    "effective_to" bigint,
    "cycle_offset_weeks" smallint NOT NULL DEFAULT 0,
    "notes" text,
    "assigned_by_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_worker_shift_assignments" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_shift_assignments_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_shift_assignments_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_shift_assignments_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_shift_assignments_template" FOREIGN KEY ("shift_template_id", "organization_id", "business_unit_id") REFERENCES "shift_templates"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_worker_shift_assignments_window" CHECK ("effective_to" IS NULL OR "effective_to" >= "effective_from"),
    CONSTRAINT "chk_worker_shift_assignments_offset" CHECK ("cycle_offset_weeks" >= 0 AND "cycle_offset_weeks" <= 7)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_shift_assignments_open" ON "worker_shift_assignments"("organization_id", "business_unit_id", "worker_id")
WHERE
    "effective_to" IS NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_shift_assignments_worker" ON "worker_shift_assignments"("organization_id", "business_unit_id", "worker_id", "effective_from" DESC);

--bun:split
COMMENT ON COLUMN worker_shift_assignments.cycle_offset_weeks IS 'Which week of the template''s cycle this worker starts on, so an A/B rotation is one template and two offsets rather than two templates.';

--bun:split
CREATE TABLE IF NOT EXISTS "worker_availability_preferences"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "worker_id" varchar(100) NOT NULL,
    "day_of_week" smallint NOT NULL,
    "preference" availability_preference_enum NOT NULL DEFAULT 'Available',
    "note" varchar(255),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_worker_availability_preferences" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_availability_preferences_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_availability_preferences_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_availability_preferences_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_worker_availability_preferences_day" CHECK ("day_of_week" >= 0 AND "day_of_week" <= 6)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_availability_preferences_day" ON "worker_availability_preferences"("organization_id", "business_unit_id", "worker_id", "day_of_week");

--bun:split
COMMENT ON TABLE worker_availability_preferences IS 'What a driver would rather work. A preference is a statement, never a constraint: dispatch is free to override it, and the rota shows where it did so the override is visible rather than silent.';

--bun:split
CREATE TABLE IF NOT EXISTS "shift_swap_requests"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "requesting_worker_id" varchar(100) NOT NULL,
    "counterparty_worker_id" varchar(100),
    "status" shift_swap_status_enum NOT NULL DEFAULT 'Proposed',
    "shift_date" bigint NOT NULL,
    "counterparty_shift_date" bigint,
    "reason" varchar(255),
    "response_note" varchar(255),
    "responded_at" bigint,
    "decided_at" bigint,
    "decided_by_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_shift_swap_requests" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_shift_swap_requests_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shift_swap_requests_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shift_swap_requests_requester" FOREIGN KEY ("requesting_worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shift_swap_requests_counterparty" FOREIGN KEY ("counterparty_worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_shift_swap_requests_distinct" CHECK ("counterparty_worker_id" IS NULL OR "counterparty_worker_id" <> "requesting_worker_id")
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_shift_swap_requests_open" ON "shift_swap_requests"("organization_id", "business_unit_id", "status", "shift_date");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_shift_swap_requests_worker" ON "shift_swap_requests"("organization_id", "business_unit_id", "requesting_worker_id", "shift_date" DESC);

--bun:split
COMMENT ON TABLE shift_swap_requests IS 'One driver asking another to take a day. Two acceptances are needed: the colleague''s and a manager''s, because a swap the office never saw is a shift nobody is covering.';
