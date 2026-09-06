--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TYPE "dot_test_type_enum" AS ENUM(
    'PreEmployment',
    'Random',
    'PostAccident',
    'ReasonableSuspicion',
    'ReturnToDuty',
    'FollowUp',
    'Other'
);

CREATE TYPE "dot_test_substance_enum" AS ENUM(
    'Drug',
    'Alcohol'
);

CREATE TYPE "dot_test_status_enum" AS ENUM(
    'Scheduled',
    'Collected',
    'AwaitingResult',
    'Completed',
    'Cancelled'
);

CREATE TYPE "dot_test_result_enum" AS ENUM(
    'Pending',
    'Negative',
    'NegativeDilute',
    'Positive',
    'Refusal',
    'Adulterated',
    'Substituted',
    'Invalid',
    'Cancelled'
);

CREATE TYPE "dot_random_period_enum" AS ENUM(
    'Monthly',
    'Quarterly',
    'SemiAnnual',
    'Annual'
);

CREATE TYPE "dot_random_draw_status_enum" AS ENUM(
    'Draft',
    'Final',
    'Cancelled'
);

CREATE TYPE "dot_random_entry_status_enum" AS ENUM(
    'Selected',
    'Notified',
    'Completed',
    'Excused',
    'Missed'
);

CREATE TYPE "dot_violation_type_enum" AS ENUM(
    'PositiveTest',
    'TestRefusal',
    'AlcoholUse',
    'DrugUse',
    'ActualKnowledge',
    'Other'
);

CREATE TYPE "dot_violation_status_enum" AS ENUM(
    'Open',
    'SAPEvaluation',
    'RTDPending',
    'FollowUp',
    'Resolved'
);

CREATE TYPE "dot_clearinghouse_query_type_enum" AS ENUM(
    'PreEmploymentFull',
    'AnnualLimited',
    'Full',
    'Limited'
);

CREATE TYPE "dot_clearinghouse_result_enum" AS ENUM(
    'Pending',
    'NoViolations',
    'ViolationsFound',
    'ConsentDenied'
);

CREATE TYPE "worker_drug_alcohol_status_enum" AS ENUM(
    'Unknown',
    'Clear',
    'Pending',
    'Prohibited'
);

CREATE TYPE "worker_return_to_duty_status_enum" AS ENUM(
    'NotRequired',
    'SAPEvaluation',
    'RTDTestRequired',
    'FollowUpTesting',
    'Complete'
);

--bun:split
CREATE TABLE IF NOT EXISTS "dot_random_pools"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "code" varchar(50) NOT NULL,
    "name" varchar(100) NOT NULL,
    "description" text,
    "status" status_enum NOT NULL DEFAULT 'Active',
    "period" dot_random_period_enum NOT NULL DEFAULT 'Quarterly',
    "drug_rate_percent" smallint NOT NULL DEFAULT 50,
    "alcohol_rate_percent" smallint NOT NULL DEFAULT 10,
    "included_driver_types" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "is_default" boolean NOT NULL DEFAULT FALSE,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_dot_random_pools" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_dot_random_pools_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_dot_random_pools_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_dot_random_pools_drug_rate" CHECK ("drug_rate_percent" >= 0 AND "drug_rate_percent" <= 100),
    CONSTRAINT "chk_dot_random_pools_alcohol_rate" CHECK ("alcohol_rate_percent" >= 0 AND "alcohol_rate_percent" <= 100)
);

--bun:split
ALTER TABLE "dot_random_pools"
    ADD COLUMN IF NOT EXISTS search_vector tsvector GENERATED ALWAYS AS (setweight(immutable_to_tsvector('simple', COALESCE("code", '')), 'A') || setweight(immutable_to_tsvector('simple', COALESCE("name", '')), 'A') || setweight(immutable_to_tsvector('simple', COALESCE("description", '')), 'B')) STORED;

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_dot_random_pools_code" ON "dot_random_pools"("organization_id", "business_unit_id", LOWER("code"));

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_dot_random_pools_default" ON "dot_random_pools"("organization_id", "business_unit_id")
WHERE
    "is_default";

--bun:split
CREATE INDEX IF NOT EXISTS "idx_dot_random_pools_search" ON "dot_random_pools" USING GIN(search_vector);

--bun:split
COMMENT ON TABLE dot_random_pools IS 'A random testing pool with the annual rates it must meet (49 CFR 382.305). Empty included_driver_types means every safety-sensitive driver.';

