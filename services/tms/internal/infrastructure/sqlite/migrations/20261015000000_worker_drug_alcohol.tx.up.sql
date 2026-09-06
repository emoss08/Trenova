-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261015000000_worker_drug_alcohol.tx.up.sql

CREATE TABLE IF NOT EXISTS "dot_random_pools"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "code" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "period" TEXT NOT NULL DEFAULT 'Quarterly',
    "drug_rate_percent" INTEGER NOT NULL DEFAULT 50,
    "alcohol_rate_percent" INTEGER NOT NULL DEFAULT 10,
    "included_driver_types" TEXT NOT NULL DEFAULT '[]',
    "is_default" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_dot_random_pools" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_dot_random_pools_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_dot_random_pools_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_dot_random_pools_drug_rate" CHECK ("drug_rate_percent" >= 0 AND "drug_rate_percent" <= 100),
    CONSTRAINT "chk_dot_random_pools_alcohol_rate" CHECK ("alcohol_rate_percent" >= 0 AND "alcohol_rate_percent" <= 100)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_dot_random_pools_code" ON "dot_random_pools" ("organization_id", "business_unit_id", LOWER("code"));

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_dot_random_pools_default" ON "dot_random_pools" ("organization_id", "business_unit_id")WHERE
    "is_default";

--bun:split

CREATE TABLE IF NOT EXISTS "dot_random_draws"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "pool_id" TEXT NOT NULL,
    "period_key" TEXT NOT NULL,
    "period_start" INTEGER NOT NULL,
    "period_end" INTEGER NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Draft',
    "pool_size" INTEGER NOT NULL DEFAULT 0,
    "drug_target" INTEGER NOT NULL DEFAULT 0,
    "alcohol_target" INTEGER NOT NULL DEFAULT 0,
    "drug_selected" INTEGER NOT NULL DEFAULT 0,
    "alcohol_selected" INTEGER NOT NULL DEFAULT 0,
    "seed" TEXT NOT NULL,
    "method" TEXT NOT NULL,
    "notes" TEXT,
    "drawn_at" INTEGER NOT NULL,
    "drawn_by_id" TEXT,
    "finalized_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_dot_random_draws" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_dot_random_draws_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_dot_random_draws_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_dot_random_draws_pool" FOREIGN KEY ("pool_id", "organization_id", "business_unit_id") REFERENCES "dot_random_pools"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_dot_random_draws_period" CHECK ("period_end" > "period_start"),
    CONSTRAINT "chk_dot_random_draws_final" CHECK (("status" = 'Final') = ("finalized_at" IS NOT NULL))
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_dot_random_draws_period" ON "dot_random_draws" ("organization_id", "business_unit_id", "pool_id", "period_key")WHERE
    "status" <> 'Cancelled';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_dot_random_draws_drawn" ON "dot_random_draws" ("organization_id", "business_unit_id", "drawn_at" DESC);

--bun:split

CREATE TABLE IF NOT EXISTS "dot_random_draw_entries"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "draw_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "substance" TEXT NOT NULL,
    "rank" INTEGER NOT NULL DEFAULT 0,
    "status" TEXT NOT NULL DEFAULT 'Selected',
    "notified_at" INTEGER,
    "completed_at" INTEGER,
    "test_id" TEXT,
    "excuse_reason" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_dot_random_draw_entries" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_dot_random_draw_entries_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_dot_random_draw_entries_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_dot_random_draw_entries_draw" FOREIGN KEY ("draw_id", "organization_id", "business_unit_id") REFERENCES "dot_random_draws"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_dot_random_draw_entries_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_dot_random_draw_entries_excuse" CHECK (("status" <> 'Excused') OR ("excuse_reason" IS NOT NULL))
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_dot_random_draw_entries_worker" ON "dot_random_draw_entries" ("organization_id", "business_unit_id", "draw_id", "worker_id", "substance");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_dot_random_draw_entries_worker" ON "dot_random_draw_entries" ("worker_id", "status");

--bun:split

CREATE TABLE IF NOT EXISTS "worker_dot_tests"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "test_type" TEXT NOT NULL,
    "substance" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Scheduled',
    "result" TEXT NOT NULL DEFAULT 'Pending',
    "is_dot" INTEGER NOT NULL DEFAULT 1,
    "reason" TEXT,
    "scheduled_at" INTEGER,
    "collected_at" INTEGER,
    "result_at" INTEGER,
    "collection_site" TEXT,
    "collector_name" TEXT,
    "specimen_id" TEXT,
    "lab_name" TEXT,
    "mro_name" TEXT,
    "mro_verified_at" INTEGER,
    "alcohol_concentration" REAL,
    "safety_event_id" TEXT,
    "draw_entry_id" TEXT,
    "document_id" TEXT,
    "notes" TEXT,
    "ordered_by_id" TEXT,
    "recorded_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
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

CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_dot_tests_draw_entry" ON "worker_dot_tests" ("organization_id", "business_unit_id", "draw_entry_id")WHERE
    "draw_entry_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_dot_tests_worker" ON "worker_dot_tests" ("worker_id", "collected_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_dot_tests_open" ON "worker_dot_tests" ("organization_id", "business_unit_id", "status")WHERE
    "status" IN ('Scheduled', 'Collected', 'AwaitingResult');

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_dot_tests_type" ON "worker_dot_tests" ("organization_id", "business_unit_id", "test_type", "collected_at" DESC);

--bun:split

CREATE TABLE IF NOT EXISTS "worker_dot_violations"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "violation_type" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Open',
    "occurred_at" INTEGER NOT NULL,
    "source_test_id" TEXT,
    "reported_to_clearinghouse_at" INTEGER,
    "sap_name" TEXT,
    "sap_referred_at" INTEGER,
    "sap_evaluation_completed_at" INTEGER,
    "rtd_test_id" TEXT,
    "rtd_completed_at" INTEGER,
    "follow_up_test_count" INTEGER NOT NULL DEFAULT 0,
    "follow_up_tests_completed" INTEGER NOT NULL DEFAULT 0,
    "follow_up_ends_at" INTEGER,
    "resolved_at" INTEGER,
    "document_id" TEXT,
    "notes" TEXT,
    "recorded_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
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

CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_dot_violations_open" ON "worker_dot_violations" ("organization_id", "business_unit_id", "worker_id")WHERE
    "status" <> 'Resolved';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_dot_violations_worker" ON "worker_dot_violations" ("worker_id", "occurred_at" DESC);

--bun:split

CREATE TABLE IF NOT EXISTS "worker_clearinghouse_queries"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "query_type" TEXT NOT NULL,
    "result" TEXT NOT NULL DEFAULT 'Pending',
    "consent_obtained_at" INTEGER,
    "consent_expires_at" INTEGER,
    "requested_at" INTEGER NOT NULL,
    "completed_at" INTEGER,
    "violation_count" INTEGER NOT NULL DEFAULT 0,
    "reference" TEXT,
    "document_id" TEXT,
    "notes" TEXT,
    "performed_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_worker_clearinghouse_queries" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_clearinghouse_queries_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_clearinghouse_queries_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_clearinghouse_queries_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_clearinghouse_queries_document" FOREIGN KEY ("document_id", "organization_id", "business_unit_id") REFERENCES "documents"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_worker_clearinghouse_queries_completed" CHECK (("result" = 'Pending') = ("completed_at" IS NULL)),
    CONSTRAINT "chk_worker_clearinghouse_queries_violations" CHECK ("violation_count" >= 0 AND ("result" <> 'ViolationsFound' OR "violation_count" > 0))
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_clearinghouse_queries_worker" ON "worker_clearinghouse_queries" ("worker_id", "requested_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_clearinghouse_queries_pending" ON "worker_clearinghouse_queries" ("organization_id", "business_unit_id", "requested_at")WHERE
    "result" = 'Pending';

--bun:split

ALTER TABLE "worker_profiles" ADD COLUMN "drug_alcohol_status" TEXT NOT NULL DEFAULT 'Unknown';

--bun:split

ALTER TABLE "worker_profiles" ADD COLUMN "return_to_duty_status" TEXT NOT NULL DEFAULT 'NotRequired';

--bun:split

ALTER TABLE "worker_profiles" ADD COLUMN "last_clearinghouse_query_at" INTEGER;

--bun:split

ALTER TABLE "worker_profiles" ADD COLUMN "next_clearinghouse_query_due" INTEGER;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_profiles_drug_alcohol" ON "worker_profiles" ("organization_id", "business_unit_id", "drug_alcohol_status");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_profiles_clearinghouse_due" ON "worker_profiles" ("next_clearinghouse_query_due")WHERE
    "next_clearinghouse_query_due" IS NOT NULL;

--bun:split

UPDATE
    "worker_profiles"
SET
    "drug_alcohol_status" = 'Clear'
WHERE
    "last_drug_test" > 0;
