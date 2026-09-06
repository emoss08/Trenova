--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TYPE "worker_employment_event_kind_enum" AS ENUM(
    'Hired',
    'ProbationEnded',
    'Promoted',
    'Transferred',
    'LeaveStarted',
    'LeaveEnded',
    'Suspended',
    'Reinstated',
    'Terminated',
    'Rehired',
    'RateChanged'
);

--bun:split
CREATE TABLE IF NOT EXISTS "worker_employment_events"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "worker_id" varchar(100) NOT NULL,
    "kind" worker_employment_event_kind_enum NOT NULL,
    "effective_at" bigint NOT NULL,
    "reason" varchar(255),
    "notes" text,
    "from_values" jsonb NOT NULL DEFAULT '{}'::jsonb,
    "to_values" jsonb NOT NULL DEFAULT '{}'::jsonb,
    "document_id" varchar(100),
    "recorded_by_id" varchar(100),
    "amended_by_id" varchar(100),
    "amended_at" bigint,
    "amendment_note" varchar(255),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_worker_employment_events" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_employment_events_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_employment_events_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_employment_events_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_employment_events_document" FOREIGN KEY ("document_id", "organization_id", "business_unit_id") REFERENCES "documents"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_worker_employment_events_effective" CHECK ("effective_at" > 0),
    CONSTRAINT "chk_worker_employment_events_amendment" CHECK (("amended_at" IS NULL) = ("amended_by_id" IS NULL))
);
--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_employment_events_worker" ON "worker_employment_events"("worker_id", "effective_at" DESC, "created_at" DESC);
--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_employment_events_kind" ON "worker_employment_events"("organization_id", "business_unit_id", "kind", "effective_at" DESC);
--bun:split
COMMENT ON TABLE worker_employment_events IS 'Append-only employment history per worker. Status-changing events (terminate, rehire, suspend, leave) are the only path that moves worker status; amendments correct reason, notes, dates or documents without replaying effects.';
--bun:split
INSERT INTO "worker_employment_events"("id", "business_unit_id", "organization_id", "worker_id", "kind", "effective_at", "to_values", "created_at", "updated_at")
SELECT
    'wee_' || upper(substr(md5(wp.worker_id || 'Hired'), 1, 26)),
    wp.business_unit_id,
    wp.organization_id,
    wp.worker_id,
    'Hired',
    wp.hire_date,
    jsonb_build_object('hireDate', wp.hire_date::text),
    wp.created_at,
    wp.updated_at
FROM
    "worker_profiles" wp
WHERE
    wp.hire_date > 0
ON CONFLICT DO NOTHING;
--bun:split
INSERT INTO "worker_employment_events"("id", "business_unit_id", "organization_id", "worker_id", "kind", "effective_at", "to_values", "created_at", "updated_at")
SELECT
    'wee_' || upper(substr(md5(wp.worker_id || 'Terminated'), 1, 26)),
    wp.business_unit_id,
    wp.organization_id,
    wp.worker_id,
    'Terminated',
    wp.termination_date,
    jsonb_build_object('terminationDate', wp.termination_date::text),
    wp.updated_at,
    wp.updated_at
FROM
    "worker_profiles" wp
WHERE
    wp.termination_date IS NOT NULL
    AND wp.termination_date > 0
ON CONFLICT DO NOTHING;