--bun:split
CREATE TABLE IF NOT EXISTS "dot_random_draws"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "pool_id" varchar(100) NOT NULL,
    "period_key" varchar(20) NOT NULL,
    "period_start" bigint NOT NULL,
    "period_end" bigint NOT NULL,
    "status" dot_random_draw_status_enum NOT NULL DEFAULT 'Draft',
    "pool_size" integer NOT NULL DEFAULT 0,
    "drug_target" integer NOT NULL DEFAULT 0,
    "alcohol_target" integer NOT NULL DEFAULT 0,
    "drug_selected" integer NOT NULL DEFAULT 0,
    "alcohol_selected" integer NOT NULL DEFAULT 0,
    "seed" varchar(64) NOT NULL,
    "method" varchar(60) NOT NULL,
    "notes" text,
    "drawn_at" bigint NOT NULL,
    "drawn_by_id" varchar(100),
    "finalized_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_dot_random_draws" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_dot_random_draws_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_dot_random_draws_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_dot_random_draws_pool" FOREIGN KEY ("pool_id", "organization_id", "business_unit_id") REFERENCES "dot_random_pools"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_dot_random_draws_period" CHECK ("period_end" > "period_start"),
    CONSTRAINT "chk_dot_random_draws_final" CHECK (("status" = 'Final') = ("finalized_at" IS NOT NULL))
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_dot_random_draws_period" ON "dot_random_draws"("organization_id", "business_unit_id", "pool_id", "period_key")
WHERE
    "status" <> 'Cancelled';

--bun:split
CREATE INDEX IF NOT EXISTS "idx_dot_random_draws_drawn" ON "dot_random_draws"("organization_id", "business_unit_id", "drawn_at" DESC);

--bun:split
COMMENT ON TABLE dot_random_draws IS 'One selection round for a pool and period. The seed and method are the evidence of how the draw was made: the same seed over the same roster reproduces the same names.';

--bun:split
CREATE TABLE IF NOT EXISTS "dot_random_draw_entries"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "draw_id" varchar(100) NOT NULL,
    "worker_id" varchar(100) NOT NULL,
    "substance" dot_test_substance_enum NOT NULL,
    "rank" integer NOT NULL DEFAULT 0,
    "status" dot_random_entry_status_enum NOT NULL DEFAULT 'Selected',
    "notified_at" bigint,
    "completed_at" bigint,
    "test_id" varchar(100),
    "excuse_reason" varchar(255),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_dot_random_draw_entries" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_dot_random_draw_entries_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_dot_random_draw_entries_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_dot_random_draw_entries_draw" FOREIGN KEY ("draw_id", "organization_id", "business_unit_id") REFERENCES "dot_random_draws"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_dot_random_draw_entries_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_dot_random_draw_entries_excuse" CHECK (("status" <> 'Excused') OR ("excuse_reason" IS NOT NULL))
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_dot_random_draw_entries_worker" ON "dot_random_draw_entries"("organization_id", "business_unit_id", "draw_id", "worker_id", "substance");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_dot_random_draw_entries_worker" ON "dot_random_draw_entries"("worker_id", "status");

--bun:split
COMMENT ON TABLE dot_random_draw_entries IS 'A worker drawn for one substance in one round. Rank is the position the draw produced, kept so the selection can be re-checked against the seed.';

