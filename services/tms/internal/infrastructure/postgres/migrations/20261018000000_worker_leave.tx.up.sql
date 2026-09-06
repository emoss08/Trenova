--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TYPE "leave_measurement_method_enum" AS ENUM(
    'CalendarYear',
    'HireAnniversary',
    'ForwardFromFirstUse',
    'RollingBackward'
);

CREATE TYPE "leave_case_status_enum" AS ENUM(
    'Pending',
    'Approved',
    'Denied',
    'Closed'
);

CREATE TYPE "leave_frequency_enum" AS ENUM(
    'Continuous',
    'Intermittent',
    'ReducedSchedule'
);

CREATE TYPE "leave_certification_status_enum" AS ENUM(
    'NotRequired',
    'Requested',
    'Received',
    'Insufficient',
    'Overdue',
    'Waived'
);

--bun:split
CREATE TABLE IF NOT EXISTS "leave_controls"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "measurement_method" leave_measurement_method_enum NOT NULL DEFAULT 'RollingBackward',
    "entitlement_weeks" numeric(5, 2) NOT NULL DEFAULT 12,
    "military_caregiver_weeks" numeric(5, 2) NOT NULL DEFAULT 26,
    "workweek_hours" numeric(5, 2) NOT NULL DEFAULT 40,
    "eligibility_months" integer NOT NULL DEFAULT 12,
    "eligibility_hours" integer NOT NULL DEFAULT 1250,
    "certification_due_days" integer NOT NULL DEFAULT 15,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_leave_controls" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_leave_controls_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_leave_controls_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_leave_controls_weeks" CHECK ("entitlement_weeks" > 0 AND "military_caregiver_weeks" > 0),
    CONSTRAINT "chk_leave_controls_hours" CHECK ("workweek_hours" > 0 AND "eligibility_hours" >= 0),
    CONSTRAINT "chk_leave_controls_days" CHECK ("certification_due_days" > 0)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_leave_controls_tenant" ON "leave_controls"("organization_id", "business_unit_id");

--bun:split
COMMENT ON TABLE leave_controls IS 'One row per organisation. 29 CFR 825.200(e) requires the measurement method be applied consistently to every employee, so it is a single setting rather than a per-case choice.';

--bun:split
CREATE TABLE IF NOT EXISTS "worker_leave_cases"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "worker_id" varchar(100) NOT NULL,
    "leave_type" worker_leave_type_enum NOT NULL DEFAULT 'FMLA',
    "status" leave_case_status_enum NOT NULL DEFAULT 'Pending',
    "frequency" leave_frequency_enum NOT NULL DEFAULT 'Continuous',
    "reason" varchar(255),
    "fmla_designated" boolean NOT NULL DEFAULT FALSE,
    "military_caregiver" boolean NOT NULL DEFAULT FALSE,
    "requested_at" bigint NOT NULL,
    "starts_at" bigint NOT NULL,
    "ends_at" bigint,
    "decided_at" bigint,
    "closed_at" bigint,
    "certification_status" leave_certification_status_enum NOT NULL DEFAULT 'NotRequired',
    "certification_requested_at" bigint,
    "certification_due_at" bigint,
    "certification_received_at" bigint,
    "recertification_due_at" bigint,
    "eligibility_hours_worked" integer,
    "employment_event_id" varchar(100),
    "document_id" varchar(100),
    "notes" text,
    "decided_by_id" varchar(100),
    "recorded_by_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
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
CREATE INDEX IF NOT EXISTS "idx_worker_leave_cases_worker" ON "worker_leave_cases"("worker_id", "starts_at" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_leave_cases_open" ON "worker_leave_cases"("organization_id", "business_unit_id", "status")
WHERE
    "status" IN ('Pending', 'Approved');

--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_leave_cases_certification" ON "worker_leave_cases"("certification_due_at")
WHERE
    "certification_status" = 'Requested';

--bun:split
COMMENT ON TABLE worker_leave_cases IS 'One qualifying reason for leave. FMLA entitlement is drawn down by the entries under a case, so a case with no entries has used nothing however long it has been open.';

--bun:split
CREATE TABLE IF NOT EXISTS "worker_leave_entries"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "worker_id" varchar(100) NOT NULL,
    "leave_case_id" varchar(100) NOT NULL,
    "used_on" bigint NOT NULL,
    "hours" numeric(6, 2) NOT NULL,
    "counts_against_entitlement" boolean NOT NULL DEFAULT TRUE,
    "pto_id" varchar(100),
    "notes" text,
    "recorded_by_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_worker_leave_entries" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_leave_entries_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_leave_entries_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_leave_entries_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_leave_entries_case" FOREIGN KEY ("leave_case_id", "organization_id", "business_unit_id") REFERENCES "worker_leave_cases"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_leave_entries_pto" FOREIGN KEY ("pto_id", "worker_id", "organization_id", "business_unit_id") REFERENCES "worker_pto"("id", "worker_id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_worker_leave_entries_hours" CHECK ("hours" > 0 AND "hours" <= 24)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_leave_entries_day" ON "worker_leave_entries"("organization_id", "business_unit_id", "leave_case_id", "used_on");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_leave_entries_worker" ON "worker_leave_entries"("worker_id", "used_on" DESC);

--bun:split
COMMENT ON TABLE worker_leave_entries IS 'One day of leave taken. Intermittent leave is many rows; continuous leave is a row per working day. Hours rather than days, because 29 CFR 825.205 lets intermittent leave be taken in the smallest increment the employer uses for other absences.';

--bun:split
COMMENT ON COLUMN worker_leave_entries.pto_id IS 'The paid time off this day was also taken as. FMLA runs concurrently with paid leave, so a day is often both.';

--bun:split
INSERT INTO "leave_controls"("id", "business_unit_id", "organization_id")
SELECT
    'lctl_' || upper(substr(md5(o.id), 1, 26)),
    o.business_unit_id,
    o.id
FROM
    "organizations" o
ON CONFLICT
    DO NOTHING;