--bun:split
CREATE TABLE IF NOT EXISTS "worker_dot_tests"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "worker_id" varchar(100) NOT NULL,
    "test_type" dot_test_type_enum NOT NULL,
    "substance" dot_test_substance_enum NOT NULL,
    "status" dot_test_status_enum NOT NULL DEFAULT 'Scheduled',
    "result" dot_test_result_enum NOT NULL DEFAULT 'Pending',
    "is_dot" boolean NOT NULL DEFAULT TRUE,
    "reason" text,
    "scheduled_at" bigint,
    "collected_at" bigint,
    "result_at" bigint,
    "collection_site" varchar(150),
    "collector_name" varchar(100),
    "specimen_id" varchar(100),
    "lab_name" varchar(100),
    "mro_name" varchar(100),
    "mro_verified_at" bigint,
    "alcohol_concentration" numeric(4, 3),
    "safety_event_id" varchar(100),
    "draw_entry_id" varchar(100),
    "document_id" varchar(100),
    "notes" text,
    "ordered_by_id" varchar(100),
    "recorded_by_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_worker_dot_tests" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_dot_tests_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_dot_tests_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_dot_tests_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_dot_tests_safety_event" FOREIGN KEY ("safety_event_id", "organization_id", "business_unit_id") REFERENCES "worker_safety_events"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_worker_dot_tests_draw_entry" FOREIGN KEY ("draw_entry_id", "organization_id", "business_unit_id") REFERENCES "dot_random_draw_entries"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_worker_dot_tests_document" FOREIGN KEY ("document_id", "organization_id", "business_unit_id") REFERENCES "documents"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_worker_dot_tests_mro" CHECK ("substance" = 'Drug' OR ("mro_name" IS NULL AND "mro_verified_at" IS NULL)),
    CONSTRAINT "chk_worker_dot_tests_alcohol" CHECK ("substance" = 'Alcohol' OR "alcohol_concentration" IS NULL),
    CONSTRAINT "chk_worker_dot_tests_concentration" CHECK ("alcohol_concentration" IS NULL OR ("alcohol_concentration" >= 0 AND "alcohol_concentration" <= 1)),
    CONSTRAINT "chk_worker_dot_tests_completed" CHECK ("status" <> 'Completed' OR ("result" <> 'Pending' AND "result_at" IS NOT NULL)),
    CONSTRAINT "chk_worker_dot_tests_collected" CHECK ("status" IN ('Scheduled', 'Cancelled') OR "collected_at" IS NOT NULL)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_dot_tests_draw_entry" ON "worker_dot_tests"("organization_id", "business_unit_id", "draw_entry_id")
WHERE
    "draw_entry_id" IS NOT NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_dot_tests_worker" ON "worker_dot_tests"("worker_id", "collected_at" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_dot_tests_open" ON "worker_dot_tests"("organization_id", "business_unit_id", "status")
WHERE
    "status" IN ('Scheduled', 'Collected', 'AwaitingResult');

--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_dot_tests_type" ON "worker_dot_tests"("organization_id", "business_unit_id", "test_type", "collected_at" DESC);

--bun:split
COMMENT ON TABLE worker_dot_tests IS 'One row per substance analysed. A collection covering both drug and alcohol is two rows, because only the drug half has an MRO and only the alcohol half has a concentration.';

--bun:split
CREATE TABLE IF NOT EXISTS "worker_dot_violations"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "worker_id" varchar(100) NOT NULL,
    "violation_type" dot_violation_type_enum NOT NULL,
    "status" dot_violation_status_enum NOT NULL DEFAULT 'Open',
    "occurred_at" bigint NOT NULL,
    "source_test_id" varchar(100),
    "reported_to_clearinghouse_at" bigint,
    "sap_name" varchar(100),
    "sap_referred_at" bigint,
    "sap_evaluation_completed_at" bigint,
    "rtd_test_id" varchar(100),
    "rtd_completed_at" bigint,
    "follow_up_test_count" integer NOT NULL DEFAULT 0,
    "follow_up_tests_completed" integer NOT NULL DEFAULT 0,
    "follow_up_ends_at" bigint,
    "resolved_at" bigint,
    "document_id" varchar(100),
    "notes" text,
    "recorded_by_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_worker_dot_violations" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_dot_violations_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_dot_violations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_dot_violations_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_dot_violations_source_test" FOREIGN KEY ("source_test_id", "organization_id", "business_unit_id") REFERENCES "worker_dot_tests"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_worker_dot_violations_rtd_test" FOREIGN KEY ("rtd_test_id", "organization_id", "business_unit_id") REFERENCES "worker_dot_tests"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_worker_dot_violations_document" FOREIGN KEY ("document_id", "organization_id", "business_unit_id") REFERENCES "documents"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_worker_dot_violations_follow_up" CHECK ("follow_up_test_count" >= 0 AND "follow_up_tests_completed" >= 0 AND "follow_up_tests_completed" <= "follow_up_test_count"),
    CONSTRAINT "chk_worker_dot_violations_resolved" CHECK (("status" = 'Resolved') = ("resolved_at" IS NOT NULL))
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_dot_violations_open" ON "worker_dot_violations"("organization_id", "business_unit_id", "worker_id")
WHERE
    "status" <> 'Resolved';

--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_dot_violations_worker" ON "worker_dot_violations"("worker_id", "occurred_at" DESC);

--bun:split
COMMENT ON TABLE worker_dot_violations IS 'A standing prohibition and the return-to-duty process that clears it (49 CFR 382 Subpart O). At most one unresolved row per worker: a second violation during the process extends the one that is open.';

--bun:split
CREATE TABLE IF NOT EXISTS "worker_clearinghouse_queries"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "worker_id" varchar(100) NOT NULL,
    "query_type" dot_clearinghouse_query_type_enum NOT NULL,
    "result" dot_clearinghouse_result_enum NOT NULL DEFAULT 'Pending',
    "consent_obtained_at" bigint,
    "consent_expires_at" bigint,
    "requested_at" bigint NOT NULL,
    "completed_at" bigint,
    "violation_count" integer NOT NULL DEFAULT 0,
    "reference" varchar(100),
    "document_id" varchar(100),
    "notes" text,
    "performed_by_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_worker_clearinghouse_queries" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_clearinghouse_queries_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_clearinghouse_queries_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_clearinghouse_queries_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_clearinghouse_queries_document" FOREIGN KEY ("document_id", "organization_id", "business_unit_id") REFERENCES "documents"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_worker_clearinghouse_queries_completed" CHECK (("result" = 'Pending') = ("completed_at" IS NULL)),
    CONSTRAINT "chk_worker_clearinghouse_queries_violations" CHECK ("violation_count" >= 0 AND ("result" <> 'ViolationsFound' OR "violation_count" > 0))
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_clearinghouse_queries_worker" ON "worker_clearinghouse_queries"("worker_id", "requested_at" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_clearinghouse_queries_pending" ON "worker_clearinghouse_queries"("organization_id", "business_unit_id", "requested_at")
WHERE
    "result" = 'Pending';

--bun:split
COMMENT ON TABLE worker_clearinghouse_queries IS 'FMCSA Clearinghouse queries (49 CFR 382 Subpart G): the full query before hire and the limited query every year, each with the consent that authorised it.';

--bun:split
ALTER TABLE "worker_profiles"
    ADD COLUMN IF NOT EXISTS "drug_alcohol_status" worker_drug_alcohol_status_enum NOT NULL DEFAULT 'Unknown',
    ADD COLUMN IF NOT EXISTS "return_to_duty_status" worker_return_to_duty_status_enum NOT NULL DEFAULT 'NotRequired',
    ADD COLUMN IF NOT EXISTS "last_clearinghouse_query_at" bigint,
    ADD COLUMN IF NOT EXISTS "next_clearinghouse_query_due" bigint;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_profiles_drug_alcohol" ON "worker_profiles"("organization_id", "business_unit_id", "drug_alcohol_status");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_profiles_clearinghouse_due" ON "worker_profiles"("next_clearinghouse_query_due")
WHERE
    "next_clearinghouse_query_due" IS NOT NULL;

--bun:split
COMMENT ON COLUMN worker_profiles.drug_alcohol_status IS 'Roll-up of the testing record, refreshed whenever a test, violation or query changes. Unknown means nothing is on file, not that the driver is clear.';

--bun:split
INSERT INTO "dot_random_pools"("id", "business_unit_id", "organization_id", "code", "name", "description", "period", "drug_rate_percent", "alcohol_rate_percent", "is_default")
SELECT
    'drpool_' || upper(substr(md5(o.id || 'DOT'), 1, 24)),
    o.business_unit_id,
    o.id,
    'DOT',
    'DOT Random Pool',
    'Every safety-sensitive driver, drawn each quarter at the FMCSA minimum rates.',
    'Quarterly',
    50,
    10,
    TRUE
FROM
    "organizations" o
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "worker_dot_tests"("id", "business_unit_id", "organization_id", "worker_id", "test_type", "substance", "status", "result", "collected_at", "result_at", "notes", "created_at", "updated_at")
SELECT
    'wdot_' || upper(substr(md5(wp.worker_id || 'MIGRATED'), 1, 26)),
    wp.business_unit_id,
    wp.organization_id,
    wp.worker_id,
    'PreEmployment',
    'Drug',
    'Completed',
    'Negative',
    wp.last_drug_test,
    wp.last_drug_test,
    'Migrated from worker_profiles.last_drug_test. The profile field recorded the last drug test on file and compliance already treated it as passed.',
    wp.created_at,
    wp.updated_at
FROM
    "worker_profiles" wp
WHERE
    wp.last_drug_test > 0
ON CONFLICT
    DO NOTHING;

--bun:split
UPDATE
    "worker_profiles"
SET
    "drug_alcohol_status" = 'Clear'
WHERE
    "last_drug_test" > 0;
